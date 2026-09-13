# Retired milestone M14 - Redis And Spread Cache Reliability

**Milestone.** M14
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M14.md
**Source specification.** memory-bank/milestone.md#m14---redis-and-spread-cache-reliability
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M14 - Redis And Spread Cache Reliability `[+]`

Harden the Redis/static spread cache path after the post-M13 cache review.

Scope:

- Make spread subscriptions receive cache subjects whose payload keys contain
  dots, especially publisher domains.
- Replace in-place spread file writes with atomic snapshot replacement.
- Let `cmd/spread` recover local files on startup when Redis/MySQL are
  available. O03's successor implementation compiles from MySQL under the
  shared cache lease and uses Redis only for ownership and sequencing.
- Make full Redis and spread refreshes remove stale static cache records.
- Recompute item-level RAdv cache refreshes from MySQL slot state rather than
  merging against local spread files.
- Add an in-process local static cache for local/spread bid serving, while
  keeping caps and uploaded audiences Redis-backed.

Acceptance:

- Focused command/package tests cover spread subject routing, reset/cleanup
  handling, creative file map keys, and the local bid path.
- Docs describe the Redis, NATS, file, and in-memory cache roles clearly.
````

## Status record

````markdown
# Status M14 - Redis And Spread Cache Reliability

## Goal

Resolve the post-M13 findings around Redis/static spread cache correctness,
local file IO, and local-mode bid-path cache reads.

## Carry-Forward Notes

- `[+]` 2026-05-12: The post-review concern about request-path filesystem
  generation checks in the local static cache was carried into M15 and resolved
  there by loading snapshots at controller startup and refreshing them through
  an explicit reload hook.

## Completed

- `[+]` Spread subscription now uses the NATS tail wildcard `>` so subjects with
  dotted publisher domains are delivered to `cmd/spread`.
  - Files: `cmd/spread/main.go`, `cmd/spread/main_test.go`.

- `[+]` Spread file writes now use temp-file, fsync, atomic rename, and directory
  fsync instead of truncating the live snapshot path.
  - Files: `cmd/spread/main.go`.

- `[+]` `cmd/spread` originally added a best-effort Redis bootstrap on startup.
  O03 preserves fresh-node recovery but now compiles the snapshot from MySQL
  under the shared cache lease; Redis supplies only ownership and the monotonic
  sequence so older cache payloads cannot displace a spread-only publication.
  - Files: `cmd/spread/main.go`.

- `[+]` Full cache refreshes now clear stale static state.
  - Redis refresh deletes `pubmap`, `audience`, `creative`, and existing
    `slot:*` keys before repopulation.
  - Spread refresh publishes `__reset__` family subjects before repopulation.
  - Files: `cmd/redis-cache/main.go`, `cmd/spread/main.go`.

- `[+]` Slot cleanup semantics are explicit.
  - `cleanup` suffix handling applies only to `slot` subjects.
  - `slot:<size_id>:cleanup` clears a size directory even when there are no
    nonempty slots to write.
  - Files: `cmd/spread/main.go`, `match/radv.go`, `cmd/spread/main_test.go`.

- `[+]` Item-level RAdv refreshes now recompute affected creative sizes from
  MySQL `proc_slot` output instead of merging with current Redis or local spread
  state. This removes stale local file state as an update input.
  - Files: `match/radv.go`.

- `[+]` Local/spread bid serving now uses an in-process static cache over spread
  files. Directory mtimes form the reload generation; publisher, slot RAdv,
  audience, and creative reads are served from memory between generations.
  Frequency caps and uploaded audience sets remain Redis-backed.
  Local static bids without caps/uploads can complete without Redis; bids that
  require those mutable families fail closed when Redis is unavailable.
  - Files: `dsp/local_cache.go`, `dsp/controller.go`,
    `dsp/local_cache_test.go`.

- `[+]` Fixed `CreativeMapFromIO` to key creative snapshots by creative id rather
  than creative size id.
  - Files: `match/creative.go`, `match/creative_cache_test.go`.

## Verification

- `[+]` `GOWORK=off go test ./cmd/redis-cache ./cmd/spread ./acl ./match ./dsp ./uploaded`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| 2026-05-12: The post-review concern about request-path filesystem | `[+]` | |
| Spread subscription now uses the NATS tail wildcard `>` so subjects with | `[+]` | |
| Spread file writes now use temp-file, fsync, atomic rename, and directory | `[+]` | |
| `cmd/spread` originally added a best-effort Redis bootstrap on startup. | `[+]` | |
| Full cache refreshes now clear stale static state. | `[+]` | |
| Slot cleanup semantics are explicit. | `[+]` | |
| Item-level RAdv refreshes now recompute affected creative sizes from | `[+]` | |
| Local/spread bid serving now uses an in-process static cache over spread | `[+]` | |
| Fixed `CreativeMapFromIO` to key creative snapshots by creative id rather | `[+]` | |
| `GOWORK=off go test ./cmd/redis-cache ./cmd/spread ./acl ./match ./dsp ./uploaded` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |

````
