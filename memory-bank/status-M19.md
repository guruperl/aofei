# Status M19 - Maintenance Job Package Refactor

## Goal

Refactor cache and ledger logic for reuse, keep Redis cache refresh as a
singleton standalone operational job, and keep ledger on the log aggregation
node.

## Completed

- `[+]` Reusable job packages.
  - Added `internal/jobs/cache` for Redis/spread/all cache refresh, cache read,
    mode validation, and pubmap attribute-log updates.
  - Added `internal/jobs/ledger` for interval ledger, daily ledger, and
    missing-input handling.

- `[+]` Thin command wrappers.
  - `cmd/redis-cache` keeps its existing flags and delegates to the cache job
    package.
  - `cmd/ledger` keeps interval/daily CLI behavior and delegates to the ledger
    job package.

- `[+]` `unify` boundary preserved.
  - No cache or ledger scheduler runs inside `cmd/unify`.
  - UI and ADX HTTP service nodes do not need per-node cache/ledger job config.

- `[+]` Singleton operational placement.
  - Redis cache refresh remains a singleton cron/timer job on one dedicated
    cache-maintenance node, not an embedded `unify` job.
  - Ledger remains a singleton cron/timer job on the log aggregation node where
    the complete `log_winloss/winloss.<stamp>` stream is available, not an
    embedded `unify` job.
  - `cmd/nats-client` runs as a separate systemd service from `cmd/unify`.
  - `cmd/spread` remains a separate service only on nodes that need spread disk
    snapshots.

## Carry Forward

- `[ ]` Middleman bidder runtime moves to M20.

## Verification

- `[+]` `GOWORK=off go test ./internal/jobs/cache ./internal/jobs/ledger ./cmd/redis-cache ./cmd/ledger ./cmd/unify`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache ./cmd/ledger ./internal/jobs/cache ./internal/jobs/ledger`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
