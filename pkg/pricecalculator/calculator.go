package pricecalculator

type Calculator interface {
	Calculate(req CalculateRequest) (CalculateResult, error)
}

type calculator struct{}

func NewCalculator() Calculator {
	return &calculator{}
}

func (c *calculator) Calculate(req CalculateRequest) (CalculateResult, error) {
	// If start_timestamp is provided, validate it (duration is still required and used)
	if req.StartTimestamp != 0 {
		if err := calculateDurationFromTimestamp(req.StartTimestamp); err != nil {
			return CalculateResult{}, err
		}
	}

	if err := validateRequest(req); err != nil {
		return CalculateResult{}, err
	}

	normalizedReq := req
	normalizedReq.RequestedDurationMinutes = normalizeDuration(
		req.RequestedDurationMinutes,
		effectiveRequestedDurationStepMinutes(req.RequestedDurationStepMinutes),
	)

	minimumDuration := minDuration(normalizedReq.Periods)

	var (
		result CalculateResult
		err    error
	)

	if normalizedReq.PricingMode == PricingModeRoundUpMinimumAndProrateAny {
		if normalizedReq.RequestedDurationMinutes < minimumDuration {
			result, err = priceBelowMinimum(CalculateRequest{
				RequestedDurationMinutes: normalizedReq.RequestedDurationMinutes,
				Periods:                  normalizedReq.Periods,
				PricingMode:              PricingModeRoundUp,
			})
		} else {
			result, err = optimizePriceWithOptionalProration(normalizedReq.RequestedDurationMinutes, normalizedReq.Periods)
		}
	} else if normalizedReq.PricingMode == PricingModeProrateAny {
		result, err = optimizePriceWithOptionalProration(normalizedReq.RequestedDurationMinutes, normalizedReq.Periods)
	} else if normalizedReq.RequestedDurationMinutes < minimumDuration {
		result, err = priceBelowMinimum(normalizedReq)
	} else {
		result, err = optimizePrice(normalizedReq.RequestedDurationMinutes, normalizedReq.Periods)
	}

	if err != nil {
		return CalculateResult{}, err
	}

	result.TotalPrice = roundUpPrice(result.TotalPrice, effectiveTotalPriceStep(req.TotalPriceStep))

	// If start_timestamp was provided, include it and calculate end_timestamp
	if req.StartTimestamp != 0 {
		result.StartTimestamp = req.StartTimestamp
		result.EndTimestamp = req.StartTimestamp + int64(result.CoveredMinutes*60)
	}

	return result, nil
}
