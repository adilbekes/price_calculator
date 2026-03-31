# Price Calculator Spec v1

## Input
- Requested duration in minutes
- Optional requested duration step in minutes, default `5`
- Optional minimum requested duration in minutes, default `5`
- List of pricing periods
- Pricing mode for requests that may need proration

## Validation
- Raw requested duration must be >= minimum requested duration
- Requested duration step must be > 0 after defaults are applied
- Minimum requested duration must be > 0 after defaults are applied
- Periods must not be empty
- Each period:
    - DurationMinutes > 0
    - Price >= 0
- Exact duplicate periods with the same `DurationMinutes` and `Price` are invalid
- Same `DurationMinutes` with different `Price` values are allowed

## Calculation
- Requested duration is rounded up to the nearest configured step before pricing
- Periods can be reused unlimited times
- Find minimum total price
- Covered time may exceed requested
- Breakdown items represent the source pricing periods used
- For prorated results, breakdown duration and price still reference the full source period; actual charged amount is reflected in `TotalPrice`
- If the same source period is used multiple times, it is shown once in breakdown with an aggregated `Quantity`

### Pricing modes
- `PricingModeRoundUp`: below-minimum requests are rounded up to the cheapest minimum-duration period
- `PricingModeProrate`: below-minimum requests are prorated from the cheapest minimum-duration period
- `PricingModeProrateAnyRange`: any request compares normal full-period coverage against a result built from full periods plus one prorated remainder from the minimum-duration period, and returns the cheaper option
- `PricingModeRoundUpBelowMinimumProrateAnyRange`: below-minimum requests are rounded up, while requests at or above the minimum duration compare normal coverage against any-range proration from the minimum-duration period and return the cheaper option

## Errors
- ErrInvalidDuration
- ErrInvalidPeriods
- ErrNoSolution