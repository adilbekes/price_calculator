package pricecalculator

import (
	"encoding/json"
	"fmt"
)

type PricingMode int

const (
	PricingModeRoundUp PricingMode = iota
	PricingModeProrateMinimum
	PricingModeProrateAny
	PricingModeRoundUpMinimumAndProrateAny
)

var pricingModeNames = map[PricingMode]string{
	PricingModeRoundUp:                     "RoundUp",
	PricingModeProrateMinimum:              "ProrateMinimum",
	PricingModeProrateAny:                  "ProrateAny",
	PricingModeRoundUpMinimumAndProrateAny: "RoundUpMinimumAndProrateAny",
}

var pricingModeValues = map[string]PricingMode{
	"RoundUp":                     PricingModeRoundUp,
	"ProrateMinimum":              PricingModeProrateMinimum,
	"ProrateAny":                  PricingModeProrateAny,
	"RoundUpMinimumAndProrateAny": PricingModeRoundUpMinimumAndProrateAny,
}

func (pm PricingMode) String() string {
	if name, ok := pricingModeNames[pm]; ok {
		return name
	}
	return fmt.Sprintf("PricingMode(%d)", int(pm))
}

func (pm PricingMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(pm.String())
}

func (pm *PricingMode) UnmarshalJSON(data []byte) error {
	// accept string names
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v, ok := pricingModeValues[s]
		if !ok {
			return fmt.Errorf("unknown pricing mode %q", s)
		}
		*pm = v
		return nil
	}
	// accept integer values
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("pricing_mode must be a string name or integer: %w", err)
	}
	*pm = PricingMode(n)
	return nil
}

type PricingPeriod struct {
	Id              string `json:"id,omitempty"`
	DurationMinutes int    `json:"duration"`
	Price           int64  `json:"price"`
}

func (pp PricingPeriod) String() string {
	return fmt.Sprintf("%d⏱ - %d💰", pp.DurationMinutes, pp.Price)
}

type CalculateRequest struct {
	RequestedDurationMinutes        int             `json:"duration"` // Required: rental duration in minutes
	StartTimestamp                  int64           `json:"start_timestamp,omitempty"` // Optional: Unix timestamp (seconds) - if provided, duration is still used to calculate end time
	RequestedDurationStepMinutes    int             `json:"duration_step,omitempty"`
	RequestedMinimumDurationMinutes int             `json:"min_duration,omitempty"`
	Periods                         []PricingPeriod `json:"periods"`
	PricingMode                     PricingMode     `json:"mode"`
	TotalPriceStep                  int64           `json:"price_step,omitempty"` // optional; defaults to 1 when zero (no rounding)
}

type BreakdownItem struct {
	Id              string `json:"id,omitempty"`
	DurationMinutes int    `json:"duration"`
	Price           int64  `json:"price"`
	Quantity        int    `json:"quantity"`
}

func (bi BreakdownItem) String() string {
	return fmt.Sprintf("%dx[%d⏱ - %d💰]", bi.Quantity, bi.DurationMinutes, bi.Price)
}

type CalculateResult struct {
	StartTimestamp int64           `json:"start_timestamp,omitempty"` // Unix timestamp (seconds) if provided in request
	EndTimestamp   int64           `json:"end_timestamp,omitempty"`   // Calculated as start_timestamp + (duration * 60)
	TotalPrice     int64           `json:"total"`
	CoveredMinutes int             `json:"covered"`
	Breakdown      []BreakdownItem `json:"breakdown"`
}
