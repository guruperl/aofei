# Retired milestone M40 - Redis Cache Availability And Route Efficiency

**Milestone.** M40
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M40.md
**Source specification.** memory-bank/milestone.md#m40---redis-cache-availability-and-route-efficiency
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M40 - Redis Cache Availability And Route Efficiency `[+]`

Remove the serving gap during full Redis cache refresh and avoid fetching and
decoding the complete middleman route cache on every eligible request.

Scope:

- Build static cache generations under shadow keys and atomically swap all
  related live families in one Redis transaction.
- Add a short-lived, single-flight controller snapshot for middleman routes.
- Raise the cache attribute-log scanner limit to the ledger limit.

Acceptance:

- Failed cache builds leave the previous live generation untouched, successful
  swaps cannot expose a partially deleted or mixed generation, and stale slot
  keys are removed atomically.
- Middleman route reads are memoized for a configurable interval and refresh
  errors do not fan out using expired routes.
- Attribute log lines up to 8 MiB are accepted.

Result:

- Full Redis refreshes build shadow families and atomically install one
  complete generation; failed builds preserve live data and successful swaps
  remove empty and obsolete families in the same transaction.
- Workers memoize decoded middleman routes for a configurable five-second
  default with context-aware single-flight refresh and short error caching.
- Attribute log scanning now accepts lines up to 8 MiB, and miniredis plus
  Docker smoke/serving-loop checks cover the replacement path.
- Reopened to serialize every mutating cache mode on one resource lock and let
  route-cache waiters retry after a canceled refresh leader.
- Review remediation completed with one writer lock, cancellation-aware waiter
  retry, and exact scanner/cap transaction coverage in CI.
- Reopened after the follow-up review found that the initiating HTTP request
  still owned and could cancel the shared route refresh.
- Follow-up remediation now runs the shared load under its own
  `middleman_timeout_ms` context while each caller waits independently.
````

## Status record

````markdown
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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Build full Redis static-cache generations under shadow keys. | `[+]` | |
| Atomically swap all live families and remove obsolete slot keys. | `[+]` | |
| Preserve route-only direct atomic refresh behavior. | `[+]` | |
| Add configured middleman route memoization with context-aware | `[+]` | |
| Add route-cache hit, miss, refresh, and error metrics. | `[+]` | |
| Raise the attribute-log scanner limit to 8 MiB. | `[+]` | |
| Add unit, Redis, and Docker serving-continuity coverage. | `[+]` | |
| Update cache/config/operator docs and memory-bank contracts. | `[+]` | |
| Run closeout verification and deep review. | `[+]` | |
| Use one mutation lock across all full and partial Redis cache writers. | `[+]` | |
| Make route-refresh waiters retry after a canceled refresh leader. | `[+]` | |
| Exercise frequency-cap and maximum-size attribute-log paths in CI. | `[+]` | |
| Run review-remediation verification and close the reopened milestone. | `[+]` | |
| Detach each shared route refresh from the initiating request and bound | `[+]` | |
| Cache the shared load result/error while letting each caller wait on | `[+]` | |
| Run follow-up review-remediation verification and close the reopened | `[+]` | |
| Failed builds leave live keys untouched and successful swaps expose one | `[+]` | |
| Empty families and obsolete slot keys disappear in the same atomic | `[+]` | |
| Route payloads are fetched at most once per configured interval under | `[+]` | |
| Attribute log lines up to 8 MiB parse without scanner failure. | `[+]` | |
| Initiator cancellation neither cancels the shared load nor fails other | `[+]` | |
| `GOWORK=off go test ./internal/jobs/cache ./match ./dsp` | `[+]` | |
| `GOWORK=off go test -race ./internal/jobs/cache ./match ./dsp` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off go vet ./...` | `[+]` | |
| `GOWORK=off staticcheck ./...` | `[+]` | |
| `./scripts/aofei-cache-smoke.sh` | `[+]` | |
| Eight concurrent Redis refresh and `TestServeBidSmoke` iterations. | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |
| `GOWORK=off go test -race ./dsp ./match ./internal/jobs/cache | `[+]` | |
| M39-M44 full Go, race, vet, staticcheck, docs, actionlint, sibling, | `[+]` | |
| Go 1.23.5 follow-up full/scoped race gates and canceled-initiator plus | `[+]` | |

````
