# INTEGRATION.md — Integration Notes for External Systems and Package Imports

## Public API

Import the package and use `NewCalculator()` then `Calculate(req)`. All other symbols are unexported and must not be relied upon.

```go
import "price_calculator/pkg/pricecalculator"

c := pricecalculator.NewCalculator()
result, err := c.Calculate(pricecalculator.CalculateRequest{
    Duration: 60,
    Mode:     pricecalculator.RoundUp,
    Periods: []pricecalculator.PricingPeriod{
        {Duration: 30, Price: 500},
        {Duration: 60, Price: 800},
    },
})
```

**Do not** depend on unexported functions (`optimizePrice`, `validateRequest`, etc.) — they may change without notice.

---

## Error Handling

Distinguish error categories with `errors.Is`:

```go
result, err := c.Calculate(req)
if err != nil {
    switch {
    case errors.Is(err, pricecalculator.ErrInvalidRequest):
        // Bad request structure (e.g., missing required field)
    case errors.Is(err, pricecalculator.ErrInvalidDuration):
        // Duration-specific validation failure
    case errors.Is(err, pricecalculator.ErrInvalidPricingPeriods):
        // Periods-specific validation failure
    default:
        // Unexpected error
    }
}
```

---

## JSON Integration (CLI Pattern)

The CLI (`cmd/calculator/main.go`) demonstrates the canonical integration pattern:

1. Read a JSON file from disk.
2. Unmarshal into `pricecalculator.CalculateRequest`.
3. Call `c.Calculate(req)`.
4. Marshal `pricecalculator.CalculateResult` to JSON and write to output file.
5. On error, write `{"error":"<message>"}` to the output file.

Replicate this flow in any other system that integrates the calculator.

---

## Availability Map — JSON Type Handling

The `availability` field on `PricingPeriod` is `map[string]interface{}`. When unmarshaling from JSON, Go's standard library maps:

| JSON value | Go type after unmarshal |
|---|---|
| `true` / `false` | `bool` |
| `"HH:MM-HH:MM"` | `string` |
| `["HH:MM-HH:MM", ...]` | `[]interface{}` (elements are `string`) |

Ensure your JSON encoder/decoder preserves these types. Do **not** pre-marshal availability values as nested JSON strings.

---

## `start_time` Field

- If `start_time` is **omitted** from the request, the calculator uses `time.Now()` in the **local timezone**.
- This means results are **non-deterministic** without `start_time`, because availability and time-based period restrictions are evaluated relative to wall-clock time.
- **Always provide `start_time`** when you need reproducible or testable results.

```json
{
  "start_time": "2026-04-03 09:00:00",
  "duration": 90,
  ...
}
```

---

## Timezone

- All times are parsed using **`time.Local`**.
- Ensure the process timezone (`TZ` environment variable or system timezone) is set correctly for your deployment environment before starting the process.
- There is no per-request timezone override — timezone is global to the process.

---

## Dependencies

| Dependency | Scope | Purpose |
|---|---|---|
| Go standard library | Runtime | All production code |
| `github.com/stretchr/testify` | Test only | `require` / `assert` helpers |

There are **no external runtime dependencies**. The `pkg/pricecalculator` library is self-contained.

