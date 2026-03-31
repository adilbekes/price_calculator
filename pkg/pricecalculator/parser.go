package pricecalculator

const (
	defaultRequestedDurationStepMinutes    = 5
	defaultMinimumRequestedDurationMinutes = 5
	defaultTotalPriceStep                  = 1
)

// calculateDurationFromTimestamp validates timestamp format (Unix seconds) and returns duration.
// The timestamp is used for informational purposes only; duration is always the source of truth.
func calculateDurationFromTimestamp(timestamp int64) error {
	if timestamp <= 0 {
		return NewRequestError("start_timestamp must be a positive Unix timestamp (seconds since epoch)")
	}
	return nil
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
