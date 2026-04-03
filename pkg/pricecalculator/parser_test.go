package pricecalculator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── parseTimeHHMM ────────────────────────────────────────────────────────────

func TestParseTimeHHMM_Midnight_Valid(t *testing.T) {
	h, m, err := parseTimeHHMM("00:00")
	require.NoError(t, err)
	assert.Equal(t, 0, h)
	assert.Equal(t, 0, m)
}

func TestParseTimeHHMM_EndOfDay_Valid(t *testing.T) {
	h, m, err := parseTimeHHMM("23:59")
	require.NoError(t, err)
	assert.Equal(t, 23, h)
	assert.Equal(t, 59, m)
}

func TestParseTimeHHMM_Noon_Valid(t *testing.T) {
	h, m, err := parseTimeHHMM("12:30")
	require.NoError(t, err)
	assert.Equal(t, 12, h)
	assert.Equal(t, 30, m)
}

func TestParseTimeHHMM_InvalidMinute_ReturnsError(t *testing.T) {
	_, _, err := parseTimeHHMM("10:60")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestParseTimeHHMM_InvalidHour_ReturnsError(t *testing.T) {
	_, _, err := parseTimeHHMM("24:00")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestParseTimeHHMM_SingleDigitHour_ReturnsError(t *testing.T) {
	_, _, err := parseTimeHHMM("9:00")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestParseTimeHHMM_EmptyString_ReturnsError(t *testing.T) {
	_, _, err := parseTimeHHMM("")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestParseTimeHHMM_NoColon_ReturnsError(t *testing.T) {
	_, _, err := parseTimeHHMM("0900")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestParseTimeHHMM_DatetimeString_ReturnsError(t *testing.T) {
	_, _, err := parseTimeHHMM("2026-04-01 09:00:00")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

// ─── getEffectiveStartTime ────────────────────────────────────────────────────

func TestGetEffectiveStartTime_EmptyString_DelegatesToNowTime(t *testing.T) {
	fixed := time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local)
	orig := nowTime
	t.Cleanup(func() { nowTime = orig })
	nowTime = func() time.Time { return fixed }

	result, err := getEffectiveStartTime("")
	require.NoError(t, err)
	assert.Equal(t, fixed, result)
}

func TestGetEffectiveStartTime_ValidDatetime_ParsesAllFields(t *testing.T) {
	result, err := getEffectiveStartTime("2026-04-01 09:30:00")
	require.NoError(t, err)
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.April, result.Month())
	assert.Equal(t, 1, result.Day())
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 30, result.Minute())
	assert.Equal(t, 0, result.Second())
}

func TestGetEffectiveStartTime_DateOnlyFormat_ReturnsError(t *testing.T) {
	_, err := getEffectiveStartTime("2026-04-01")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestGetEffectiveStartTime_WrongDelimiters_ReturnsError(t *testing.T) {
	_, err := getEffectiveStartTime("2026/04/01 09:00:00")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestGetEffectiveStartTime_ArbitraryString_ReturnsError(t *testing.T) {
	_, err := getEffectiveStartTime("not-a-date")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestGetEffectiveStartTime_TimeOnly_ReturnsError(t *testing.T) {
	_, err := getEffectiveStartTime("09:00:00")
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

// ─── getEffectiveDurationMinutes ─────────────────────────────────────────────

func TestGetEffectiveDurationMinutes_PositiveDuration_Valid(t *testing.T) {
	d, err := getEffectiveDurationMinutes(PricingPeriod{DurationMinutes: 90})
	require.NoError(t, err)
	assert.Equal(t, 90, d)
}

func TestGetEffectiveDurationMinutes_ZeroDuration_ReturnsError(t *testing.T) {
	_, err := getEffectiveDurationMinutes(PricingPeriod{DurationMinutes: 0})
	assert.Error(t, err)
}

func TestGetEffectiveDurationMinutes_NegativeDuration_ReturnsError(t *testing.T) {
	_, err := getEffectiveDurationMinutes(PricingPeriod{DurationMinutes: -10})
	assert.Error(t, err)
}

func TestGetEffectiveDurationMinutes_ValidStartTimeFormat_DoesNotError(t *testing.T) {
	d, err := getEffectiveDurationMinutes(PricingPeriod{DurationMinutes: 60, StartTime: "09:00"})
	require.NoError(t, err)
	assert.Equal(t, 60, d)
}

func TestGetEffectiveDurationMinutes_InvalidStartTimeFormat_ReturnsError(t *testing.T) {
	_, err := getEffectiveDurationMinutes(PricingPeriod{DurationMinutes: 60, StartTime: "9:00"})
	assert.Error(t, err)
}

func TestGetEffectiveDurationMinutes_InvalidStartTimeHour_ReturnsError(t *testing.T) {
	_, err := getEffectiveDurationMinutes(PricingPeriod{DurationMinutes: 60, StartTime: "25:00"})
	assert.Error(t, err)
}

// ─── normalizeDuration ────────────────────────────────────────────────────────

func TestNormalizeDuration_AlreadyAlignedToStep_Unchanged(t *testing.T) {
	assert.Equal(t, 60, normalizeDuration(60, 5))
}

func TestNormalizeDuration_BelowStep_RoundsUpToStep(t *testing.T) {
	assert.Equal(t, 60, normalizeDuration(1, 60))
}

func TestNormalizeDuration_OneAboveMultiple_RoundsToNext(t *testing.T) {
	assert.Equal(t, 65, normalizeDuration(61, 5))
}

func TestNormalizeDuration_ExactMultiple_Unchanged(t *testing.T) {
	assert.Equal(t, 120, normalizeDuration(120, 120))
}

// ─── roundUpPrice ─────────────────────────────────────────────────────────────

func TestRoundUpPrice_StepOne_ReturnsUnchanged(t *testing.T) {
	assert.Equal(t, int64(1003), roundUpPrice(1003, 1))
}

func TestRoundUpPrice_StepZero_TreatedAsStepOne(t *testing.T) {
	assert.Equal(t, int64(1003), roundUpPrice(1003, 0))
}

func TestRoundUpPrice_AlreadyMultipleOfStep_Unchanged(t *testing.T) {
	assert.Equal(t, int64(1000), roundUpPrice(1000, 10))
}

func TestRoundUpPrice_NotAligned_RoundsUpToNextMultiple(t *testing.T) {
	assert.Equal(t, int64(1010), roundUpPrice(1001, 10))
	assert.Equal(t, int64(1010), roundUpPrice(1009, 10))
}

func TestRoundUpPrice_LargeStep_RoundsUpCorrectly(t *testing.T) {
	assert.Equal(t, int64(500), roundUpPrice(1, 500))
}

func TestRoundUpPrice_ZeroPrice_ReturnsZero(t *testing.T) {
	assert.Equal(t, int64(0), roundUpPrice(0, 10))
}

// ─── touchedDates ─────────────────────────────────────────────────────────────

func TestTouchedDates_SameDayInterval_ReturnsSingleDate(t *testing.T) {
	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local)
	end := time.Date(2026, 4, 1, 18, 0, 0, 0, time.Local)
	dates := touchedDates(start, end)
	require.Len(t, dates, 1)
	assert.Equal(t, "2026-04-01", dates[0].Format(time.DateOnly))
}

func TestTouchedDates_OvernightInterval_ReturnsBothDays(t *testing.T) {
	start := time.Date(2026, 4, 1, 22, 0, 0, 0, time.Local)
	end := time.Date(2026, 4, 2, 4, 0, 0, 0, time.Local)
	dates := touchedDates(start, end)
	require.Len(t, dates, 2)
	assert.Equal(t, "2026-04-01", dates[0].Format(time.DateOnly))
	assert.Equal(t, "2026-04-02", dates[1].Format(time.DateOnly))
}

func TestTouchedDates_MultiDay_ReturnsAllDates(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 4, 4, 0, 0, 0, 0, time.Local)
	dates := touchedDates(start, end)
	require.Len(t, dates, 3)
	assert.Equal(t, "2026-04-01", dates[0].Format(time.DateOnly))
	assert.Equal(t, "2026-04-03", dates[2].Format(time.DateOnly))
}

func TestTouchedDates_EndEqualsStart_ReturnsEmpty(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local)
	assert.Empty(t, touchedDates(t0, t0))
}

func TestTouchedDates_EndBeforeStart_ReturnsEmpty(t *testing.T) {
	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local)
	end := time.Date(2026, 4, 1, 9, 0, 0, 0, time.Local)
	assert.Empty(t, touchedDates(start, end))
}

func TestTouchedDates_EndExactlyAtMidnight_ExcludesNextDay(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 4, 2, 0, 0, 0, 0, time.Local)
	dates := touchedDates(start, end)
	require.Len(t, dates, 1)
	assert.Equal(t, "2026-04-01", dates[0].Format(time.DateOnly))
}

// ─── availabilityWindowForDate ────────────────────────────────────────────────

func TestAvailabilityWindowForDate_FalseValue_NotAvailable(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	avail, _, _, err := availabilityWindowForDate(date, false)
	require.NoError(t, err)
	assert.False(t, avail)
}

func TestAvailabilityWindowForDate_TrueValue_AvailableFullDay(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	avail, start, end, err := availabilityWindowForDate(date, true)
	require.NoError(t, err)
	assert.True(t, avail)
	assert.Equal(t, "2026-04-01", start.Format(time.DateOnly))
	assert.Equal(t, "2026-04-02", end.Format(time.DateOnly))
}

func TestAvailabilityWindowForDate_TimeRangeString_ReturnsWindow(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	avail, start, end, err := availabilityWindowForDate(date, "09:00-18:00")
	require.NoError(t, err)
	assert.True(t, avail)
	assert.Equal(t, 9, start.Hour())
	assert.Equal(t, 0, start.Minute())
	assert.Equal(t, 18, end.Hour())
	assert.Equal(t, 0, end.Minute())
}

func TestAvailabilityWindowForDate_TimeRangeArray_EarliestStartAndLatestEnd(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	avail, start, end, err := availabilityWindowForDate(date, []interface{}{"14:00-18:00", "09:00-12:00"})
	require.NoError(t, err)
	assert.True(t, avail)
	assert.Equal(t, 9, start.Hour())
	assert.Equal(t, 18, end.Hour())
}

func TestAvailabilityWindowForDate_TimeRangeArray_SingleEntry_Works(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	avail, start, end, err := availabilityWindowForDate(date, []interface{}{"10:00-16:00"})
	require.NoError(t, err)
	assert.True(t, avail)
	assert.Equal(t, 10, start.Hour())
	assert.Equal(t, 16, end.Hour())
}

func TestAvailabilityWindowForDate_EmptyArray_ReturnsError(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	_, _, _, err := availabilityWindowForDate(date, []interface{}{})
	assert.Error(t, err)
}

func TestAvailabilityWindowForDate_ArrayWithNonString_ReturnsError(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	_, _, _, err := availabilityWindowForDate(date, []interface{}{true})
	assert.Error(t, err)
}

func TestAvailabilityWindowForDate_InvalidType_ReturnsError(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	_, _, _, err := availabilityWindowForDate(date, 42)
	assert.Error(t, err)
}

func TestAvailabilityWindowForDate_InvalidTimeRangeString_ReturnsError(t *testing.T) {
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
	_, _, _, err := availabilityWindowForDate(date, "not-a-range")
	assert.Error(t, err)
}

// ─── effective defaults ────────────────────────────────────────────────────────

func TestEffectiveDurationStep_ZeroInput_ReturnsDefault(t *testing.T) {
	assert.Equal(t, defaultRequestedDurationStepMinutes, effectiveRequestedDurationStepMinutes(0))
}

func TestEffectiveDurationStep_NonZeroInput_PassesThrough(t *testing.T) {
	assert.Equal(t, 15, effectiveRequestedDurationStepMinutes(15))
}

func TestEffectiveMinimumDuration_ZeroInput_ReturnsDefault(t *testing.T) {
	assert.Equal(t, defaultMinimumRequestedDurationMinutes, effectiveRequestedMinimumDurationMinutes(0))
}

func TestEffectiveMinimumDuration_NonZeroInput_PassesThrough(t *testing.T) {
	assert.Equal(t, 30, effectiveRequestedMinimumDurationMinutes(30))
}

func TestEffectiveTotalPriceStep_ZeroInput_ReturnsDefault(t *testing.T) {
	assert.Equal(t, int64(defaultTotalPriceStep), effectiveTotalPriceStep(0))
}

func TestEffectiveTotalPriceStep_NonZeroInput_PassesThrough(t *testing.T) {
	assert.Equal(t, int64(50), effectiveTotalPriceStep(50))
}

