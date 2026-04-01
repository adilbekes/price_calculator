package pricecalculator

import (
	"regexp"
	"strconv"
	"time"
)

const (
	defaultRequestedDurationStepMinutes    = 5
	defaultMinimumRequestedDurationMinutes = 5
	defaultTotalPriceStep                  = 1
)

var nowTime = func() time.Time {
	return time.Now()
}

func timeLocation(t time.Time) *time.Location {
	if loc := t.Location(); loc != nil {
		return loc
	}
	return time.Local
}

// parseTimeHHMM parses a HH:MM time string and returns the hour and minute
func parseTimeHHMM(timeStr string) (int, int, error) {
	re := regexp.MustCompile(`^(\d{2}):(\d{2})$`)
	matches := re.FindStringSubmatch(timeStr)
	if matches == nil {
		return 0, 0, NewRequestError("invalid time format: %s (expected HH:MM, e.g., '09:00')", timeStr)
	}

	hour, _ := strconv.Atoi(matches[1])
	minute, _ := strconv.Atoi(matches[2])

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, NewRequestError("invalid time: %02d:%02d (hour must be 0-23, minute must be 0-59)", hour, minute)
	}

	return hour, minute, nil
}

// getEffectiveDurationMinutes validates period duration settings.
// Duration is always taken from DurationMinutes; StartTime is optional.
func getEffectiveDurationMinutes(period PricingPeriod) (int, error) {
	if period.StartTime != "" {
		if _, _, err := parseTimeHHMM(period.StartTime); err != nil {
			return 0, err
		}
	}

	if period.DurationMinutes <= 0 {
		return 0, NewRequestError("period '%s': duration must be positive", period.Identifier())
	}

	return period.DurationMinutes, nil
}

// getEffectiveStartTime returns the start time from request, or current time if not provided.
func getEffectiveStartTime(datetimeStr string) (time.Time, error) {
	if datetimeStr == "" {
		return nowTime(), nil
	}
	parsedTime, err := time.ParseInLocation(time.DateTime, datetimeStr, time.Local)
	if err != nil {
		return time.Time{}, NewRequestError("start_time must be in datetime format 'YYYY-MM-DD HH:MM:SS'")
	}
	return parsedTime, nil
}

// validateStartTime validates that the time is valid.
func validateStartTime(t time.Time) error {
	if t.IsZero() {
		return NewRequestError("start_time must be a valid datetime")
	}
	return nil
}

func requestInterval(startTime time.Time, durationMinutes int) (time.Time, time.Time) {
	end := startTime.Add(time.Duration(durationMinutes) * time.Minute)
	return startTime, end
}

func isAfterOrAtPeriodStartTime(period PricingPeriod, checkTime time.Time) (bool, error) {
	if period.StartTime == "" {
		return true, nil
	}

	hour, minute, err := parseTimeHHMM(period.StartTime)
	if err != nil {
		return false, NewRequestError("period '%s': %s", period.Identifier(), err.Error())
	}

	start := time.Date(checkTime.Year(), checkTime.Month(), checkTime.Day(), hour, minute, 0, 0, timeLocation(checkTime))
	return checkTime.Equal(start) || checkTime.After(start), nil
}

// getPeriodWindowOverlapMinutes computes how many minutes of a period are usable within a given request interval.
// When a period has start_time and duration, it defines a daily window [start_time, start_time + duration].
// This function returns the overlap between that window and the request interval for each day.
// If no overlap or period has no start_time, returns the full period duration.
func getPeriodWindowOverlapMinutes(period PricingPeriod, intervalStart, intervalEnd time.Time) (int, error) {
	if period.StartTime == "" {
		return period.DurationMinutes, nil
	}

	hour, minute, err := parseTimeHHMM(period.StartTime)
	if err != nil {
		return 0, err
	}

	loc := timeLocation(intervalStart)
	totalOverlapMinutes := 0

	// Check overlap for each day in the interval
	for _, day := range touchedDates(intervalStart, intervalEnd) {
		windowStart := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
		windowEnd := windowStart.Add(time.Duration(period.DurationMinutes) * time.Minute)

		// Clamp window to the request interval
		overlapStart := windowStart
		if overlapStart.Before(intervalStart) {
			overlapStart = intervalStart
		}

		overlapEnd := windowEnd
		if overlapEnd.After(intervalEnd) {
			overlapEnd = intervalEnd
		}

		// If there's overlap on this day, add it
		if overlapEnd.After(overlapStart) {
			totalOverlapMinutes += int(overlapEnd.Sub(overlapStart).Minutes())
		}
	}

	// If no overlap at all, we can't use this period for the interval
	if totalOverlapMinutes == 0 {
		return 0, nil
	}

	return totalOverlapMinutes, nil
}

func touchedDates(start time.Time, end time.Time) []time.Time {
	if !end.After(start) {
		return nil
	}

	loc := timeLocation(start)
	current := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	lastTouched := end.Add(-time.Nanosecond)
	lastDate := time.Date(lastTouched.Year(), lastTouched.Month(), lastTouched.Day(), 0, 0, 0, 0, timeLocation(lastTouched))

	dates := make([]time.Time, 0, int(lastDate.Sub(current).Hours()/24)+1)
	for !current.After(lastDate) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, 1)
	}

	return dates
}

func availabilityWindowForDate(date time.Time, availability interface{}) (bool, time.Time, time.Time, error) {
	loc := timeLocation(date)
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	nextDay := dayStart.AddDate(0, 0, 1)

	if boolVal, ok := availability.(bool); ok {
		if !boolVal {
			return false, time.Time{}, time.Time{}, nil
		}
		return true, dayStart, nextDay, nil
	}

	if timeRangeStr, ok := availability.(string); ok {
		windowStart, windowEnd, err := parseTimeRangeForDate(date, timeRangeStr)
		if err != nil {
			return false, time.Time{}, time.Time{}, err
		}
		return true, windowStart, windowEnd, nil
	}

	return false, time.Time{}, time.Time{}, NewRequestError(
		"availability value for %s must be boolean or time range string (e.g., '10:00-18:00')",
		date.Format(time.DateOnly),
	)
}

func parseTimeRangeForDate(date time.Time, timeRangeStr string) (time.Time, time.Time, error) {
	re := regexp.MustCompile(`^(\d{2}):(\d{2})-(\d{2}):(\d{2})$`)
	matches := re.FindStringSubmatch(timeRangeStr)
	if matches == nil {
		return time.Time{}, time.Time{}, NewRequestError(
			"invalid time range format: %s (expected HH:MM-HH:MM, e.g., '10:00-18:00')",
			timeRangeStr,
		)
	}

	startHour, _ := strconv.Atoi(matches[1])
	startMin, _ := strconv.Atoi(matches[2])
	endHour, _ := strconv.Atoi(matches[3])
	endMin, _ := strconv.Atoi(matches[4])

	if startHour < 0 || startHour > 23 || startMin < 0 || startMin > 59 {
		return time.Time{}, time.Time{}, NewRequestError("invalid start time in range: %02d:%02d", startHour, startMin)
	}
	if endHour < 0 || endHour > 23 || endMin < 0 || endMin > 59 {
		return time.Time{}, time.Time{}, NewRequestError("invalid end time in range: %02d:%02d", endHour, endMin)
	}

	loc := timeLocation(date)
	windowStart := time.Date(date.Year(), date.Month(), date.Day(), startHour, startMin, 0, 0, loc)
	windowEnd := time.Date(date.Year(), date.Month(), date.Day(), endHour, endMin, 0, 0, loc)
	if !windowEnd.After(windowStart) {
		if windowEnd.Equal(windowStart) {
			return time.Time{}, time.Time{}, NewRequestError(
				"invalid time range: %s - start and end times cannot be the same",
				timeRangeStr,
			)
		}
		// end time is before start time on the same day - reject it
		return time.Time{}, time.Time{}, NewRequestError(
			"invalid time range: %s - end time (%02d:%02d) cannot be before start time (%02d:%02d) on the same day",
			timeRangeStr, endHour, endMin, startHour, startMin,
		)
	}

	return windowStart, windowEnd, nil
}

func isPeriodAvailableForInterval(period PricingPeriod, start time.Time, end time.Time) (bool, error) {
	allowedByStartTime, err := isAfterOrAtPeriodStartTime(period, start)
	if err != nil {
		return false, err
	}
	if !allowedByStartTime {
		return false, nil
	}

	if len(period.Availability) == 0 {
		return true, nil
	}

	days := touchedDates(start, end)

	for _, day := range days {
		dateStr := day.Format(time.DateOnly)
		availability, exists := period.Availability[dateStr]
		if !exists {
			return false, NewRequestError(
				"period '%s' availability must define date %s for the requested interval",
				period.Identifier(),
				dateStr,
			)
		}

		if _, _, _, err := availabilityWindowForDate(day, availability); err != nil {
			return false, NewRequestError("period '%s': %s", period.Identifier(), err.Error())
		}
	}

	for _, day := range days {
		dateStr := day.Format(time.DateOnly)
		availability := period.Availability[dateStr]
		available, windowStart, windowEnd, err := availabilityWindowForDate(day, availability)
		if err != nil {
			return false, NewRequestError("period '%s': %s", period.Identifier(), err.Error())
		}
		if !available {
			return false, nil
		}

		segmentStart := start
		if segmentStart.Before(day) {
			segmentStart = day
		}
		segmentEnd := end
		nextDay := day.AddDate(0, 0, 1)
		if segmentEnd.After(nextDay) {
			segmentEnd = nextDay
		}

		if segmentStart.Before(windowStart) || segmentEnd.After(windowEnd) {
			return false, nil
		}
	}

	return true, nil
}

// isPeriodAvailableAtTime checks if a period is available on a given date and time
func isPeriodAvailableAtTime(period PricingPeriod, checkTime time.Time) (bool, error) {
	allowedByStartTime, err := isAfterOrAtPeriodStartTime(period, checkTime)
	if err != nil {
		return false, err
	}
	if !allowedByStartTime {
		return false, nil
	}

	// If no availability map, period is always available
	if len(period.Availability) == 0 {
		return true, nil
	}

	// Convert time to date string (YYYY-MM-DD)
	dateStr := checkTime.Format(time.DateOnly)

	// Check if date exists in availability map
	availability, exists := period.Availability[dateStr]
	if !exists {
		return false, nil // Date not in map means not available
	}

	// Handle boolean values
	if boolVal, ok := availability.(bool); ok {
		return boolVal, nil
	}

	// Handle time range strings (e.g., "10:00-18:00")
	if timeRangeStr, ok := availability.(string); ok {
		return isTimeWithinRange(checkTime, timeRangeStr)
	}

	return false, NewRequestError("period[%s]: availability value for %s must be boolean or time range string (e.g., '10:00-18:00')",
		period.Id, dateStr)
}

// isTimeWithinRange checks if a given time falls within a time range string like "10:00-18:00"
func isTimeWithinRange(checkTime time.Time, timeRangeStr string) (bool, error) {
	// Validate format with regex
	re := regexp.MustCompile(`^(\d{2}):(\d{2})-(\d{2}):(\d{2})$`)
	matches := re.FindStringSubmatch(timeRangeStr)
	if matches == nil {
		return false, NewRequestError("invalid time range format: %s (expected HH:MM-HH:MM, e.g., '10:00-18:00')", timeRangeStr)
	}

	startHour, _ := strconv.Atoi(matches[1])
	startMin, _ := strconv.Atoi(matches[2])
	endHour, _ := strconv.Atoi(matches[3])
	endMin, _ := strconv.Atoi(matches[4])

	// Validate hours and minutes
	if startHour < 0 || startHour > 23 || startMin < 0 || startMin > 59 {
		return false, NewRequestError("invalid start time in range: %02d:%02d", startHour, startMin)
	}
	if endHour < 0 || endHour > 23 || endMin < 0 || endMin > 59 {
		return false, NewRequestError("invalid end time in range: %02d:%02d", endHour, endMin)
	}

	// Create time objects for comparison (using today's date)
	loc := timeLocation(checkTime)
	startTime := time.Date(checkTime.Year(), checkTime.Month(), checkTime.Day(), startHour, startMin, 0, 0, loc)
	endTime := time.Date(checkTime.Year(), checkTime.Month(), checkTime.Day(), endHour, endMin, 0, 0, loc)

	// Validate that end time is not before start time on the same day
	if endTime.Before(startTime) {
		return false, NewRequestError(
			"invalid time range: %s - end time (%02d:%02d) cannot be before start time (%02d:%02d) on the same day",
			timeRangeStr, endHour, endMin, startHour, startMin,
		)
	}

	if endTime.Equal(startTime) {
		return false, NewRequestError(
			"invalid time range: %s - start and end times cannot be the same",
			timeRangeStr,
		)
	}

	return (checkTime.Equal(startTime) || checkTime.After(startTime)) && checkTime.Before(endTime), nil
}

func effectiveRequestedDurationStepMinutes(stepMinutes int) int {
	if stepMinutes == 0 {
		return defaultRequestedDurationStepMinutes
	}
	return stepMinutes
}

func effectiveRequestedMinimumDurationMinutes(minimumMinutes int) int {
	if minimumMinutes == 0 {
		return defaultMinimumRequestedDurationMinutes
	}
	return minimumMinutes
}

func effectiveTotalPriceStep(step int64) int64 {
	if step == 0 {
		return defaultTotalPriceStep
	}
	return step
}

func normalizeDuration(minutes int, stepMinutes int) int {
	return ((minutes + stepMinutes - 1) / stepMinutes) * stepMinutes
}

// roundUpPrice rounds price up to the nearest multiple of step.
// If step is 1 the price is returned unchanged.
func roundUpPrice(price, step int64) int64 {
	if step <= 1 {
		return price
	}
	return ((price + step - 1) / step) * step
}
