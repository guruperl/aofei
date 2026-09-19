# Retired milestone M44 - Bid-Path Logging And Benchmark Cleanup

**Milestone.** M44
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M44.md
**Source specification.** memory-bank/milestone.md#m44---bid-path-logging-and-benchmark-cleanup
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M44 - Bid-Path Logging And Benchmark Cleanup `[+]`

Reduce avoidable bid-path logging/allocation work and measurement-gate any
future weighted-selection optimization.

Scope:

- Replace sugared request-path logs with structured logging and remove numbered
  progress messages and expected no-bid noise.
- Simplify local effective-CPM selection without changing auction results.
- Add parallel weighted-selection and representative bid-path benchmarks.

Acceptance:

- Bid responses and auction choices remain unchanged with materially quieter,
  structured request logging.
- Benchmarks record the selection and request-path baseline; `math/rand` is not
  replaced without measured evidence.

Result:

- Bid handling, auction fallback, callback setup, and adjacent tracker failures
  use structured `zap` fields; routine progress, success, NATS-unavailable, and
  expected no-bid messages no longer produce process-log noise.
- Local effective CPM validation retains the existing error metric and returns
  the already-materialized bid price directly.
- Parallel weighted-selection and successful two-impression local HTTP
  benchmarks establish an allocation/time baseline without changing the RNG.
- Deep review corrected the weighted-distribution test's bucket upper bounds,
  and all Aofei/pzdesign closeout gates pass under the documented toolchains.
````

## Status record

````markdown
# Status M44 - Bid-Path Logging And Benchmark Cleanup

State: `[+]` Completed

## Tasks

- `[+]` Replace bid request-path sugared logging with structured logging.
- `[+]` Remove numbered progress and expected no-bid log noise.
- `[+]` Simplify local effective-CPM return logic without behavior changes.
- `[+]` Add parallel selection and representative bid-path benchmarks.
- `[+]` Record benchmark results and run closeout verification/deep review.

## Acceptance

- `[+]` Existing bid and auction tests preserve responses and choices.
- `[+]` Request logging is structured and limited to actionable failures.
- `[+]` Benchmarks exist before any RNG implementation change.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./match`
- `[+]` `GOWORK=off go test ./dsp ./match -run '^$' -bench . -benchmem`
- `[+]` `GOWORK=off go test -race ./dsp ./match`
- `[+]` Full Aofei tests, vet, staticcheck, scoped race, docs, Docker cache
  smoke, and diff hygiene.
- `[+]` Full pzdesign tests, vet, staticcheck, template parsing, and diff
  hygiene.
- `[+]` Go 1.23.5 package compatibility and pinned staticcheck v0.5.1.
- `[+]` M39-M44 follow-up rerun: Go 1.23.5 benchmarks, full/scoped race gates,
  sibling verification, Docker cache smoke, docs/actionlint, and diff hygiene.

## Notes

- Findings: C3 and C4.
- C8 is rejected as stated: default top-level `math/rand` uses the runtime
  source without the claimed shared mutex, and this repository neither calls
  `rand.Seed` nor disables automatic seeding. Selection remains
  measurement-gated.
- The Go 1.26.1 linux/amd64 baseline records parallel selection at
  150.8-156.8 ns/op with zero allocations and the successful two-impression
  local HTTP path at 145.7-212.7 us/op with 121.4-123.9 KB and 1,062-1,066
  allocations per operation.
- Deep review corrected two ineffective upper-bound checks in the existing
  weighted-distribution test.
- No `evolution/` entry was added because logging policy and benchmark coverage
  do not change product, architecture, or public contracts.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Replace bid request-path sugared logging with structured logging. | `[+]` | |
| Remove numbered progress and expected no-bid log noise. | `[+]` | |
| Simplify local effective-CPM return logic without behavior changes. | `[+]` | |
| Add parallel selection and representative bid-path benchmarks. | `[+]` | |
| Record benchmark results and run closeout verification/deep review. | `[+]` | |
| Existing bid and auction tests preserve responses and choices. | `[+]` | |
| Request logging is structured and limited to actionable failures. | `[+]` | |
| Benchmarks exist before any RNG implementation change. | `[+]` | |
| `GOWORK=off go test ./dsp ./match` | `[+]` | |
| `GOWORK=off go test ./dsp ./match -run '^$' -bench . -benchmem` | `[+]` | |
| `GOWORK=off go test -race ./dsp ./match` | `[+]` | |
| Full Aofei tests, vet, staticcheck, scoped race, docs, Docker cache | `[+]` | |
| Full pzdesign tests, vet, staticcheck, template parsing, and diff | `[+]` | |
| Go 1.23.5 package compatibility and pinned staticcheck v0.5.1. | `[+]` | |
| M39-M44 follow-up rerun: Go 1.23.5 benchmarks, full/scoped race gates, | `[+]` | |

````
