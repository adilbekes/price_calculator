# CONVENTIONS.md — Project-Specific Coding Conventions and Constraints

## Error Handling

- **Always use domain-specific error constructors** defined in `errors.go`:
  - `NewRequestError(...)` — for general request validation failures
  - `NewDurationError(...)` — for duration-related validation failures
  - `NewPeriodsError(...)` — for period-related validation failures
- **Never use `fmt.Errorf` directly** for user-facing errors.
- Errors wrap sentinel values so callers can use `errors.Is`:
  - `ErrInvalidRequest`
  - `ErrInvalidDuration`
  - `ErrInvalidPricingPeriods`

```go
// ✅ Correct
return NewDurationError("duration must be greater than 0")

// ❌ Wrong
return fmt.Errorf("duration must be greater than 0")
```

---

## No Global Mutable State

- `nowTime` is a **package-level `var` function** (not a constant) specifically to allow injection in tests.
- **Never call `time.Now()` directly** inside library code — always call `nowTime()`.

```go
// ✅ Correct
var nowTime = time.Now

func someFunc() time.Time {
    return nowTime()
}

// ❌ Wrong
func someFunc() time.Time {
    return time.Now()
}
```

---

## Regex Compilation

- **Never compile a regex inside a hot loop or a frequently-called function.**
- Compile once at **package level** or cache the compiled result.
- Note: `parseTimeHHMM` currently compiles its regex inside the function body — this is a known issue; do **not** replicate this pattern in new code.

```go
// ✅ Correct — compile once at package level
var timeRangeRe = regexp.MustCompile(`^\d{2}:\d{2}-\d{2}:\d{2}$`)

// ❌ Wrong — compiled on every call
func parse(s string) { re := regexp.MustCompile(`^\d{2}:\d{2}-\d{2}:\d{2}$`) ... }
```

---

## Period IDs

- Either **ALL** periods have an `id` (each must be unique), or **NONE** do.
- Mixed id/no-id is a validation error — catch it in `validateRequest`.

---

## Time Formats

| Context | Format | Example |
|---|---|---|
| `CalculateRequest.start_time` | `"2006-01-02 15:04:05"` (Go `time.DateTime`) | `"2026-04-03 09:00:00"` |
| `PricingPeriod.start_time` | `"HH:MM"` (24-hour, zero-padded) | `"09:00"`, `"14:30"` |
| Availability map keys | `"YYYY-MM-DD"` | `"2026-04-03"` |
| Availability time ranges | `"HH:MM-HH:MM"` | `"08:00-18:00"` |

---

## Availability Defaults

- A **missing date key** in an `availability` map means the period is **available all day** for that date.
- Explicitly set `false` to block a date entirely.

---

## Availability Array (Union Window)

When a date's value is a list of time-range strings:
- **Interval-availability checks**: union window is earliest start → latest end.
- **Point-in-time checks**: period is available if the time falls within **any** of the listed ranges.

---

## JSON Field Names

- Use **`snake_case`** for all JSON fields (`duration`, `start_time`, `price_step`, etc.).
- Use `omitempty` to omit zero/empty fields from serialized output.

```go
// ✅ Correct
type BreakdownItem struct {
    ID          string `json:"id,omitempty"`
    Duration    int    `json:"duration"`
    UsedPrice   int64  `json:"used_price"`
}
```

---

## No Panics in Library Code

- **All errors must be returned**, never panicked.
- `cmd/` entry points may `log.Fatal` on unrecoverable errors, but `pkg/pricecalculator` must never panic.

---

## Tests

- **Table-driven tests** wherever there are multiple input scenarios.
- Use **`require`** for assertions that should stop the test immediately on failure; use **`assert`** for non-fatal checks.
- Import from `github.com/stretchr/testify/require` and `github.com/stretchr/testify/assert`.
- Test files must be named `*_test.go` and placed **alongside** the file under test (not in a separate directory).
- Test-inject time via `nowTime` to avoid flaky time-dependent tests.

```go
// ✅ Correct table-driven test skeleton
func TestCalculate(t *testing.T) {
    tests := []struct {
        name    string
        req     CalculateRequest
        want    CalculateResult
        wantErr error
    }{
        // ...cases...
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            c := NewCalculator()
            got, err := c.Calculate(tc.req)
            if tc.wantErr != nil {
                require.ErrorIs(t, err, tc.wantErr)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tc.want, got)
        })
    }
}
```

