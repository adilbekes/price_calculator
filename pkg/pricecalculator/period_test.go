package pricecalculator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDurationPeriod_RequiresPositiveDuration(t *testing.T) {
	_, err := resolveDurationPeriod(PricingPeriod{Price: 1000})
	require.Error(t, err)
}

func TestResolveTimeRangePeriod_DerivesDurationFromEndTime(t *testing.T) {
	period, err := resolveTimeRangePeriod(PricingPeriod{
		Type:      PeriodTypeTimeRange,
		StartTime: "09:00",
		EndTime:   "18:00",
		Price:     4000,
	})
	require.NoError(t, err)
	assert.Equal(t, 540, period.DurationMinutes)
}

func TestResolveTimeRangePeriod_RejectsDurationMismatch(t *testing.T) {
	_, err := resolveTimeRangePeriod(PricingPeriod{
		Type:            PeriodTypeTimeRange,
		StartTime:       "09:00",
		EndTime:         "18:00",
		DurationMinutes: 60,
		Price:           1000,
	})
	require.Error(t, err)
}

func TestResolveCalendarPeriod_MonthFromSeventh(t *testing.T) {
	start := time.Date(2026, 4, 7, 10, 0, 0, 0, time.Local)
	period, err := resolveCalendarPeriod(PricingPeriod{
		Type:             PeriodTypeCalendar,
		CalendarUnit:     CalendarUnitMonth,
		CalendarInterval: 1,
		Price:            50000,
	}, start)
	require.NoError(t, err)

	end := time.Date(2026, 5, 7, 10, 0, 0, 0, time.Local)
	assert.Equal(t, int(end.Sub(start).Minutes()), period.DurationMinutes)
}

func TestCalculate_CalendarPeriod_CoversRequestedMonth(t *testing.T) {
	c := NewCalculator()
	start := "2026-04-07 10:00:00"
	r, err := c.Calculate(CalculateRequest{
		RequestedDurationMinutes: 30 * 24 * 60, // ~30 days
		StartTime:                start,
		PricingMode:              PricingModeRoundUp,
		Periods: []PricingPeriod{
			{
				Id:               "month",
				Type:             PeriodTypeCalendar,
				CalendarUnit:     CalendarUnitMonth,
				CalendarInterval: 1,
				Price:            50000,
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(50000), r.TotalPrice)
	assert.GreaterOrEqual(t, r.CoveredMinutes, 28*24*60)
}

func TestCalculate_TimeRangePeriod_WithEndTime(t *testing.T) {
	c := NewCalculator()
	r, err := c.Calculate(CalculateRequest{
		RequestedDurationMinutes: 540,
		StartTime:                "2026-04-01 09:00:00",
		PricingMode:              PricingModeRoundUp,
		Periods: []PricingPeriod{
			{
				Id:        "day",
				Type:      PeriodTypeTimeRange,
				StartTime: "09:00",
				EndTime:   "18:00",
				Price:     4000,
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4000), r.TotalPrice)
	assert.Equal(t, 540, r.CoveredMinutes)
}

func TestCalculate_DurationPeriod_Unchanged(t *testing.T) {
	c := NewCalculator()
	r, err := c.Calculate(CalculateRequest{
		RequestedDurationMinutes: 60,
		StartTime:                "2026-04-01 12:00:00",
		PricingMode:              PricingModeRoundUp,
		Periods: []PricingPeriod{
			{Id: "hour", Type: PeriodTypeDuration, DurationMinutes: 60, Price: 1000},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1000), r.TotalPrice)
}
