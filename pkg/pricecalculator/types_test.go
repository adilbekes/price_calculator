package pricecalculator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── PricingMode ──────────────────────────────────────────────────────────────

func TestPricingMode_String_AllKnownValues(t *testing.T) {
	cases := map[PricingMode]string{
		PricingModeRoundUp:                     "RoundUp",
		PricingModeProrateMinimum:              "ProrateMinimum",
		PricingModeProrateAny:                  "ProrateAny",
		PricingModeRoundUpMinimumAndProrateAny: "RoundUpMinimumAndProrateAny",
	}
	for mode, want := range cases {
		assert.Equal(t, want, mode.String())
	}
}

func TestPricingMode_String_UnknownValue_ContainsInteger(t *testing.T) {
	unknown := PricingMode(99)
	assert.Contains(t, unknown.String(), "99")
}

func TestPricingMode_UnmarshalJSON_AcceptsStringName(t *testing.T) {
	var pm PricingMode
	require.NoError(t, json.Unmarshal([]byte(`"ProrateAny"`), &pm))
	assert.Equal(t, PricingModeProrateAny, pm)
}

func TestPricingMode_UnmarshalJSON_AcceptsIntegerValue(t *testing.T) {
	var pm PricingMode
	require.NoError(t, json.Unmarshal([]byte(`2`), &pm))
	assert.Equal(t, PricingModeProrateAny, pm)
}

func TestPricingMode_UnmarshalJSON_UnknownStringName_ReturnsError(t *testing.T) {
	var pm PricingMode
	err := json.Unmarshal([]byte(`"InvalidMode"`), &pm)
	assert.Error(t, err)
}

func TestPricingMode_MarshalJSON_ReturnsQuotedStringName(t *testing.T) {
	pm := PricingModeRoundUp
	b, err := json.Marshal(&pm)
	require.NoError(t, err)
	assert.Equal(t, `"RoundUp"`, string(b))
}

func TestPricingMode_MarshalJSON_NilPointer_ReturnsNull(t *testing.T) {
	var pm *PricingMode
	b, err := json.Marshal(pm)
	require.NoError(t, err)
	assert.Equal(t, "null", string(b))
}

// ─── PricingPeriod ────────────────────────────────────────────────────────────

func TestPricingPeriod_String_WithStartTime_IncludesAll(t *testing.T) {
	p := PricingPeriod{DurationMinutes: 60, Price: 1000, StartTime: "09:00"}
	s := p.String()
	assert.Contains(t, s, "09:00")
	assert.Contains(t, s, "60")
	assert.Contains(t, s, "1000")
}

func TestPricingPeriod_String_WithoutStartTime_NoClock(t *testing.T) {
	p := PricingPeriod{DurationMinutes: 60, Price: 1000}
	s := p.String()
	assert.Contains(t, s, "60")
	assert.Contains(t, s, "1000")
	assert.NotContains(t, s, ":")
}

func TestPricingPeriod_Identifier_ReturnsId_WhenSet(t *testing.T) {
	p := PricingPeriod{Id: "rate-a", DurationMinutes: 60, Price: 1000}
	assert.Equal(t, "rate-a", p.Identifier())
}

func TestPricingPeriod_Identifier_FallsBackToString_WhenNoId(t *testing.T) {
	p := PricingPeriod{DurationMinutes: 60, Price: 1000}
	assert.Equal(t, p.String(), p.Identifier())
}

// ─── BreakdownItem ────────────────────────────────────────────────────────────

func TestBreakdownItem_String_FullMatch_ShowsQuantityAndSinglePair(t *testing.T) {
	bi := BreakdownItem{Quantity: 3, DurationMinutes: 60, UsedDuration: 60, Price: 1000, UsedPrice: 1000}
	s := bi.String()
	assert.Contains(t, s, "3")
	assert.Contains(t, s, "60")
	assert.Contains(t, s, "1000")
}

func TestBreakdownItem_String_DurationAndPriceBothDiffer_ShowsBothPairs(t *testing.T) {
	bi := BreakdownItem{Quantity: 1, DurationMinutes: 60, UsedDuration: 30, Price: 1000, UsedPrice: 500}
	s := bi.String()
	assert.Contains(t, s, "30")
	assert.Contains(t, s, "60")
	assert.Contains(t, s, "500")
	assert.Contains(t, s, "1000")
}

func TestBreakdownItem_String_DurationDiffersButPriceSame_ShowsBothDurations(t *testing.T) {
	bi := BreakdownItem{Quantity: 1, DurationMinutes: 60, UsedDuration: 30, Price: 1000, UsedPrice: 1000}
	s := bi.String()
	assert.Contains(t, s, "30")
	assert.Contains(t, s, "60")
}

func TestBreakdownItem_String_WithStartAndEndTime_IncludesTimestamps(t *testing.T) {
	bi := BreakdownItem{
		Quantity: 1, DurationMinutes: 60, UsedDuration: 60, Price: 1000, UsedPrice: 1000,
		StartTime: "2026-04-01 09:00:00", EndTime: "2026-04-01 10:00:00",
	}
	s := bi.String()
	assert.Contains(t, s, "2026-04-01 09:00:00")
	assert.Contains(t, s, "2026-04-01 10:00:00")
}

func TestBreakdownItem_String_NoStartEndTime_OmitsArrow(t *testing.T) {
	bi := BreakdownItem{Quantity: 1, DurationMinutes: 60, UsedDuration: 60, Price: 1000, UsedPrice: 1000}
	assert.NotContains(t, bi.String(), "→")
}

