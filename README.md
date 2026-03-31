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
# Using -d flag with duration only
./bin/calculator -d '{"duration":150,"mode":"RoundUp","periods":[{"duration":60,"price":1000},{"duration":120,"price":1800}]}'

# Using -d flag with duration and Unix timestamp
./bin/calculator -d '{"duration":150,"start_timestamp":1743379200,"mode":"RoundUp","periods":[{"duration":60,"price":1000}]}'

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
| `start_timestamp` | int64 | ❌ | — | Optional: Unix timestamp in seconds (for metadata/scheduling only; duration is always the authoritative value) |
| `periods` | array | ✅ | — | List of `{id, duration, price}` catalog periods |
| `mode` | string | ✅ | — | See [Pricing modes](#pricing-modes) |
| `duration_step` | int | ❌ | `5` | Duration is rounded up to this step before pricing |
| `min_duration` | int | ❌ | `5` | Requests below this are rejected with an error |
| `price_step` | int | ❌ | `1` | Total price is rounded up to the nearest multiple of this step (e.g. step `5`: 1084 → 1085) |


### Output

**Success** (exit 0) with timestamp:
```json
{"start_timestamp":1743379200,"end_timestamp":1743390000,"total":2300,"covered":180,"breakdown":[{"id":"2","duration":120,"price":1300,"quantity":1}]}
```

**Success** (exit 0) without timestamp:
```json
{"total":2300,"covered":180,"breakdown":[{"id":"2","duration":120,"price":1300,"quantity":1}]}
```

**Error** (exit 1):
```json
{"error":"duration must be greater than 0"}
```

**Output Fields:**
| Field | Type | Description |
|---|---|---|
| `start_timestamp` | int64 | Unix timestamp (seconds) - only included if provided in request |
| `end_timestamp` | int64 | Calculated as `start_timestamp + (covered_minutes * 60)` - only included if provided in request |
| `total` | int64 | Final price after all calculations and rounding |
| `covered` | int | Actual minutes covered by the pricing |
| `breakdown` | array | List of periods used with quantities |
| `error` | string | Error message (only in failure case) |

### Calling from Python

```python
import subprocess, json

result = subprocess.run(
    ["./bin/calculator", "-d", json.dumps({
        "duration": 150,
        "mode": "RoundUp",
        "periods": [
            {"duration": 60,  "price": 1000},
            {"duration": 120, "price": 1800},
        ]
    })],
    capture_output=True, text=True
)
data = json.loads(result.stdout)
if result.returncode != 0:
    raise RuntimeError(data["error"])
print(data["total"])
```

### Calling from PHP

```php
$input = json_encode([
    "duration" => 150,
    "mode" => "RoundUp",
    "periods" => [
        ["duration" => 60,  "price" => 1000],
        ["duration" => 120, "price" => 1800],
    ],
]);
$proc = proc_open('./bin/calculator', [['pipe','r'],['pipe','w'],['pipe','w']], $pipes);
fwrite($pipes[0], $input);
fclose($pipes[0]);
$data = json_decode(stream_get_contents($pipes[1]), true);
proc_close($proc);
echo $data['total'];
```

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
| 120 min | 1300 |
| 190 min | 2500 |

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
| `RoundUp` | 2300 | 180 | 120 min + 60 min = 1300 + 1000 (over-coverage to 180 min) |
| `ProrateMinimum` | 2300 | 180 | Same as `RoundUp` — prorate only applies below minimum |
| `ProrateAny` | 1800 | 150 | 120 min + prorate(60 min for 30 min) = 1300 + 500 (exact coverage) |
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
