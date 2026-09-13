# Retired milestone M19 - Maintenance Job Package Refactor

**Milestone.** M19
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M19.md
**Source specification.** memory-bank/milestone.md#m19---maintenance-job-package-refactor
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M19 - Maintenance Job Package Refactor `[+]`

Refactor cache and ledger jobs for reuse, keep Redis cache population as a
singleton scheduled command, and keep ledger on the log aggregation node.

Scope:

- Refactor `cmd/redis-cache` and `cmd/ledger` logic into reusable internal job
  packages while keeping the standalone command flags.
- Keep Redis cache refresh, ledger, `cmd/nats-client`, `cmd/spread`, and
  `cmd/winloss` separate commands.

Acceptance:

- `cmd/unify` behavior remains HTTP UI and ADX serving only.
- Redis cache refresh remains a singleton cron/timer job on one dedicated node.
- Ledger runs only on the node where `cmd/nats-client` aggregates win/loss log
  files.
- Standalone cache and ledger commands remain available for cron, timers, and
  manual operation.
````

## Status record

````markdown
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

- `[+]` Middleman bidder runtime moved to and was completed in M20.

## Verification

- `[+]` `GOWORK=off go test ./internal/jobs/cache ./internal/jobs/ledger ./cmd/redis-cache ./cmd/ledger ./cmd/unify`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache ./cmd/ledger ./internal/jobs/cache ./internal/jobs/ledger`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Reusable job packages. | `[+]` | |
| Thin command wrappers. | `[+]` | |
| `unify` boundary preserved. | `[+]` | |
| Singleton operational placement. | `[+]` | |
| Middleman bidder runtime moved to and was completed in M20. | `[+]` | |
| `GOWORK=off go test ./internal/jobs/cache ./internal/jobs/ledger ./cmd/redis-cache ./cmd/ledger ./cmd/unify` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache ./cmd/ledger ./internal/jobs/cache ./internal/jobs/ledger` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |

````
