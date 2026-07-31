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
