package pricecalculator

import "time"

type Calculator interface {
	Calculate(req CalculateRequest) (CalculateResult, error)
}

type calculator struct{}

func NewCalculator() Calculator {
	return &calculator{}
}

func (c *calculator) Calculate(req CalculateRequest) (CalculateResult, error) {
	// If start_time is provided, validate and parse it; otherwise use current time.
	effectiveStartTime, parseErr := getEffectiveStartTime(req.StartTime)
	if parseErr != nil {
		return CalculateResult{}, parseErr
	}

	if err := validateStartTime(effectiveStartTime); err != nil {
		return CalculateResult{}, err
	}

	// Store the original input for later
	originalStartTime := req.StartTime

	if err := validateRequest(req); err != nil {
		return CalculateResult{}, err
	}

	normalizedDuration := normalizeDuration(
		req.RequestedDurationMinutes,
		effectiveRequestedDurationStepMinutes(req.RequestedDurationStepMinutes),
	)
	intervalStart, intervalEnd := requestInterval(effectiveStartTime, normalizedDuration)

	availablePeriods := make([]PricingPeriod, 0, len(req.Periods))
	startAvailablePeriods := make([]PricingPeriod, 0, len(req.Periods))
	// hasTimeBasedRestrictions is true when any period uses start_time or a
	// time-range string in its availability map. In that case the timeline-aware
	// optimizer is required. Boolean per-date availability (true/false) can be
	// handled by pre-filtering and then using the cheaper DP optimizer.
	hasTimeBasedRestrictions := false
	for _, period := range req.Periods {
		if period.StartTime != "" {
			hasTimeBasedRestrictions = true
		}
		for _, v := range period.Availability {
			if _, isStr := v.(string); isStr {
				hasTimeBasedRestrictions = true
				break
			}
		}

		startAvailable, err := isPeriodAvailableAtTime(period, effectiveStartTime)
		if err != nil {
			return CalculateResult{}, err
		}
		if startAvailable {
			startAvailablePeriods = append(startAvailablePeriods, period)
		}

		available, err := isPeriodAvailableForInterval(period, intervalStart, intervalEnd)
		if err != nil {
			return CalculateResult{}, err
		}
		if available {
			availablePeriods = append(availablePeriods, period)
		}
	}

	if len(availablePeriods) == 0 {
		// Always provide a pricing result. If no period can cover the whole interval,
		// prefer periods that are available at request start; otherwise use full catalog.
		if len(startAvailablePeriods) > 0 {
			availablePeriods = startAvailablePeriods
		} else {
			availablePeriods = req.Periods
		}
	}

	normalizedReq := req
	normalizedReq.Periods = availablePeriods
	normalizedReq.RequestedDurationMinutes = normalizedDuration
	// timelinePeriods is the full catalog when time-based restrictions are present,
	// so the optimizer can switch between periods as their windows open/close.
	timelinePeriods := normalizedReq.Periods
	if hasTimeBasedRestrictions {
		timelinePeriods = req.Periods
		if normalizedReq.PricingMode == PricingModeRoundUp {
			trimmedPeriods := make([]PricingPeriod, 0, len(timelinePeriods))
			for _, period := range timelinePeriods {
				isOversizedUnrestricted := period.StartTime == "" && len(period.Availability) == 0 && period.DurationMinutes > normalizedReq.RequestedDurationMinutes
				if isOversizedUnrestricted {
					continue
				}
				trimmedPeriods = append(trimmedPeriods, period)
			}
			if len(trimmedPeriods) > 0 {
				timelinePeriods = trimmedPeriods
			}
		}
	}

	// If there's a single period that covers the entire request, pick the cheapest one
	// instead of using timeline-aware optimization
	// Only use single-period optimization for RoundUp mode when a single full period covers the request
	// and the period doesn't have time window constraints
	if normalizedReq.PricingMode == PricingModeRoundUp && !hasTimeBasedRestrictions {
		var singleCoveringPeriod *PricingPeriod
		var cheapestPrice int64
		for i, period := range timelinePeriods {
			if period.StartTime != "" {
				// Skip periods with time windows - let timeline optimizer handle them
				continue
			}

			// Check if period can cover request without time window constraints
			if period.DurationMinutes < normalizedDuration {
				continue
			}

			// Verify period is available at the request start time
			available, err := isPeriodAvailableAtTime(period, effectiveStartTime)
			if err != nil {
				continue
			}
			if !available {
				continue
			}
			if singleCoveringPeriod == nil || period.Price < cheapestPrice {
				singleCoveringPeriod = &timelinePeriods[i]
				cheapestPrice = period.Price
			}
		}
		if singleCoveringPeriod != nil {
			breakdownItem := BreakdownItem{
				Id:              singleCoveringPeriod.Id,
				DurationMinutes: singleCoveringPeriod.DurationMinutes,
				UsedDuration:    singleCoveringPeriod.DurationMinutes,
				Price:           singleCoveringPeriod.Price,
				UsedPrice:       singleCoveringPeriod.Price,
				Quantity:        1,
			}
			if singleCoveringPeriod.StartTime != "" {
				breakdownItem.StartTime = effectiveStartTime.Format(time.DateTime)
				endTime := effectiveStartTime.Add(time.Duration(singleCoveringPeriod.DurationMinutes) * time.Minute)
				breakdownItem.EndTime = endTime.Format(time.DateTime)
			}
			return CalculateResult{
				StartTime:      originalStartTime,
				EndTime:        effectiveStartTime.Add(time.Duration(singleCoveringPeriod.DurationMinutes) * time.Minute).Format(time.DateTime),
				TotalPrice:     singleCoveringPeriod.Price,
				CoveredMinutes: singleCoveringPeriod.DurationMinutes,
				Breakdown:      []BreakdownItem{breakdownItem},
			}, nil
		}
	}

	minimumDurationPeriods := normalizedReq.Periods
	if hasTimeBasedRestrictions {
		minimumDurationPeriods = timelinePeriods
	}
	minimumDuration := minDuration(minimumDurationPeriods)

	var result CalculateResult
	var err error

	if normalizedReq.PricingMode == PricingModeRoundUpMinimumAndProrateAny {
		if normalizedReq.RequestedDurationMinutes < minimumDuration {
			result, err = priceBelowMinimum(CalculateRequest{
				RequestedDurationMinutes: normalizedReq.RequestedDurationMinutes,
				Periods:                  normalizedReq.Periods,
				PricingMode:              PricingModeRoundUp,
			})
		} else {
			if hasTimeBasedRestrictions {
				result, err = optimizeTimelineAware(normalizedReq.RequestedDurationMinutes, timelinePeriods, effectiveStartTime, normalizedReq.PricingMode)
			} else {
				result, err = optimizePriceWithOptionalProration(normalizedReq.RequestedDurationMinutes, normalizedReq.Periods)
			}
		}
	} else if normalizedReq.PricingMode == PricingModeProrateAny {
		if hasTimeBasedRestrictions {
			result, err = optimizeTimelineAware(normalizedReq.RequestedDurationMinutes, timelinePeriods, effectiveStartTime, normalizedReq.PricingMode)
		} else {
			result, err = optimizePriceWithOptionalProration(normalizedReq.RequestedDurationMinutes, normalizedReq.Periods)
		}
	} else if normalizedReq.RequestedDurationMinutes < minimumDuration {
		result, err = priceBelowMinimum(normalizedReq)
	} else {
		if hasTimeBasedRestrictions {
			result, err = optimizeTimelineAware(normalizedReq.RequestedDurationMinutes, timelinePeriods, effectiveStartTime, normalizedReq.PricingMode)
		} else {
			result, err = optimizePrice(normalizedReq.RequestedDurationMinutes, normalizedReq.Periods)
		}
	}

	if err != nil {
		return CalculateResult{}, err
	}

	for i := range result.Breakdown {
		if result.Breakdown[i].UsedDuration <= 0 {
			result.Breakdown[i].UsedDuration = result.Breakdown[i].DurationMinutes
		}
		if result.Breakdown[i].UsedPrice <= 0 {
			result.Breakdown[i].UsedPrice = result.Breakdown[i].Price
		}
	}

	result.TotalPrice = roundUpPrice(result.TotalPrice, effectiveTotalPriceStep(req.TotalPriceStep))

	// If start_time was provided in the original request, include it and calculate end_time.
	if originalStartTime != "" {
		result.StartTime = originalStartTime
		endTime := effectiveStartTime.Add(time.Duration(result.CoveredMinutes) * time.Minute)
		result.EndTime = endTime.Format(time.DateTime)
	}

	return result, nil
}
