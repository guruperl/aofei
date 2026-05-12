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

- `[+]` `cmd/spread` attempts a best-effort Redis bootstrap on startup. When
  Redis and MySQL are reachable, static Redis cache families are mirrored into
  the configured spread directory before live NATS updates are consumed.
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
