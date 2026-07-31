# Status M40 - Redis Cache Availability And Route Efficiency

State: `[+]` Completed

## Tasks

- `[+]` Build full Redis static-cache generations under shadow keys.
- `[+]` Atomically swap all live families and remove obsolete slot keys.
- `[+]` Preserve route-only direct atomic refresh behavior.
- `[+]` Add configured middleman route memoization with context-aware
  single-flight refresh.
- `[+]` Add route-cache hit, miss, refresh, and error metrics.
- `[+]` Raise the attribute-log scanner limit to 8 MiB.
- `[+]` Add unit, Redis, and Docker serving-continuity coverage.
- `[+]` Update cache/config/operator docs and memory-bank contracts.
- `[+]` Run closeout verification and deep review.
- `[+]` Use one mutation lock across all full and partial Redis cache writers.
- `[+]` Make route-refresh waiters retry after a canceled refresh leader.
- `[+]` Exercise frequency-cap and maximum-size attribute-log paths in CI.
- `[+]` Run review-remediation verification and close the reopened milestone.
- `[+]` Detach each shared route refresh from the initiating request and bound
  it with `middleman_timeout_ms`.
- `[+]` Cache the shared load result/error while letting each caller wait on
  its own context.
- `[+]` Run follow-up review-remediation verification and close the reopened
  milestone.

## Acceptance

- `[+]` Failed builds leave live keys untouched and successful swaps expose one
  complete generation.
- `[+]` Empty families and obsolete slot keys disappear in the same atomic
  transaction that installs the new generation.
- `[+]` Route payloads are fetched at most once per configured interval under
  concurrent traffic; expired routes are not used after refresh failure.
- `[+]` Attribute log lines up to 8 MiB parse without scanner failure.
- `[+]` Initiator cancellation neither cancels the shared load nor fails other
  waiters, and timeout/error caching never authorizes expired routes.

## Verification

- `[+]` `GOWORK=off go test ./internal/jobs/cache ./match ./dsp`
- `[+]` `GOWORK=off go test -race ./internal/jobs/cache ./match ./dsp`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off go vet ./...`
- `[+]` `GOWORK=off staticcheck ./...`
- `[+]` `./scripts/aofei-cache-smoke.sh`
- `[+]` Eight concurrent Redis refresh and `TestServeBidSmoke` iterations.
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
- `[+]` `GOWORK=off go test -race ./dsp ./match ./internal/jobs/cache
  ./cmd/redis-cache`
- `[+]` M39-M44 full Go, race, vet, staticcheck, docs, actionlint, sibling,
  Docker cache-smoke, and diff-hygiene gates.
- `[+]` Go 1.23.5 follow-up full/scoped race gates and canceled-initiator plus
  shared-timeout/error-cache regressions.

## Notes

- Findings: B1, B2, and C6.
- Live Redis key names and cache payload shapes remain unchanged.
- A canceled single-flight leader is immediately retryable and is not cached as
  a worker-wide route failure.
- `github.com/alicebob/miniredis/v2` is a test-only dependency for deterministic
  transaction coverage.
- No `evolution/` entry was added because cache ownership and public contracts
  are unchanged.
- Reopened after the M39-M44 review found that mode-specific writer locks could
  overlap on shared shadow keys and that current waiters inherited a canceled
  route-refresh leader's error without retrying.
- Review remediation now uses one `aofei:redis-cache` lock for every mutating
  mode, retries waiters after canceled leaders without caching cancellation,
  and tests the exact 8 MiB scanner boundary plus cap transactions in CI.
- Reopened again because the initiating request still owned the shared load;
  the follow-up implementation gives the refresh an independent timeout.
- Follow-up closeout confirms one refresh populates/error-caches the controller
  snapshot even when its initiating caller is canceled.
