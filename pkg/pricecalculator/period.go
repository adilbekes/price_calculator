package pricecalculator

import "time"

const minutesPerDay = 1440

// resolvePeriods fills DurationMinutes for every period (calendar → minutes from
// requestStart) and normalizes time_range start/end into a consistent shape.
// Returned periods are copies safe for the optimizer to mutate.
func resolvePeriods(periods []PricingPeriod, requestStart time.Time) ([]PricingPeriod, error) {
	resolved := make([]PricingPeriod, 0, len(periods))

	for i, period := range periods {
		normalized, err := resolvePeriod(period, requestStart)
		if err != nil {
			return nil, NewPeriodsError("period[%d]: %s", i, err.Error())
		}
		resolved = append(resolved, normalized)
	}

	return resolved, nil
}

func resolvePeriod(period PricingPeriod, requestStart time.Time) (PricingPeriod, error) {
	switch period.EffectiveType() {
	case PeriodTypeDuration:
		return resolveDurationPeriod(period)
	case PeriodTypeTimeRange:
		return resolveTimeRangePeriod(period)
	case PeriodTypeCalendar:
		return resolveCalendarPeriod(period, requestStart)
	default:
		return PricingPeriod{}, NewRequestError("unknown period type %q (expected duration, time_range, or calendar)", period.Type)
	}
}

func resolveDurationPeriod(period PricingPeriod) (PricingPeriod, error) {
	if period.EndTime != "" {
		return PricingPeriod{}, NewRequestError("duration periods cannot have end_time")
	}
	if period.CalendarUnit != "" || period.CalendarInterval != 0 {
		return PricingPeriod{}, NewRequestError("duration periods cannot have calendar_unit/calendar_interval")
	}
	if period.StartTime != "" {
		if _, _, err := parseTimeHHMM(period.StartTime); err != nil {
			return PricingPeriod{}, err
		}
	}
	if period.DurationMinutes <= 0 {
		return PricingPeriod{}, NewRequestError("duration must be positive")
	}

	out := period
	out.Type = PeriodTypeDuration
	return out, nil
}

func resolveTimeRangePeriod(period PricingPeriod) (PricingPeriod, error) {
	if period.CalendarUnit != "" || period.CalendarInterval != 0 {
		return PricingPeriod{}, NewRequestError("time_range periods cannot have calendar_unit/calendar_interval")
	}
	if period.StartTime == "" {
		return PricingPeriod{}, NewRequestError("time_range periods require start_time (HH:MM)")
	}

	startHour, startMin, err := parseTimeHHMM(period.StartTime)
	if err != nil {
		return PricingPeriod{}, err
	}

	out := period
	out.Type = PeriodTypeTimeRange

	if period.EndTime != "" {
		endHour, endMin, err := parseTimeHHMM(period.EndTime)
		if err != nil {
			return PricingPeriod{}, err
		}

		startMinutes := startHour*60 + startMin
		endMinutes := endHour*60 + endMin
		if endMinutes < startMinutes {
			return PricingPeriod{}, NewRequestError("end_time must be on or after start_time on the same day")
		}

		leftover := endMinutes - startMinutes
		if period.DurationMinutes == 0 {
			if leftover < 1 {
				return PricingPeriod{}, NewRequestError("time_range duration derived from start_time/end_time must be positive")
			}
			out.DurationMinutes = leftover
		} else if period.DurationMinutes%minutesPerDay != leftover {
			return PricingPeriod{}, NewRequestError(
				"duration (%d) must agree with end_time-start_time leftover (%d) after full days",
				period.DurationMinutes,
				leftover,
			)
		}
	} else if period.DurationMinutes <= 0 {
		return PricingPeriod{}, NewRequestError("time_range periods require duration or end_time")
	}

	if out.DurationMinutes <= 0 {
		return PricingPeriod{}, NewRequestError("duration must be positive")
	}

	return out, nil
}

func resolveCalendarPeriod(period PricingPeriod, requestStart time.Time) (PricingPeriod, error) {
	if period.DurationMinutes != 0 {
		return PricingPeriod{}, NewRequestError("calendar periods cannot have duration (it is derived from calendar_unit/calendar_interval)")
	}
	if period.CalendarInterval < 1 {
		return PricingPeriod{}, NewRequestError("calendar_interval must be at least 1")
	}

	end, err := addCalendarInterval(requestStart, period.CalendarUnit, period.CalendarInterval)
	if err != nil {
		return PricingPeriod{}, err
	}

	minutes := int(end.Sub(requestStart).Minutes())
	if minutes < 1 {
		return PricingPeriod{}, NewRequestError("calendar period resolves to a non-positive duration")
	}

	if (period.StartTime == "") != (period.EndTime == "") {
		return PricingPeriod{}, NewRequestError("calendar periods require both start_time and end_time, or neither")
	}

	if period.StartTime != "" {
		startHour, startMin, err := parseTimeHHMM(period.StartTime)
		if err != nil {
			return PricingPeriod{}, err
		}
		endHour, endMin, err := parseTimeHHMM(period.EndTime)
		if err != nil {
			return PricingPeriod{}, err
		}
		if endHour*60+endMin < startHour*60+startMin {
			return PricingPeriod{}, NewRequestError("end_time must be on or after start_time on the same day")
		}
	}

	out := period
	out.Type = PeriodTypeCalendar
	out.DurationMinutes = minutes
	return out, nil
}

func addCalendarInterval(from time.Time, unit CalendarUnit, interval int) (time.Time, error) {
	switch unit {
	case CalendarUnitDay:
		return from.AddDate(0, 0, interval), nil
	case CalendarUnitWeek:
		return from.AddDate(0, 0, 7*interval), nil
	case CalendarUnitMonth:
		return from.AddDate(0, interval, 0), nil
	case CalendarUnitYear:
		return from.AddDate(interval, 0, 0), nil
	default:
		return time.Time{}, NewRequestError("calendar_unit must be day, week, month, or year, got %q", unit)
	}
}
