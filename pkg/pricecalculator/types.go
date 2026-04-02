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

func (pm *PricingMode) String() string {
	if pm == nil {
		return "PricingMode(<nil>)"
	}
	if name, ok := pricingModeNames[*pm]; ok {
		return name
	}
	return fmt.Sprintf("PricingMode(%d)", int(*pm))
}

func (pm *PricingMode) MarshalJSON() ([]byte, error) {
	if pm == nil {
		return []byte("null"), nil
	}
	return json.Marshal(pm.String())
}

func (pm *PricingMode) UnmarshalJSON(data []byte) error {
	if pm == nil {
		return fmt.Errorf("cannot unmarshal into nil *PricingMode")
	}
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
	// Optional fixed start time for this period in HH:MM format.
	// Duration is still taken from DurationMinutes.
	StartTime string `json:"start_time,omitempty"`
	// Availability map with dates (YYYY-MM-DD) as keys.
	// Value can be:
	// - true: available all day (00:00-23:59)
	// - false: not available
	// - string: time range like "10:00-17:59" (available during this range)
	Availability map[string]interface{} `json:"availability,omitempty"`
}

func (pp PricingPeriod) String() string {
	if pp.StartTime != "" {
		return fmt.Sprintf("%s + %d⏱ - %d💰", pp.StartTime, pp.DurationMinutes, pp.Price)
	}
	return fmt.Sprintf("%d⏱ - %d💰", pp.DurationMinutes, pp.Price)
}

func (pp PricingPeriod) Identifier() string {
	if pp.Id != "" {
		return pp.Id
	}

	return pp.String()
}

type CalculateRequest struct {
	RequestedDurationMinutes        int             `json:"duration"`                  // Required: rental duration in minutes
	StartTime                       string          `json:"start_time,omitempty"` // Optional: datetime string in format "2006-01-02 15:04:05"; defaults to current time
	RequestedDurationStepMinutes    int             `json:"duration_step,omitempty"`
	RequestedMinimumDurationMinutes int             `json:"min_duration,omitempty"`
	Periods                         []PricingPeriod `json:"periods"`
	PricingMode                     PricingMode     `json:"mode"`
	TotalPriceStep                  int64           `json:"price_step,omitempty"` // optional; defaults to 1 when zero (no rounding)
}

type BreakdownItem struct {
	Id              string `json:"id,omitempty"`
	DurationMinutes int    `json:"duration"`      // Period's catalog duration
	UsedDuration    int    `json:"used_duration"` // Actual minutes used; defaults to duration for non-prorated items
	Price           int64  `json:"price"`         // Period's catalog price
	UsedPrice       int64  `json:"used_price"`    // Actual charged price for the used minutes; defaults to price
	Quantity        int    `json:"quantity"`
	StartTime       string `json:"start_time,omitempty"` // Actual start time when this period is used (if period has start_time)
	EndTime         string `json:"end_time,omitempty"`   // Actual end time when this period is used (if period has start_time)
}

func (bi BreakdownItem) String() string {
	if bi.UsedDuration != bi.DurationMinutes {
		// For prorated usage, show both period duration and actual used
		if bi.UsedPrice != bi.Price {
			timeInfo := ""
			if bi.StartTime != "" && bi.EndTime != "" {
				timeInfo = fmt.Sprintf(" %s→%s", bi.StartTime, bi.EndTime)
			}
			return fmt.Sprintf("%dx[%d/%d⏱ - %d/%d💰]%s", bi.Quantity, bi.UsedDuration, bi.DurationMinutes, bi.UsedPrice, bi.Price, timeInfo)
		}
		timeInfo := ""
		if bi.StartTime != "" && bi.EndTime != "" {
			timeInfo = fmt.Sprintf(" %s→%s", bi.StartTime, bi.EndTime)
		}
		return fmt.Sprintf("%dx[%d/%d⏱ - %d💰]%s", bi.Quantity, bi.UsedDuration, bi.DurationMinutes, bi.Price, timeInfo)
	}
	timeInfo := ""
	if bi.StartTime != "" && bi.EndTime != "" {
		timeInfo = fmt.Sprintf(" %s→%s", bi.StartTime, bi.EndTime)
	}
	return fmt.Sprintf("%dx[%d⏱ - %d💰]%s", bi.Quantity, bi.DurationMinutes, bi.Price, timeInfo)
}

type CalculateResult struct {
	StartTime      string          `json:"start_time,omitempty"` // Datetime string if provided in request
	EndTime        string          `json:"end_time,omitempty"` // Calculated as start_time + (duration * 60)
	TotalPrice     int64           `json:"total"`
	CoveredMinutes int             `json:"covered"`
	Breakdown      []BreakdownItem `json:"breakdown"`
}
