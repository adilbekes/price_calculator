package pricecalculator

import (
	"strconv"
	"time"
)

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

	effectiveTimestamp, err := getEffectiveStartTime(req.StartTime)
	if err != nil {
		return err
	}

	normalizedDuration := normalizeDuration(
		req.RequestedDurationMinutes,
		effectiveRequestedDurationStepMinutes(req.RequestedDurationStepMinutes),
	)
	intervalStart, intervalEnd := requestInterval(effectiveTimestamp, normalizedDuration)

	if err := validatePeriodsAvailability(req.Periods, intervalStart, intervalEnd); err != nil {
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

	seenPeriods := make(map[string]struct{}, len(periods))
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
		// Validate that period has either time-based (startTime+endTime) or duration-based specification
		effectiveDuration, err := getEffectiveDurationMinutes(period)
		if err != nil {
			return NewPeriodsError("period[%d]: %s", i, err.Error())
		}

		if period.Price < 0 {
			return NewPeriodsError("period[%d]: price cannot be negative, got %d", i, period.Price)
		}

		if hasAnyId {
			// If any period has an ID, all periods must have an ID and IDs must be unique.
			if period.Id == "" {
				return NewPeriodsError("period[%d]: all periods must have an id if any period has an id", i)
			}

			if _, exists := seenIds[period.Id]; exists {
				return NewPeriodsError("period[%d]: duplicate id '%s'", i, period.Id)
			}
			seenIds[period.Id] = struct{}{}
			continue
		}

		key := period.StartTime + "|" + strconv.Itoa(effectiveDuration) + "|" + strconv.FormatInt(period.Price, 10)
		if _, exists := seenPeriods[key]; exists {
			if period.StartTime != "" {
				return NewPeriodsError("period[%d]: duplicate period (start_time=%s, duration=%d, price=%d)", i, period.StartTime, effectiveDuration, period.Price)
			}
			return NewPeriodsError("period[%d]: duplicate period (duration=%d, price=%d)", i, effectiveDuration, period.Price)
		}

		seenPeriods[key] = struct{}{}
	}

	return nil
}

func validatePeriodsAvailability(periods []PricingPeriod, intervalStart time.Time, intervalEnd time.Time) error {
	for _, period := range periods {
		if len(period.Availability) == 0 {
			continue
		}

		for _, day := range touchedDates(intervalStart, intervalEnd) {
			dateStr := day.Format(time.DateOnly)
			availability, exists := period.Availability[dateStr]
			if !exists {
				// Missing dates default to available all day.
				continue
			}

			if _, _, _, err := availabilityWindowForDate(day, availability); err != nil {
				return NewRequestError("period '%s': %s", period.Identifier(), err.Error())
			}
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
