package pricecalculator

// validateDuration validates requested duration in minutes.
//
// Rules:
// - duration must be greater than 0
// - if duration <= 0, return ErrInvalidDuration
func validateRequest(req CalculateRequest) error {
	// Duration is always required
	if req.RequestedDurationMinutes <= 0 {
		return NewDurationError("duration must be greater than 0")
	}

	if err := validatePeriods(req.Periods); err != nil {
		return err
	}

	if err := validateRequestedDurationSettings(req); err != nil {
		return err
	}

	minimumMinutes := effectiveRequestedMinimumDurationMinutes(req.RequestedMinimumDurationMinutes)
	if err := validateDuration(req.RequestedDurationMinutes, minimumMinutes); err != nil {
		return err
	}

	if err := validatePricingMode(req.PricingMode); err != nil {
		return err
	}

	return nil
}

func validateRequestedDurationSettings(req CalculateRequest) error {
	if req.RequestedDurationStepMinutes < 0 {
		return NewRequestError("duration_step must be non-negative, got %d", req.RequestedDurationStepMinutes)
	}

	if req.RequestedMinimumDurationMinutes < 0 {
		return NewRequestError("min_duration must be non-negative, got %d", req.RequestedMinimumDurationMinutes)
	}

	if req.TotalPriceStep < 0 {
		return NewRequestError("price_step must be non-negative, got %d", req.TotalPriceStep)
	}

	stepMinutes := effectiveRequestedDurationStepMinutes(req.RequestedDurationStepMinutes)
	minimumMinutes := effectiveRequestedMinimumDurationMinutes(req.RequestedMinimumDurationMinutes)
	if stepMinutes <= 0 || minimumMinutes <= 0 {
		return NewRequestError("invalid duration settings")
	}

	return nil
}

func validatePricingMode(mode PricingMode) error {
	switch mode {
	case PricingModeRoundUp, PricingModeProrateMinimum, PricingModeProrateAny, PricingModeRoundUpMinimumAndProrateAny:
		return nil
	default:
		return NewRequestError("invalid pricing mode: %d (valid range: 0-3)", int(mode))
	}
}

func validatePeriods(periods []PricingPeriod) error {
	if len(periods) == 0 {
		return NewPeriodsError("periods array cannot be empty")
	}

	seenPeriods := make(map[[2]int64]struct{}, len(periods))
	seenIds := make(map[string]struct{})

	// Check if any period has an ID
	hasAnyId := false
	for _, period := range periods {
		if period.Id != "" {
			hasAnyId = true
			break
		}
	}

	for i, period := range periods {
		if period.DurationMinutes <= 0 {
			return NewPeriodsError("period[%d]: duration must be positive, got %d", i, period.DurationMinutes)
		}

		if period.Price < 0 {
			return NewPeriodsError("period[%d]: price cannot be negative, got %d", i, period.Price)
		}

		key := [2]int64{int64(period.DurationMinutes), period.Price}
		if _, exists := seenPeriods[key]; exists {
			return NewPeriodsError("period[%d]: duplicate period (duration=%d, price=%d)", i, period.DurationMinutes, period.Price)
		}

		seenPeriods[key] = struct{}{}

		// If any period has an ID, all periods must have an ID
		if hasAnyId && period.Id == "" {
			return NewPeriodsError("period[%d]: all periods must have an id if any period has an id", i)
		}

		// Validate ID uniqueness if ID is provided
		if period.Id != "" {
			if _, exists := seenIds[period.Id]; exists {
				return NewPeriodsError("period[%d]: duplicate id '%s'", i, period.Id)
			}
			seenIds[period.Id] = struct{}{}
		}
	}

	return nil
}

func validateDuration(minutes int, minimumMinutes int) error {
	if minutes < minimumMinutes {
		return NewDurationError("requested duration (%d) is below minimum (%d)", minutes, minimumMinutes)
	}

	return nil
}
