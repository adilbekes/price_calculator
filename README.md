# Price Calculator

This package calculates the minimum rental price based on time periods.

## Features
- Supports multiple pricing periods
- Finds optimal price combination
- Allows over-coverage of time
- Supports exact-coverage proration modes
- Normalizes requested duration by a configurable step
- Enforces a minimum requested duration
- Validates input

## Example
See `cmd/demo/main.go`

## Usage as a Binary

The calculator is available as a standalone binary callable from any language via **JSON on stdin → JSON on stdout**.

### Build

```bash
go build -o bin/calculator ./cmd/calculator/
```

### Run

```bash
# Using -d flag with duration only (uses the current local datetime as start_time)
./bin/calculator -d '{"duration":150,"mode":"RoundUp","periods":[{"duration":60,"price":1000},{"duration":120,"price":1800}]}'

# Using -d flag with period availability (date and optional time range)
./bin/calculator -d '{"duration":150,"mode":"RoundUp","periods":[{"id":"p1","duration":60,"price":1000,"availability":{"2026-03-31":true,"2026-04-01":"10:00-18:00"}},{"id":"p2","duration":120,"price":1800}]}'

# Using -d flag with duration and explicit datetime
./bin/calculator -d '{"duration":150,"start_time":"2026-04-01 12:00:00","mode":"RoundUp","periods":[{"duration":60,"price":1000}]}'

# Using -d flag for 30 min request
./bin/calculator -d '{"duration":30,"mode":"RoundUpMinimumAndProrateAny","periods":[{"duration":60,"price":1000},{"duration":120,"price":1800}]}'


# Using -f flag (input file)
./bin/calculator -f request.json

# Using -f and -o flags (input file and output file)
./bin/calculator -f request.json -o result.json

# Using stdin (piped input)
echo '{"duration":150,"mode":"RoundUp","periods":[{"duration":60,"price":1000}]}' | ./bin/calculator
```

### CLI Flags

| Flag | Type | Description | Example |
|---|---|---|---|
| `-d` | string | JSON request as inline string | `-d '{"duration":150,...}'` |
| `-f` | string | JSON request file path | `-f request.json` |
| `-o` | string | JSON output file path (optional; default: stdout) | `-o result.json` |

**Notes:**
- `-d` and `-f` are mutually exclusive; cannot use both at the same time
- If neither `-d` nor `-f` is provided, input is read from stdin
- If `-o` is not provided, output goes to stdout
- Errors are written to the same destination as success output (stdout or `-o` file)

### Input

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `duration` | int | ✅ | — | **Required:** Requested rental duration in minutes |
| `start_time` | string | ❌ | Current local datetime | Optional: datetime string in `YYYY-MM-DD HH:MM:SS` format; if not provided, current time is used |
| `periods` | array | ✅ | — | List of `{id, duration, price}` catalog periods |
| `mode` | string | ✅ | — | See [Pricing modes](#pricing-modes) |
| `duration_step` | int | ❌ | `5` | Duration is rounded up to this step before pricing |
| `min_duration` | int | ❌ | `5` | Requests below this are rejected with an error |
| `price_step` | int | ❌ | `1` | Total price is rounded up to the nearest multiple of this step (e.g. step `5`: 1084 → 1085) |

### Period Fields

Each object in `periods` supports these fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | string | ❌ | Optional unique identifier (if any period has `id`, all periods must have unique `id`) |
| `duration` | int | ✅ | Catalog duration in minutes |
| `price` | int64 | ✅ | Catalog price |
| `start_time` | string | ❌ | Period start-of-day time in `HH:MM` format (for example, `"09:00"`) |
| `availability` | object | ❌ | Per-date availability map (see below) |

`period.start_time` semantics:
- It is a time-of-day anchor for that period, not a full datetime.
- A period with `start_time` is unavailable before that time on a given day.
- The period defines a daily window from `start_time` to `start_time + duration`.
- Requests can be split across periods when a window ends (for example, one period until 18:00, another after 18:00).

### Period Availability

Each period in the `periods` array can optionally include an `availability` object that specifies:

1. **Date-based availability** — Map of dates (YYYY-MM-DD) to availability status
2. **Time ranges** — Optional time range for each available date (HH:MM-HH:MM format)

#### Availability Values:
| Value | Type | Meaning |
|---|---|---|
| `true` | boolean | Available all day (00:00-23:59) |
| `false` | boolean | Not available |
| `"HH:MM-HH:MM"` | string | Available during this time range (e.g., `"10:00-18:00"`) |

#### Example:
```json
{
  "id": "period_1",
  "duration": 60,
  "price": 1000,
  "availability": {
    "2026-03-31": true,           // Available all day
    "2026-04-01": "10:00-18:00",  // Available 10:00-18:00
    "2026-04-02": false,          // Not available
    "2026-04-03": "22:00-23:59"   // Late-evening availability
  }
}
```

**Rules:**
- If an `availability` object is provided, it must include **every date touched by the requested interval** (`start_time` through request end)
- Missing dates in the availability map are treated as an **invalid request configuration**
- Time ranges must stay within the same day; overnight ranges such as `"22:00-02:00"` are rejected
- If no `availability` object is provided, the period is available all times
- If `period.start_time` is set, the period is only usable at or after that time, and only within its daily `start_time + duration` window
- Unavailable periods are **skipped** during calculation; the request still succeeds if the remaining periods can satisfy it
- If all periods are unavailable for the requested time, the calculator returns `no pricing solution found`

#### Example with period `start_time`
```json
{
  "duration": 540,
  "start_time": "2026-04-01 10:00:00",
  "mode": "RoundUp",
  "periods": [
    {
      "id": "day_window",
      "duration": 540,
      "price": 4000,
      "start_time": "09:00",
      "availability": {"2026-04-01": true}
    },
    {
      "id": "hourly",
      "duration": 60,
      "price": 1000,
      "availability": {"2026-04-01": true}
    }
  ]
}
```

In this example, `day_window` can only be used inside `09:00-18:00` on that day. A request continuing past 18:00 is completed by other available periods.


### Output

**Success** (exit 0) with datetime:
```json
{"start_time":"2026-04-01 20:00:00","end_time":"2026-04-01 23:00:00","total":2300,"covered":180,"breakdown":[{"id":"2","duration":120,"price":1800,"quantity":1}]}
```

**Success** (exit 0) without start datetime:
```json
{"total":2300,"covered":180,"breakdown":[{"id":"2","duration":120,"price":1800,"quantity":1}]}
```

**Error** (exit 1):
```json
{"error":"duration must be greater than 0"}
```

**Output Fields:**
| Field | Type | Description |
|---|---|---|
| `start_time` | string | Request datetime in `YYYY-MM-DD HH:MM:SS` format - only included if provided in request |
| `end_time` | string | Calculated request end datetime in `YYYY-MM-DD HH:MM:SS` format - only included if `start_time` was provided |
| `total` | int64 | Final price after all calculations and rounding |
| `covered` | int | Actual minutes covered by the pricing |
| `breakdown` | array | List of periods used with quantities |
| `error` | string | Error message (only in failure case) |

## Pricing modes

The `mode` field accepts the string name (or integer 0–3).

| Mode | String value | Int |
|---|---|---|
| `PricingModeRoundUp` | `"RoundUp"` | `0` |
| `PricingModeProrateMinimum` | `"ProrateMinimum"` | `1` |
| `PricingModeProrateAny` | `"ProrateAny"` | `2` |
| `PricingModeRoundUpMinimumAndProrateAny` | `"RoundUpMinimumAndProrateAny"` | `3` |

- `PricingModeRoundUp`: if the request is below the minimum period, round up to the cheapest minimum-duration period.
- `PricingModeProrateMinimum`: if the request is below the minimum period, prorate the cheapest minimum-duration period.
- `PricingModeProrateAny`: for any request size, compare the cheapest normal coverage with a result that combines full periods plus one prorated remainder from the minimum-duration period, and return the cheaper option.
- `PricingModeRoundUpMinimumAndProrateAny`: round up when the request is below the minimum period, otherwise compare normal coverage with any-range proration from the minimum-duration period and return the cheaper option.

### Examples

Periods used in both scenarios:

| Duration | Price |
|---|---|
| 60 min | 1000 |
| 120 min | 1800 |
| 180 min | 2500 |

**Scenario A — 30 min requested** *(below the minimum 60 min period)*

| Mode | Total Price | Covered Minutes | What happened |
|---|---|---|---|
| `RoundUp` | 1000 | 60 | Rounded up to the cheapest minimum period (60 min) |
| `ProrateMinimum` | 500 | 30 | Prorated the minimum period: 30/60 × 1000 = 500 |
| `ProrateAny` | 500 | 30 | Same as `ProrateMinimum` when below minimum |
| `RoundUpMinimumAndProrateAny` | 1000 | 60 | Rounds up when below minimum (same as `RoundUp`) |

**Scenario B — 150 min requested** *(above the minimum period)*

| Mode | Total Price | Covered Minutes | What happened |
|---|---|---|---|
| `RoundUp` | 2300 | 180 | 120 min + 60 min = 1800 + 1000 (over-coverage to 180 min) |
| `ProrateMinimum` | 2300 | 180 | Same as `RoundUp` — prorate only applies below minimum |
| `ProrateAny` | 1800 | 150 | 120 min + prorate(60 min for 30 min) = 1800 + 500 (exact coverage) |
| `RoundUpMinimumAndProrateAny` | 1800 | 150 | Same as `ProrateAny` when at or above minimum |

## Requested duration rules
- `duration_step` is optional and defaults to `5` when omitted.
- `duration` is rounded up to the nearest step before pricing. Example: `59 -> 60` with the default step.
- `min_duration` is optional and defaults to `5` when omitted.
- If the raw requested duration is below the minimum allowed duration, the calculator returns `ErrInvalidDuration`.

## Pricing period rules
- Multiple periods may share the same `duration` when their `price` differs.
- Exact duplicate periods with the same `duration` and `price` are rejected.

## Breakdown semantics
- `breakdown` items describe the source pricing periods used.
- For prorated results, `BreakdownItem.duration` and `BreakdownItem.price` still show the full catalog period.
- The actual charged amount is reflected by `total`, and the actual covered time is reflected by `covered`.
- If the same catalog period is used multiple times, it is shown once in `Breakdown` with an aggregated `Quantity`.
