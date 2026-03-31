package pricecalculator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var defaultPeriods = []PricingPeriod{
	{DurationMinutes: 60, Price: 1000},
	{DurationMinutes: 120, Price: 1800},
	{DurationMinutes: 180, Price: 2500},
}

type successTestCase struct {
	name     string
	req      CalculateRequest
	expected CalculateResult
}

type errorTestCase struct {
	name string
	req  CalculateRequest
	err  error
}

func runSuccessTests(t *testing.T, tests []successTestCase) {
	t.Helper()

	calc := NewCalculator()

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := calc.Calculate(tt.req)

			require.NoError(t, err)
			require.Equal(t, tt.expected.TotalPrice, result.TotalPrice)
			require.Equal(t, tt.expected.CoveredMinutes, result.CoveredMinutes)
		})
	}
}

func runErrorTests(t *testing.T, tests []errorTestCase) {
	t.Helper()

	calc := NewCalculator()

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := calc.Calculate(tt.req)

			require.Error(t, err)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, CalculateResult{}, result)
		})
	}
}

func TestCalculator_ValidCases(t *testing.T) {
	t.Parallel()

	tests := []successTestCase{
		{
			name: "exact 1 hour",
			req: CalculateRequest{
				RequestedDurationMinutes: 60,
				Periods:                  defaultPeriods,
			},
			expected: CalculateResult{
				TotalPrice:     1000,
				CoveredMinutes: 60,
			},
		},
		{
			name: "requested duration rounds up to default 5 minute step",
			req: CalculateRequest{
				RequestedDurationMinutes: 59,
				Periods:                  defaultPeriods,
			},
			expected: CalculateResult{
				TotalPrice:     1000,
				CoveredMinutes: 60,
			},
		},
		{
			name: "same duration with different prices is allowed and cheapest is chosen",
			req: CalculateRequest{
				RequestedDurationMinutes: 60,
				Periods: []PricingPeriod{
					{DurationMinutes: 60, Price: 1000},
					{DurationMinutes: 60, Price: 900},
					{DurationMinutes: 120, Price: 1800},
				},
			},
			expected: CalculateResult{
				TotalPrice:     900,
				CoveredMinutes: 60,
			},
		},
		{
			name: "exact 2 hours",
			req: CalculateRequest{
				RequestedDurationMinutes: 120,
				Periods:                  defaultPeriods,
			},
			expected: CalculateResult{
				TotalPrice:     1800,
				CoveredMinutes: 120,
			},
		},
		{
			name: "combine 2h and 3h for 5 hours",
			req: CalculateRequest{
				RequestedDurationMinutes: 300,
				Periods:                  defaultPeriods,
			},
			expected: CalculateResult{
				TotalPrice:     4300,
				CoveredMinutes: 300,
			},
		},
		{
			name: "over coverage allowed (5h30m -> 6h)",
			req: CalculateRequest{
				RequestedDurationMinutes: 330,
				Periods:                  defaultPeriods,
			},
			expected: CalculateResult{
				TotalPrice:     5000,
				CoveredMinutes: 360,
			},
		},
		{
			name: "unsorted periods still works",
			req: CalculateRequest{
				RequestedDurationMinutes: 300,
				Periods: []PricingPeriod{
					{DurationMinutes: 180, Price: 2500},
					{DurationMinutes: 60, Price: 1000},
					{DurationMinutes: 120, Price: 1800},
				},
			},
			expected: CalculateResult{
				TotalPrice:     4300,
				CoveredMinutes: 300,
			},
		},
		{
			name: "below minimum defaults to round up",
			req: CalculateRequest{
				RequestedDurationMinutes: 30,
				Periods:                  defaultPeriods,
			},
			expected: CalculateResult{
				TotalPrice:     1000,
				CoveredMinutes: 60,
			},
		},
		{
			name: "below minimum can be prorated",
			req: CalculateRequest{
				RequestedDurationMinutes: 30,
				Periods:                  defaultPeriods,
				PricingMode:              PricingModeProrateMinimum,
			},
			expected: CalculateResult{
				TotalPrice:     500,
				CoveredMinutes: 30,
			},
		},
		{
			name: "prorate rounds fractional price up",
			req: CalculateRequest{
				RequestedDurationMinutes: 20,
				Periods: []PricingPeriod{
					{DurationMinutes: 60, Price: 1000},
					{DurationMinutes: 120, Price: 1800},
					{DurationMinutes: 180, Price: 2500},
				},
				PricingMode: PricingModeProrateMinimum,
			},
			expected: CalculateResult{
				TotalPrice:     334,
				CoveredMinutes: 20,
			},
		},
		{
			name: "any-range proration keeps exact coverage",
			req: CalculateRequest{
				RequestedDurationMinutes: 150,
				Periods: []PricingPeriod{
					{DurationMinutes: 60, Price: 1000},
					{DurationMinutes: 120, Price: 1300},
					{DurationMinutes: 190, Price: 2500},
				},
				PricingMode: PricingModeProrateAny,
			},
			expected: CalculateResult{
				TotalPrice:     1800,
				CoveredMinutes: 150,
			},
		},
		{
			name: "any-range proration can still prefer cheaper over-coverage",
			req: CalculateRequest{
				RequestedDurationMinutes: 100,
				Periods: []PricingPeriod{
					{DurationMinutes: 60, Price: 1000},
					{DurationMinutes: 120, Price: 1300},
					{DurationMinutes: 190, Price: 2500},
				},
				PricingMode: PricingModeProrateAny,
			},
			expected: CalculateResult{
				TotalPrice:     1300,
				CoveredMinutes: 120,
			},
		},
		{
			name: "any-range proration uses minimum period below minimum",
			req: CalculateRequest{
				RequestedDurationMinutes: 30,
				Periods:                  defaultPeriods,
				PricingMode:              PricingModeProrateAny,
			},
			expected: CalculateResult{
				TotalPrice:     500,
				CoveredMinutes: 30,
			},
		},
		{
			name: "round up below minimum then any-range proration above minimum",
			req: CalculateRequest{
				RequestedDurationMinutes: 30,
				Periods:                  defaultPeriods,
				PricingMode:              PricingModeRoundUpMinimumAndProrateAny,
			},
			expected: CalculateResult{
				TotalPrice:     1000,
				CoveredMinutes: 60,
			},
		},
		{
			name: "hybrid mode uses any-range proration once request reaches minimum",
			req: CalculateRequest{
				RequestedDurationMinutes: 150,
				Periods: []PricingPeriod{
					{DurationMinutes: 60, Price: 1000},
					{DurationMinutes: 120, Price: 1300},
					{DurationMinutes: 190, Price: 2500},
				},
				PricingMode: PricingModeRoundUpMinimumAndProrateAny,
			},
			expected: CalculateResult{
				TotalPrice:     1800,
				CoveredMinutes: 150,
			},
		},
		{
			name: "custom step and minimum are applied",
			req: CalculateRequest{
				RequestedDurationMinutes:        11,
				RequestedDurationStepMinutes:    15,
				RequestedMinimumDurationMinutes: 10,
				Periods:                         defaultPeriods,
				PricingMode:                     PricingModeProrateMinimum,
			},
			expected: CalculateResult{
				TotalPrice:     250,
				CoveredMinutes: 15,
			},
		},
	}

	runSuccessTests(t, tests)
}

func TestCalculator_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []errorTestCase{
		{
			name: "empty periods",
			req: CalculateRequest{
				RequestedDurationMinutes: 120,
				Periods:                  []PricingPeriod{},
			},
			err: ErrInvalidPeriods,
		},
		{
			name: "zero duration",
			req: CalculateRequest{
				RequestedDurationMinutes: 0,
				Periods:                  defaultPeriods,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "below default minimum requested duration",
			req: CalculateRequest{
				RequestedDurationMinutes: 1,
				Periods:                  defaultPeriods,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "below custom minimum requested duration",
			req: CalculateRequest{
				RequestedDurationMinutes:        9,
				RequestedMinimumDurationMinutes: 10,
				Periods:                         defaultPeriods,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid period price",
			req: CalculateRequest{
				RequestedDurationMinutes: 60,
				Periods: []PricingPeriod{
					{DurationMinutes: 60, Price: -100},
				},
			},
			err: ErrInvalidPeriods,
		},
		{
			name: "negative requested duration step",
			req: CalculateRequest{
				RequestedDurationMinutes:     60,
				RequestedDurationStepMinutes: -5,
				Periods:                      defaultPeriods,
			},
			err: ErrInvalidRequest,
		},
		{
			name: "negative minimum requested duration",
			req: CalculateRequest{
				RequestedDurationMinutes:        60,
				RequestedMinimumDurationMinutes: -5,
				Periods:                         defaultPeriods,
			},
			err: ErrInvalidRequest,
		},
		{
			name: "invalid below minimum mode",
			req: CalculateRequest{
				RequestedDurationMinutes: 30,
				Periods:                  defaultPeriods,
				PricingMode:              PricingMode(99),
			},
			err: ErrInvalidRequest,
		},
		{
			name: "exact duplicate period",
			req: CalculateRequest{
				RequestedDurationMinutes: 60,
				Periods: []PricingPeriod{
					{DurationMinutes: 60, Price: 1000},
					{DurationMinutes: 60, Price: 1000},
				},
			},
			err: ErrInvalidPeriods,
		},
	}

	runErrorTests(t, tests)
}

func TestCalculator_AnyRangeProrationBreakdown(t *testing.T) {
	t.Parallel()

	calc := NewCalculator()

	result, err := calc.Calculate(CalculateRequest{
		RequestedDurationMinutes: 150,
		Periods: []PricingPeriod{
			{DurationMinutes: 60, Price: 1000},
			{DurationMinutes: 120, Price: 1300},
			{DurationMinutes: 190, Price: 2500},
		},
		PricingMode: PricingModeProrateAny,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1800), result.TotalPrice)
	require.Equal(t, 150, result.CoveredMinutes)
	require.Equal(t, []BreakdownItem{
		{DurationMinutes: 120, Price: 1300, Quantity: 1},
		{DurationMinutes: 60, Price: 1000, Quantity: 1},
	}, result.Breakdown)
}

func TestCalculator_AnyRangeProrationCanPreferCheaperOverCoverage(t *testing.T) {
	t.Parallel()

	calc := NewCalculator()

	result, err := calc.Calculate(CalculateRequest{
		RequestedDurationMinutes: 100,
		Periods: []PricingPeriod{
			{DurationMinutes: 60, Price: 1000},
			{DurationMinutes: 120, Price: 1300},
			{DurationMinutes: 190, Price: 2500},
		},
		PricingMode: PricingModeProrateAny,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1300), result.TotalPrice)
	require.Equal(t, 120, result.CoveredMinutes)
	require.Equal(t, []BreakdownItem{{
		DurationMinutes: 120,
		Price:           1300,
		Quantity:        1,
	}}, result.Breakdown)
}

func TestCalculator_BelowMinimumProrationBreakdownUsesFullPeriod(t *testing.T) {
	t.Parallel()

	calc := NewCalculator()

	result, err := calc.Calculate(CalculateRequest{
		RequestedDurationMinutes: 30,
		Periods:                  defaultPeriods,
		PricingMode:              PricingModeProrateMinimum,
	})

	require.NoError(t, err)
	require.Equal(t, int64(500), result.TotalPrice)
	require.Equal(t, 30, result.CoveredMinutes)
	require.Equal(t, []BreakdownItem{{
		DurationMinutes: 60,
		Price:           1000,
		Quantity:        1,
	}}, result.Breakdown)
}

func TestCalculator_HybridModeBreakdown(t *testing.T) {
	t.Parallel()

	calc := NewCalculator()

	t.Run("below minimum rounds up", func(t *testing.T) {
		t.Parallel()

		result, err := calc.Calculate(CalculateRequest{
			RequestedDurationMinutes: 30,
			Periods:                  defaultPeriods,
			PricingMode:              PricingModeRoundUpMinimumAndProrateAny,
		})

		require.NoError(t, err)
		require.Equal(t, []BreakdownItem{{
			DurationMinutes: 60,
			Price:           1000,
			Quantity:        1,
		}}, result.Breakdown)
	})

	t.Run("at or above minimum uses any-range proration", func(t *testing.T) {
		t.Parallel()

		result, err := calc.Calculate(CalculateRequest{
			RequestedDurationMinutes: 150,
			Periods: []PricingPeriod{
				{DurationMinutes: 60, Price: 1000},
				{DurationMinutes: 120, Price: 1300},
				{DurationMinutes: 190, Price: 2500},
			},
			PricingMode: PricingModeRoundUpMinimumAndProrateAny,
		})

		require.NoError(t, err)
		require.Equal(t, []BreakdownItem{
			{DurationMinutes: 120, Price: 1300, Quantity: 1},
			{DurationMinutes: 60, Price: 1000, Quantity: 1},
		}, result.Breakdown)
	})
}
