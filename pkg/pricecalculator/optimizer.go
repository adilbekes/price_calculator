package pricecalculator

import (
	"math"
	"sort"
	"time"
)

// optimizePrice calculates the minimum total price needed to cover
// at least the requested duration using the available pricing periods.
//
// Rules:
// - periods can be reused unlimited times
// - result must have the minimum possible TotalPrice
// - CoveredMinutes can be equal to or greater than requiredMinutes
// - if multiple combinations exist, return any one with the same minimum total price
// - always returns a deterministic fallback solution when DP cannot produce one
//
// Notes:
// - requiredMinutes is already validated
// - periods are already validated
// - periods may be unsorted
func optimizePrice(
	requiredMinutes int,
	periods []PricingPeriod,
) (CalculateResult, error) {
	maxPeriodDuration := maxDuration(periods)
	searchLimit := requiredMinutes + maxPeriodDuration

	const inf int64 = math.MaxInt64

	dp := make([]int64, searchLimit+1)
	prev := make([]int, searchLimit+1)
	used := make([]int, searchLimit+1)

	for i := range dp {
		dp[i] = inf
		prev[i] = -1
		used[i] = -1
	}

	dp[0] = 0

	for minutes := 1; minutes <= searchLimit; minutes++ {
		for i, period := range periods {
			if minutes < period.DurationMinutes {
				continue
			}

			previousMinutes := minutes - period.DurationMinutes
			if dp[previousMinutes] == inf {
				continue
			}

			candidatePrice := dp[previousMinutes] + period.Price
			if candidatePrice < dp[minutes] {
				dp[minutes] = candidatePrice
				prev[minutes] = previousMinutes
				used[minutes] = i
			}
		}
	}

	bestMinutes := -1
	bestPrice := inf

	for minutes := requiredMinutes; minutes <= searchLimit; minutes++ {
		if dp[minutes] == inf {
			continue
		}

		if dp[minutes] < bestPrice {
			bestPrice = dp[minutes]
			bestMinutes = minutes
		}
	}

	if bestMinutes == -1 {
		return fallbackSolution(requiredMinutes, periods), nil
	}

	breakdown := buildBreakdown(bestMinutes, prev, used, periods)

	return CalculateResult{
		TotalPrice:     bestPrice,
		CoveredMinutes: bestMinutes,
		Breakdown:      breakdown,
	}, nil
}

func optimizePriceWithOptionalProration(
	requiredMinutes int,
	periods []PricingPeriod,
) (CalculateResult, error) {
	fullCoverageResult, fullCoverageErr := optimizePrice(requiredMinutes, periods)
	exactWithProrationResult, exactWithProrationErr := optimizePriceExactWithProration(requiredMinutes, periods)

	switch {
	case fullCoverageErr == nil && exactWithProrationErr == nil:
		if exactWithProrationResult.TotalPrice < fullCoverageResult.TotalPrice {
			return exactWithProrationResult, nil
		}

		if exactWithProrationResult.TotalPrice == fullCoverageResult.TotalPrice && exactWithProrationResult.CoveredMinutes < fullCoverageResult.CoveredMinutes {
			return exactWithProrationResult, nil
		}

		return fullCoverageResult, nil
	case fullCoverageErr == nil:
		return fullCoverageResult, nil
	case exactWithProrationErr == nil:
		return exactWithProrationResult, nil
	default:
		return fallbackSolution(requiredMinutes, periods), nil
	}
}

func optimizePriceExactWithProration(
	requiredMinutes int,
	periods []PricingPeriod,
) (CalculateResult, error) {
	const inf int64 = math.MaxInt64

	minimumDuration := minDuration(periods)
	minimumPeriod, err := cheapestPeriodByDuration(periods, minimumDuration)
	if err != nil {
		return CalculateResult{}, err
	}

	dp := make([]int64, requiredMinutes+1)
	prev := make([]int, requiredMinutes+1)
	used := make([]int, requiredMinutes+1)

	for i := range dp {
		dp[i] = inf
		prev[i] = -1
		used[i] = -1
	}

	dp[0] = 0

	for minutes := 1; minutes <= requiredMinutes; minutes++ {
		for i, period := range periods {
			if minutes < period.DurationMinutes {
				continue
			}

			previousMinutes := minutes - period.DurationMinutes
			if dp[previousMinutes] == inf {
				continue
			}

			candidatePrice := dp[previousMinutes] + period.Price
			if candidatePrice < dp[minutes] {
				dp[minutes] = candidatePrice
				prev[minutes] = previousMinutes
				used[minutes] = i
			}
		}
	}

	bestPrice := inf
	bestFullMinutes := -1
	useProration := false

	if dp[requiredMinutes] != inf {
		bestPrice = dp[requiredMinutes]
		bestFullMinutes = requiredMinutes
	}

	for fullMinutes := 0; fullMinutes < requiredMinutes; fullMinutes++ {
		if dp[fullMinutes] == inf {
			continue
		}

		remainderMinutes := requiredMinutes - fullMinutes
		if remainderMinutes > minimumPeriod.DurationMinutes {
			continue
		}

		candidatePrice := dp[fullMinutes] + calculateProratedPrice(minimumPeriod, remainderMinutes)
		if candidatePrice < bestPrice {
			bestPrice = candidatePrice
			bestFullMinutes = fullMinutes
			useProration = true
		}
	}

	if bestFullMinutes == -1 {
		return fallbackSolution(requiredMinutes, periods), nil
	}

	breakdown := buildBreakdown(bestFullMinutes, prev, used, periods)
	if useProration {
		// Calculate the remainder minutes that will be used from the prorated period
		remainderMinutes := requiredMinutes - bestFullMinutes
		breakdown = append(breakdown, BreakdownItem{
			Id:              minimumPeriod.Id,
			DurationMinutes: minimumPeriod.DurationMinutes,
			UsedDuration:    remainderMinutes,
			Price:           minimumPeriod.Price,
			UsedPrice:       calculateProratedPrice(minimumPeriod, remainderMinutes),
			Quantity:        1,
		})
		breakdown = mergeBreakdownItems(breakdown)
		sortBreakdown(breakdown)
	}

	return CalculateResult{
		TotalPrice:     bestPrice,
		CoveredMinutes: requiredMinutes,
		Breakdown:      breakdown,
	}, nil
}

// optimizeTimelineAware fills the requested duration timeline sequentially,
// selecting the best available period at each step based on availability windows.
// This respects time-based period switches (e.g., cheaper period available after 22:00).
// Prorating is only applied in ProrateMinimum, ProrateAny, and RoundUpMinimumAndProrateAny modes.
func optimizeTimelineAware(
	requiredMinutes int,
	periods []PricingPeriod,
	startTime time.Time,
	mode PricingMode,
) (CalculateResult, error) {
	if len(periods) == 0 {
		return CalculateResult{}, NewRequestError("no periods available")
	}

	// Determine if prorating is allowed in this mode
	allowProrating := mode == PricingModeProrateMinimum || mode == PricingModeProrateAny || mode == PricingModeRoundUpMinimumAndProrateAny

	currentTime := startTime
	remainingMinutes := requiredMinutes
	timeline := make([]BreakdownItem, 0)
	totalPrice := int64(0)
	coveredMinutes := 0

	for remainingMinutes > 0 {
		type timelineCandidate struct {
			index     int
			effective int
		}

		candidates := make([]timelineCandidate, 0, len(periods))
		hasRestrictedCandidate := false

		for i, period := range periods {
			available, err := isPeriodAvailableAtTime(period, currentTime)
			if err != nil || !available {
				continue
			}

			windowRem := periodWindowRemainingMinutes(period, currentTime)
			effective := windowRem
			if effective > remainingMinutes {
				effective = remainingMinutes
			}
			if effective <= 0 {
				continue
			}

			if period.StartTime != "" || len(period.Availability) > 0 {
				hasRestrictedCandidate = true
			}

			candidates = append(candidates, timelineCandidate{index: i, effective: effective})
		}

		// Select the period with the lowest effective price-per-minute.
		// Effective minutes = min(window remaining, remaining request minutes).
		// Using cross-multiplication: p_a/e_a < p_b/e_b ⟺ p_a*e_b < p_b*e_a
		bestPeriod := -1
		bestEffective := 0

		for _, candidate := range candidates {
			i := candidate.index
			period := periods[i]
			effective := candidate.effective

			if hasRestrictedCandidate && period.StartTime == "" && len(period.Availability) == 0 && period.DurationMinutes > remainingMinutes {
				continue
			}

			if bestPeriod == -1 {
				bestPeriod = i
				bestEffective = effective
				continue
			}

			best := periods[bestPeriod]
			// period cheaper per minute when: period.Price * bestEffective < best.Price * effective
			if period.Price*int64(bestEffective) < best.Price*int64(effective) {
				bestPeriod = i
				bestEffective = effective
			}
		}

		if bestPeriod == -1 {
			// No period available — fallback to the cheapest by absolute price
			for i, period := range periods {
				if bestPeriod == -1 || period.Price < periods[bestPeriod].Price {
					bestPeriod = i
				}
			}
			if bestPeriod == -1 {
				break
			}
			bestEffective = periodWindowRemainingMinutes(periods[bestPeriod], currentTime)
			if bestEffective > remainingMinutes {
				bestEffective = remainingMinutes
			}
		}

		// Use the best available period
		period := periods[bestPeriod]

		// usedMinutes starts at the period's window-capped effective coverage.
		// In non-prorating modes this is how many minutes of this period we consume.
		// In prorating modes it may be further reduced when a cheaper period arrives.
		usedMinutes := bestEffective
		if usedMinutes > period.DurationMinutes {
			usedMinutes = period.DurationMinutes
		}
		price := period.Price

		// If the period has a start_time (fixed daily window), cap usedMinutes to
		// the minutes remaining in the window starting from currentTime.
		if period.StartTime != "" {
			windowRemaining := periodWindowRemainingMinutes(period, currentTime)
			if windowRemaining < usedMinutes {
				usedMinutes = windowRemaining
			}
		}

		// Only check for cheaper periods and prorate if prorating is allowed
		if allowProrating {
			// Check if a cheaper period becomes available within the next period duration
			// If so, calculate how much time until it becomes available and use the current period only for that
			timeUntilNextChange := period.DurationMinutes
			for i, otherPeriod := range periods {
				if i == bestPeriod || otherPeriod.Price >= period.Price {
					continue // Skip current period and more expensive ones
				}

				// Check future times to see when this period becomes available
				for futureMinute := 1; futureMinute <= period.DurationMinutes && futureMinute < timeUntilNextChange; futureMinute++ {
					futureTime := currentTime.Add(time.Duration(futureMinute) * time.Minute)
					available, err := isPeriodAvailableAtTime(otherPeriod, futureTime)
					if err == nil && available {
						timeUntilNextChange = futureMinute
						break
					}
				}
			}

			// Use period for the calculated time (possibly prorated)
			usedMinutes = timeUntilNextChange
			if usedMinutes > remainingMinutes {
				usedMinutes = remainingMinutes
			}

			// Calculate prorated price if using less than full duration
			price = period.Price
			if usedMinutes < period.DurationMinutes {
				price = calculateProratedPrice(period, usedMinutes)
			}
		}

		// In non-prorating modes, breakdown reflects charged full period values.
		// In prorating modes it may be further reduced when a cheaper period arrives.
		breakdownUsedDuration := period.DurationMinutes
		breakdownUsedPrice := period.Price
		if allowProrating && usedMinutes < period.DurationMinutes {
			breakdownUsedDuration = usedMinutes
			breakdownUsedPrice = price
		}

		timeline = append(timeline, BreakdownItem{
			Id:              period.Id,
			DurationMinutes: period.DurationMinutes,
			UsedDuration:    breakdownUsedDuration,
			Price:           period.Price,
			UsedPrice:       breakdownUsedPrice,
			Quantity:        1,
			StartTime:       formatPeriodWindowStartIfPresent(period, currentTime),
			EndTime:         formatPeriodWindowEndIfPresent(period, currentTime),
		})

		remainingMinutes -= usedMinutes
		coveredMinutes += usedMinutes
		totalPrice += price
		currentTime = currentTime.Add(time.Duration(usedMinutes) * time.Minute)
	}

	// Merge breakdown items with same period
	mergedTimeline := mergeBreakdownItems(timeline)
	sortBreakdown(mergedTimeline)

	return CalculateResult{
		TotalPrice:     totalPrice,
		CoveredMinutes: coveredMinutes,
		Breakdown:      mergedTimeline,
	}, nil
}

func maxDuration(periods []PricingPeriod) int {
	maxMinutes := 0

	for _, period := range periods {
		if period.DurationMinutes > maxMinutes {
			maxMinutes = period.DurationMinutes
		}
	}

	return maxMinutes
}

func minDuration(periods []PricingPeriod) int {
	minMinutes := 0

	for _, period := range periods {
		if minMinutes == 0 || period.DurationMinutes < minMinutes {
			minMinutes = period.DurationMinutes
		}
	}

	return minMinutes
}

func cheapestPeriodByDuration(periods []PricingPeriod, duration int) (PricingPeriod, error) {
	var selected PricingPeriod
	found := false

	for _, period := range periods {
		if period.DurationMinutes != duration {
			continue
		}

		if !found || period.Price < selected.Price {
			selected = period
			found = true
		}
	}

	if !found {
		return PricingPeriod{}, NewRequestError("no pricing periods available for duration %d", duration)
	}

	return selected, nil
}

func priceBelowMinimum(req CalculateRequest) (CalculateResult, error) {
	minimumDuration := minDuration(req.Periods)
	minimumPeriod, err := cheapestPeriodByDuration(req.Periods, minimumDuration)
	if err != nil {
		return CalculateResult{}, err
	}

	switch req.PricingMode {
	case PricingModeRoundUp:
		return CalculateResult{
			TotalPrice:     minimumPeriod.Price,
			CoveredMinutes: minimumPeriod.DurationMinutes,
			Breakdown: []BreakdownItem{{
				Id:              minimumPeriod.Id,
				DurationMinutes: minimumPeriod.DurationMinutes,
				UsedDuration:    minimumPeriod.DurationMinutes,
				Price:           minimumPeriod.Price,
				UsedPrice:       minimumPeriod.Price,
				Quantity:        1,
			}},
		}, nil
	case PricingModeProrateMinimum:
		price := calculateProratedPrice(minimumPeriod, req.RequestedDurationMinutes)

		return CalculateResult{
			TotalPrice:     price,
			CoveredMinutes: req.RequestedDurationMinutes,
			Breakdown: []BreakdownItem{{
				Id:              minimumPeriod.Id,
				DurationMinutes: minimumPeriod.DurationMinutes,
				UsedDuration:    req.RequestedDurationMinutes,
				Price:           minimumPeriod.Price,
				UsedPrice:       price,
				Quantity:        1,
			}},
		}, nil
	default:
		return CalculateResult{}, ErrInvalidRequest
	}
}

func ceilDivInt64(numerator, denominator int64) int64 {
	return (numerator + denominator - 1) / denominator
}

func calculateProratedPrice(period PricingPeriod, durationMinutes int) int64 {
	return ceilDivInt64(period.Price*int64(durationMinutes), int64(period.DurationMinutes))
}

func fallbackSolution(requiredMinutes int, periods []PricingPeriod) CalculateResult {
	best := periods[0]
	for _, period := range periods[1:] {
		// Compare by effective price per minute using cross multiplication.
		if period.Price*int64(best.DurationMinutes) < best.Price*int64(period.DurationMinutes) {
			best = period
		}
	}

	quantity := (requiredMinutes + best.DurationMinutes - 1) / best.DurationMinutes
	covered := quantity * best.DurationMinutes
	total := int64(quantity) * best.Price

	return CalculateResult{
		TotalPrice:     total,
		CoveredMinutes: covered,
		Breakdown: []BreakdownItem{{
			Id:              best.Id,
			DurationMinutes: best.DurationMinutes,
			UsedDuration:    best.DurationMinutes,
			Price:           best.Price,
			UsedPrice:       best.Price,
			Quantity:        quantity,
		}},
	}
}

func buildBreakdown(
	totalMinutes int,
	prev []int,
	used []int,
	periods []PricingPeriod,
) []BreakdownItem {
	quantities := make(map[int]int)

	for totalMinutes > 0 {
		periodIndex := used[totalMinutes]
		if periodIndex < 0 {
			break
		}

		quantities[periodIndex]++
		totalMinutes = prev[totalMinutes]
	}

	breakdown := make([]BreakdownItem, 0, len(quantities))
	for i, quantity := range quantities {
		period := periods[i]
		breakdown = append(breakdown, BreakdownItem{
			Id:              period.Id,
			DurationMinutes: period.DurationMinutes,
			UsedDuration:    period.DurationMinutes,
			Price:           period.Price,
			UsedPrice:       period.Price,
			Quantity:        quantity,
		})
	}

	sortBreakdown(breakdown)

	return breakdown
}

func mergeBreakdownItems(breakdown []BreakdownItem) []BreakdownItem {
	merged := make([]BreakdownItem, 0, len(breakdown))
	indexByPeriod := make(map[[4]int64]int, len(breakdown))

	for _, item := range breakdown {
		// Merge only rows with the same pricing period AND the same usage profile.
		// This keeps full-use items (used==duration) separate from prorated items.
		key := [4]int64{int64(item.DurationMinutes), item.Price, int64(item.UsedDuration), item.UsedPrice}
		if index, exists := indexByPeriod[key]; exists {
			merged[index].Quantity += item.Quantity
			continue
		}

		indexByPeriod[key] = len(merged)
		merged = append(merged, item)
	}

	return merged
}

func sortBreakdown(breakdown []BreakdownItem) {
	sort.Slice(breakdown, func(i, j int) bool {
		if breakdown[i].DurationMinutes != breakdown[j].DurationMinutes {
			return breakdown[i].DurationMinutes > breakdown[j].DurationMinutes
		}

		if breakdown[i].Price != breakdown[j].Price {
			return breakdown[i].Price > breakdown[j].Price
		}

		iFull := breakdown[i].UsedDuration == breakdown[i].DurationMinutes
		jFull := breakdown[j].UsedDuration == breakdown[j].DurationMinutes
		if iFull != jFull {
			return iFull
		}

		if breakdown[i].UsedDuration != breakdown[j].UsedDuration {
			return breakdown[i].UsedDuration > breakdown[j].UsedDuration
		}

		return breakdown[i].Quantity > breakdown[j].Quantity
	})
}

func periodWindowBoundsForReferenceDay(period PricingPeriod, reference time.Time) (time.Time, time.Time, bool) {
	if period.StartTime == "" {
		return time.Time{}, time.Time{}, false
	}

	hour, minute, err := parseTimeHHMM(period.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}

	loc := timeLocation(reference)
	windowStart := time.Date(reference.Year(), reference.Month(), reference.Day(), hour, minute, 0, 0, loc)
	windowEnd := windowStart.Add(time.Duration(period.DurationMinutes) * time.Minute)

	return windowStart, windowEnd, true
}

func formatPeriodWindowStartIfPresent(period PricingPeriod, reference time.Time) string {
	windowStart, _, ok := periodWindowBoundsForReferenceDay(period, reference)
	if !ok {
		return ""
	}
	return windowStart.Format(time.DateTime)
}

func formatPeriodWindowEndIfPresent(period PricingPeriod, reference time.Time) string {
	_, windowEnd, ok := periodWindowBoundsForReferenceDay(period, reference)
	if !ok {
		return ""
	}
	return windowEnd.Format(time.DateTime)
}

// periodWindowRemainingMinutes returns how many minutes remain in the period's daily window
// starting from currentTime. When the period has no start_time it returns the full DurationMinutes.
func periodWindowRemainingMinutes(period PricingPeriod, currentTime time.Time) int {
	if period.StartTime == "" {
		return period.DurationMinutes
	}
	hour, minute, err := parseTimeHHMM(period.StartTime)
	if err != nil {
		return period.DurationMinutes
	}
	loc := timeLocation(currentTime)
	windowStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), hour, minute, 0, 0, loc)
	windowEnd := windowStart.Add(time.Duration(period.DurationMinutes) * time.Minute)
	if !windowEnd.After(currentTime) {
		return 0
	}
	return int(windowEnd.Sub(currentTime).Minutes())
}
