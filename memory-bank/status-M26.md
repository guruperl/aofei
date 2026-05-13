# Status M26 - Middle-Term Review

## Goal

Address the independent deep code quality and architecture review as an
explicit remediation milestone. M26 is the active tracking milestone for these
findings; closed milestones remain closed and are referenced only as lineage.

Source review: `~/.claude/plans/run-a-deep-analysis-prancy-glade.md`.

## Priority Order

1. Signature, replay, and SSRF hardening: C1, C2, C3.
2. Config validation and crash prevention: C4, H1.
3. Callback retry and singleton operations reliability: H4, H5.
4. Cache compatibility and serializer boundaries: H2, H3.
5. Observability baseline: H6, M3, M4, M7.
6. Test seams, ops bootstrap, and medium-term API boundaries: H7, H8, M5, M6.
7. Schema, auth migration, sample hygiene, and low-risk cleanup: C5, H9,
   M1-M2, M8-M11, L1-L5.

## Critical Findings

- `[+]` C1 - Middleman callback price signature scope.
  - Area: `dsp/middleman_callback.go`.
  - Lineage: M21 callback proxy and price reconciliation.
  - Disposition: M26 security fix. Middleman ledger math is server-owned and
    ignores incoming unsigned `auction_price`/`auction_currency` values.
  - Verification: regression tests prove tampered `/mid/win`, `/mid/bill`, and
    `/mid/loss` prices are rejected or ignored, and charge never exceeds the
    selected upstream bid price.

- `[+]` C2 - No timestamp or nonce for tracking and callback URLs.
  - Area: `dsp/tracking.go`, `dsp/middleman_callback.go`, win/loss handlers.
  - Lineage: M21 callback proxy; M24 idempotent callback retry.
  - Disposition: M26 security fix. Added strict `sig_ts` replay bounds for
    `/imp`, `/clk`, `/win`, `/loss`, and `/mid/*`; duplicate win/loss and
    middleman status records short-circuit when their Redis idempotency key is
    present.
  - Verification: tests cover expired signatures and duplicate win/loss
    short-circuiting.

- `[+]` C3 - SSRF in downstream middleman callback forwarding.
  - Area: `dsp/middleman_callback.go`.
  - Lineage: M21 downstream callback forwarding.
  - Disposition: M26 security fix. Live middleman forwarding and retry command
    forwarding reject unsafe callback targets before HTTP requests, with a
    guarded transport for production forwarding.
  - Verification: tests reject loopback, private, link-local, unspecified, and
    DNS-rebinding callback targets while allowing valid public HTTP(S) targets.

- `[+]` C4 - `Config.GetRedisDB` panics on truncated `ConnectArray`.
  - Area: `dsp/config.go`.
  - Lineage: config validation architecture gap.
  - Disposition: M26 reliability fix. `NewConfig`, `Validate`, `OpenDB`, and
    `GetRedisDB` now check `connect_array` before indexing.
  - Verification: config tests return actionable errors for missing or
    one-element `connect_array` values.

- `[ ]` C5 - SHA1 admin and agent password hashing.
  - Area: `../pzdesign` Genelet/Summer auth with Aofei sample config and schema.
  - Lineage: M11/M18 Summer/Genelet compatibility; pzdesign split.
  - Disposition: M26 migration design, likely multi-step because it affects
    existing stored credentials and login procedures.
  - Verification: auth tests cover bcrypt or argon2id login, legacy rotation,
    and stale-password reset behavior.

## High Findings

- `[+]` H1 - Add a `Config.Validate()` boundary.
  - Area: `dsp/config.go`, command startup paths.
  - Lineage: documented architecture gap.
  - Disposition: M26 reliability foundation. Added mode-aware validation for
    bid, cache, retry, spread, MaxMind, NATS, Redis, and database startup paths.
  - Verification: mode-specific config tests cover bid, cache, ledger, retry,
    spread, and MaxMind command needs.

- `[ ]` H2 - Version cache payloads.
  - Area: `match` Redis/spread payloads for demand, audience, and creative data.
  - Lineage: documented cache compatibility architecture gap; M19/M24 cache
    operations.
  - Disposition: M26 cache compatibility work.
  - Verification: decode tests reject unknown versions and prove current
    payloads round-trip.

- `[ ]` H3 - Replace cache serializer `interface{}` sinks.
  - Area: `match` cache writers.
  - Lineage: M19 cache job refactor.
  - Disposition: M26 maintainability work after H2 shape is clear.
  - Verification: compile-time `CacheSink` implementations for Redis and
    spread/NATS paths plus existing cache tests.

- `[ ]` H4 - Add singleton locks for operations commands.
  - Area: `cmd/redis-cache`, `cmd/ledger`, `cmd/mid-callback-retry`,
    `cmd/winloss`.
  - Lineage: M19 and M24 singleton job ownership.
  - Disposition: M26 operations reliability work.
  - Verification: command tests or integration tests prove second concurrent
    runner exits without processing.

- `[+]` H5 - Recover orphaned `mid_callback_retry` rows stuck in `Processing`.
  - Area: `internal/jobs/midcallback`.
  - Lineage: M24 durable callback retry.
  - Disposition: M26 reliability fix. Added `claimed_at`; retry workers reclaim
    stale `Processing` rows and clear the claim on success, retry, or abandon.
  - Verification: DB-backed tests cover stale claim reclaim, success, retry,
    and abandoned rows.

- `[ ]` H6 - Surface bid-path silent errors.
  - Area: `dsp` bid selection, audit publishing, middleman callback setup.
  - Lineage: observability architecture gap; M20-M25 bid-path behavior.
  - Disposition: M26 observability work.
  - Verification: tests or counters prove ECPM errors, audit publish failures,
    and middleman callback preparation failures are observable.

- `[ ]` H7 - Add controller test seams.
  - Area: `dsp.Controller` construction and dependencies.
  - Lineage: M21/M24 callback-store testing.
  - Disposition: M26 testability work. Extend existing controller options
    without changing production constructor behavior.
  - Verification: tests construct controllers with injected Redis, DB,
    callback store, and IP search substitutes.

- `[ ]` H8 - Share operational command bootstrap.
  - Area: `cmd/*` startup and connection lifecycle.
  - Lineage: M19 maintenance job package refactor.
  - Disposition: M26 maintainability work after H1 validation modes are
    defined.
  - Verification: command tests cover lifecycle, signal handling, and opt-outs
    for NATS and MaxMind.

- `[ ]` H9 - Sanitize sample Summer config secrets.
  - Area: `etc/summer.json` plus pzdesign/Summer docs.
  - Lineage: M11/M18 config compatibility; pzdesign split.
  - Disposition: M26 config hygiene work. Keep Aofei owning runtime config; do
    not move database schema/config ownership to pzdesign.
  - Verification: startup/config tests reject placeholder secrets where
    runtime secrets are required.

## Medium Findings

- `[ ]` M1 - Guard empty MaxMind subdivision slices.
  - Area: `maxmind/ipsearch.go`.
  - Disposition: M26 small correctness fix.
  - Verification: unit test with empty non-nil `Subdivisions`.

- `[ ]` M2 - Evaluate hot-path allocation pooling.
  - Area: `dsp` OpenRTB response and audit encoding.
  - Disposition: M26 performance investigation, implement only after
    measurement.
  - Verification: benchmark before and after any pooling change.

- `[ ]` M3 - Add audit drop observability.
  - Area: `dsp/audit.go`.
  - Disposition: M26 observability work, likely with M4.
  - Verification: tests or expvar/metrics checks show dropped audit count
    increments.

- `[ ]` M4 - Add a minimal metrics endpoint.
  - Area: `dsp` counters and `../pzdesign/cmd/unify` mux.
  - Disposition: M26 observability baseline. Prefer stdlib `expvar` unless a
    stronger metrics stack is selected later.
  - Verification: HTTP test exposes bid/no-bid, audit drop, middleman forward,
    and cache reload metrics.

- `[ ]` M5 - Isolate middleman feature gating.
  - Area: `dsp/controller.go`, `dsp/middleman.go`.
  - Lineage: M20-M25 middleman runtime.
  - Disposition: M26 medium-term refactor after critical callback hardening.
  - Verification: disabled middleman path is a no-op handler and existing
    disabled tests still pass.

- `[ ]` M6 - Stabilize cross-module Aofei to pzdesign API boundary.
  - Area: `match` types consumed by `../pzdesign/summer`.
  - Lineage: pzdesign split; module boundary architecture gap.
  - Disposition: M26 design/refactor track. Consider stable `api` types and
    conversion at the boundary.
  - Verification: pzdesign builds against stable exported API without importing
    internal runtime structs unnecessarily.

- `[ ]` M7 - Instrument local cache reload latency and size.
  - Area: `dsp/local_cache.go`.
  - Disposition: M26 observability work.
  - Verification: reload logs or metrics include duration, bytes, and entry
    counts.

- `[ ]` M8 - Clarify and test `dh` hour semantics.
  - Area: `dh`.
  - Disposition: M26 small documentation/test fix.
  - Verification: unit test documents the intended 1-based hour behavior.

- `[ ]` M9 - Standardize schema charset.
  - Area: `etc/step4_init.sql`.
  - Disposition: M26 schema hygiene, requires reset/load/check/diff.
  - Verification: schema diff proves intended `utf8mb4` charset changes only.

- `[ ]` M10 - Tune MySQL connection pools.
  - Area: `dsp.Config.GetRedisDB` and command modes.
  - Lineage: H1 config validation.
  - Disposition: M26 operations reliability work.
  - Verification: config tests cover default and mode-specific pool settings.

- `[ ]` M11 - Add ACL package documentation.
  - Area: `acl/doc.go`.
  - Disposition: M26 onboarding documentation.
  - Verification: `GOWORK=off go test ./acl` and doc review.

## Low Findings

- `[ ]` L1 - Improve controller test error assertions.
  - Area: `dsp/controller_test.go`.
  - Disposition: M26 test cleanup.
  - Verification: tests use named errors or `errors.Is`/`errors.As`.

- `[ ]` L2 - Add safety around destructive local script paths.
  - Area: `scripts/aofei-local.sh`.
  - Disposition: M26 tooling hardening.
  - Verification: script tests or dry runs cover confirmation and cleanup
    behavior.

- `[ ]` L3 - Add DB-backed job schema smoke tests.
  - Area: ledger and middleman retry jobs.
  - Disposition: M26 test coverage improvement.
  - Verification: Docker MySQL smoke validates schema shape for key job
    tables.

- `[ ]` L4 - Replace magic ledger timestamps in tests.
  - Area: `internal/jobs/ledger/ledger_test.go`.
  - Disposition: M26 test cleanup.
  - Verification: tests use fixed `time.Date` anchors.

- `[ ]` L5 - Re-audit MySQL dependency and pool tuning for new ops commands.
  - Area: command bootstrap and config.
  - Disposition: tracked through M10/H8; keep as checklist item for future
    commands.
  - Verification: any new DB-using command declares mode and pool needs.

## Reviewed Non-Issues

- `[X]` Local static cache map swap concurrency.
  - The review confirmed the `RWMutex` and map-pointer swap pattern is safe.

- `[X]` Double-lock risk between local cache locks.
  - The review confirmed independent locks are acquired in fixed order.

- `[X]` Nil inner map read in MaxMind state lookup.
  - Go safely returns zero value for read-indexing a nil inner map. The real
    actionable issue is M1.

## Verification For M26 Status Creation

- `[X]` `./scripts/aofei-doc-check.sh`
- `[X]` `git diff --check`

## Verification For M26 Security Tranche

- `[X]` `GOWORK=off go test ./dsp ./internal/jobs/midcallback ./internal/safehttp ./cmd/mid-callback-retry ./cmd/redis-cache ./cmd/maxmind ./cmd/spread ./cmd/nats-client`
- `[X]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load && ./scripts/aofei-local.sh check-sql && ./scripts/aofei-local.sh diff-schema`
- `[X]` `GOWORK=off go test ./...`
- `[X]` `./scripts/aofei-local.sh reset-sample && ./scripts/aofei-cache-smoke.sh`
- `[X]` `GOWORK=off staticcheck ./dsp ./internal/jobs/midcallback ./internal/safehttp ./cmd/redis-cache ./cmd/mid-callback-retry ./cmd/maxmind ./cmd/spread ./cmd/nats-client`
- `[X]` `./scripts/aofei-doc-check.sh`
- `[X]` `git diff --check`
