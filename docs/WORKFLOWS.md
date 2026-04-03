# WORKFLOWS.md — Developer Workflows for Building, Running, and Testing

## Prerequisites

- Go toolchain installed (see `go.mod` for required version)
- Working directory: project root (`/price_calculator`)

---

## Build

Compile the CLI binary to `bin/calculator`:

```zsh
go build -o bin/calculator ./cmd/calculator/
```

---

## Run

Execute the calculator against a JSON input file and write results to an output file:

```zsh
./bin/calculator -f example.json -o output.json
```

Inspect the output with `jq`:

```zsh
jq . output.json
```

**Input**: a JSON file matching the `CalculateRequest` schema (see [ARCHITECTURE.md](./ARCHITECTURE.md)).  
**Output**: a JSON file containing a `CalculateResult`; on error, the file will contain `{"error":"..."}`.

---

## Run Demo

```zsh
go run ./cmd/demo/
```

---

## Test

Run all tests once (no result caching):

```zsh
go test ./pkg/pricecalculator -count=1
```

With an explicit timeout (recommended for CI):

```zsh
go test ./pkg/pricecalculator -count=1 -timeout 30s
```

Run a single test by name:

```zsh
go test ./pkg/pricecalculator -count=1 -run TestCalculate
```

Run with verbose output:

```zsh
go test ./pkg/pricecalculator -count=1 -v
```

---

## Lint / Vet

```zsh
go vet ./...
```

---

## Pre-Commit Checklist

Run all three steps in order before committing. Do **not** commit if any step fails.

```zsh
go build -o bin/calculator ./cmd/calculator/ && \
go vet ./... && \
go test ./pkg/pricecalculator -count=1 -timeout 30s
```

---

## Quick Reference

| Task | Command |
|---|---|
| Build | `go build -o bin/calculator ./cmd/calculator/` |
| Run | `./bin/calculator -f example.json -o output.json` |
| Inspect output | `jq . output.json` |
| Test | `go test ./pkg/pricecalculator -count=1` |
| Test (with timeout) | `go test ./pkg/pricecalculator -count=1 -timeout 30s` |
| Vet | `go vet ./...` |
| All checks | build → vet → test |

