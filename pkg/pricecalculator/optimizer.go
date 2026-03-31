package pricecalculator

import (
	"math"
	"sort"
)

// optimizePrice calculates the minimum total price needed to cover
// at least the requested duration using the available pricing periods.
//
// Rules:
// - periods can be reused unlimited times
// - result must have the minimum possible TotalPrice
// - CoveredMinutes can be equal to or greater than requiredMinutes
// - if multiple combinations exist, return any one with the same minimum total price
// - if no valid solution exists, return ErrNoSolution
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
		return CalculateResult{}, ErrNoSolution
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
		return CalculateResult{}, ErrNoSolution
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
		return CalculateResult{}, ErrNoSolution
	}

	breakdown := buildBreakdown(bestFullMinutes, prev, used, periods)
	if useProration {
		breakdown = append(breakdown, BreakdownItem{
			Id:              minimumPeriod.Id,
			DurationMinutes: minimumPeriod.DurationMinutes,
			Price:           minimumPeriod.Price,
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
		return PricingPeriod{}, ErrNoSolution
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
				Price:           minimumPeriod.Price,
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
				Price:           minimumPeriod.Price,
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
			Price:           period.Price,
			Quantity:        quantity,
		})
	}

	sortBreakdown(breakdown)

	return breakdown
}

func mergeBreakdownItems(breakdown []BreakdownItem) []BreakdownItem {
	merged := make([]BreakdownItem, 0, len(breakdown))
	indexByPeriod := make(map[[2]int64]int, len(breakdown))

	for _, item := range breakdown {
		key := [2]int64{int64(item.DurationMinutes), item.Price}
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

		return breakdown[i].Quantity > breakdown[j].Quantity
	})
}
