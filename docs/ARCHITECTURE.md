# ARCHITECTURE.md — System Architecture, Component Boundaries, and Data Flow

## Purpose

Given a `CalculateRequest`, produce a `CalculateResult` with the optimal (cheapest) price breakdown for the requested rental duration.

---

## Package Layout

| Path | Role |
|---|---|
| `pkg/pricecalculator/` | Core library — all business logic |
| `cmd/calculator/main.go` | CLI entry point (reads JSON → writes JSON) |
| `cmd/demo/main.go` | Demo entry point for manual exploration |

---

## Key Types (`types.go`)

### `CalculateRequest`

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `duration` | `int` (minutes) | ✅ | — | Total rental duration |
| `start_time` | `string` (`"YYYY-MM-DD HH:MM:SS"`) | ❌ | `time.Now()` | If omitted, uses local wall clock |
| `duration_step` | `int` (minutes) | ❌ | `5` | Duration is normalized to this step |
| `min_duration` | `int` (minutes) | ❌ | `5` | Minimum chargeable duration |
| `price_step` | `int64` | ❌ | `1` (no rounding) | Final price is rounded up to nearest multiple |
| `mode` | `PricingMode` (string/int) | ✅ | — | Controls proration/rounding strategy |
| `periods` | `[]PricingPeriod` | ✅ | — | Available pricing periods |

### `PricingPeriod`

| Field | Type | Required | Notes |
|---|---|---|---|
| `id` | `string` | ❌ | If any period has an `id`, all must have unique ids |
| `duration` | `int` (minutes) | ✅ | Must be > 0 |
| `price` | `int64` | ✅ | Price for one full use of this period |
| `start_time` | `string` (`"HH:MM"`) | ❌ | Fixed clock start; period usable only at/after this time each day |
| `availability` | `map[string]interface{}` | ❌ | Date-keyed availability map (see below) |

**Availability map values** (keys are `"YYYY-MM-DD"` date strings):

| Value type | Meaning |
|---|---|
| *(missing key)* | Available all day for that date |
| `true` | Available all day |
| `false` | Not available that day |
| `"HH:MM-HH:MM"` | Available during this time range |
| `["HH:MM-HH:MM", ...]` | Available during any of these ranges (union: earliest start → latest end) |

### `CalculateResult`

| Field | Type | Notes |
|---|---|---|
| `start_time` | `string` | Echoed from request if provided; omitted otherwise |
| `end_time` | `string` (`"YYYY-MM-DD HH:MM:SS"`) | `start_time + covered minutes`; omitted if no `start_time` in request |
| `total` | `int64` | Total price (after `price_step` rounding) |
| `covered` | `int` | Total minutes covered by the breakdown |
| `breakdown` | `[]BreakdownItem` | Per-period usage details |

### `BreakdownItem`

| Field | Notes |
|---|---|
| `id` | Period id (if periods use ids) |
| `duration` | Full period duration |
| `used_duration` | Minutes from this period used in solution |
| `price` | Full period price |
| `used_price` | Price charged for `used_duration` |
| `quantity` | How many full uses of this period |
| `start_time`, `end_time` | Set only when the period has a `start_time` field |

---

## Pricing Modes (`PricingMode`)

| Value | Name | Behaviour |
|---|---|---|
| `0` | `RoundUp` | Round requested duration **up** to nearest period boundary |
| `1` | `ProrateMinimum` | Prorate only when duration is below the minimum period |
| `2` | `ProrateAny` | Prorate any combination of periods |
| `3` | `RoundUpMinimumAndProrateAny` | Round up below minimum; prorate above |

---

## Data Flow (`calculator.go`)

```
CalculateRequest
      │
      ▼
1. Parse start_time (or time.Now())
      │
      ▼
2. validateRequest
   └─ duration > 0, periods valid, mode valid, availability valid
      │
      ▼
3. Normalize duration → nearest duration_step multiple
      │
      ▼
4. Filter periods by availability for [start_time, start_time+duration]
      │
      ▼
5. Detect hasTimeBasedRestrictions
   (any period has start_time or string/array availability values)
      │
      ├─── RoundUp + NO time-based restrictions ──▶ pick single cheapest period
      │                                             that covers full duration
      │
      └─── Otherwise ──▶ Optimizer
            ├─ optimizeTimelineAware         (time-based restrictions present)
            ├─ optimizePrice                 (RoundUp / ProrateMinimum, standard)
            ├─ optimizePriceWithOptionalProration  (ProrateAny / RoundUpMinimumAndProrateAny)
            └─ priceBelowMinimum             (duration < min_duration)
      │
      ▼
6. Apply price_step rounding (round total up to nearest multiple)
      │
      ▼
7. Set start_time / end_time on result (only if request had start_time)
      │
      ▼
CalculateResult
```

---

## Component Boundaries

| File | Responsibility |
|---|---|
| `types.go` | All shared data types and constants |
| `calculator.go` | Orchestration; top-level `Calculate` entry point |
| `validator.go` | `validateRequest` — structural and semantic validation |
| `parser.go` | Time/availability string parsing helpers |
| `optimizer.go` | DP and timeline-aware pricing optimizers |
| `format.go` | Result formatting (end_time, breakdown assembly) |
| `errors.go` | Domain error types and sentinel values |

