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

- `[+]` C5 - SHA1 admin and agent password hashing.
  - Area: `../pzdesign` Genelet/Summer auth with Aofei sample config and schema.
  - Lineage: M11/M18 Summer/Genelet compatibility; pzdesign split.
  - Disposition: M26 reconciliation. Active Summer config uses Genelet
    `Password_hash` with stored bcrypt hashes; stale SHA1-era docs were
    removed and checked-in examples are tested for no SHA1 login SQL.
  - Verification: `GOWORK=off go test ./etc` plus pzdesign Genelet/Summer tests.

## High Findings

- `[+]` H1 - Add a `Config.Validate()` boundary.
  - Area: `dsp/config.go`, command startup paths.
  - Lineage: documented architecture gap.
  - Disposition: M26 reliability foundation. Added mode-aware validation for
    bid, cache, retry, spread, MaxMind, NATS, Redis, and database startup paths.
  - Verification: mode-specific config tests cover bid, cache, ledger, retry,
    spread, and MaxMind command needs.

- `[+]` H2 - Version cache payloads.
  - Area: `match` Redis/spread payloads for demand, audience, and creative data.
  - Lineage: documented cache compatibility architecture gap; M19/M24 cache
    operations.
  - Disposition: M26 cache compatibility fix. RAdvs, audience, and creative
    payloads now carry typed version envelopes while new readers still decode
    legacy unversioned payloads.
  - Verification: cache payload tests reject unknown versions and prove
    versioned/legacy round-trips.

- `[+]` H3 - Replace cache serializer `interface{}` sinks.
  - Area: `match` cache writers.
  - Lineage: M19 cache job refactor.
  - Disposition: M26 maintainability fix. Added typed `CacheSink`
    implementations for Redis and spread/NATS static cache writers.
  - Verification: `GOWORK=off go test ./match`.

- `[+]` H4 - Add singleton locks for operations commands.
  - Area: `cmd/redis-cache`, `cmd/ledger`, `cmd/mid-callback-retry`,
    `cmd/winloss`.
  - Lineage: M19 and M24 singleton job ownership.
  - Disposition: M26 operations reliability fix. Mutating redis-cache, ledger,
    mid-callback-retry, and winloss runs acquire Redis singleton locks; read and
    dry-run paths skip locks.
  - Verification: command packages build with shared lock bootstrap.

- `[+]` H5 - Recover orphaned `mid_callback_retry` rows stuck in `Processing`.
  - Area: `internal/jobs/midcallback`.
  - Lineage: M24 durable callback retry.
  - Disposition: M26 reliability fix. Added `claimed_at`; retry workers reclaim
    stale `Processing` rows and clear the claim on success, retry, or abandon.
  - Verification: DB-backed tests cover stale claim reclaim, success, retry,
    and abandoned rows.

- `[+]` H6 - Surface bid-path silent errors.
  - Area: `dsp` bid selection, audit publishing, middleman callback setup.
  - Lineage: observability architecture gap; M20-M25 bid-path behavior.
  - Disposition: M26 observability fix. Added expvar counters for bid outcomes,
    ECPM errors, audit drops/publish errors, and middleman callback failures.
  - Verification: `GOWORK=off go test ./dsp`.

- `[+]` H7 - Add controller test seams.
  - Area: `dsp.Controller` construction and dependencies.
  - Lineage: M21/M24 callback-store testing.
  - Disposition: M26 testability fix. Controller options now accept injected
    Redis, DB, NATS, IP search, HTTP client, logger, callback guard, and
    callback store seams without changing production defaults.
  - Verification: controller option tests cover dependency injection.

- `[+]` H8 - Share operational command bootstrap.
  - Area: `cmd/*` startup and connection lifecycle.
  - Lineage: M19 maintenance job package refactor.
  - Disposition: M26 maintainability fix. Added shared signal context and Redis
    singleton lock bootstrap used by mutating operational commands.
  - Verification: command packages build.

- `[+]` H9 - Sanitize sample Summer config secrets.
  - Area: `etc/summer.json` plus pzdesign/Summer docs.
  - Lineage: M11/M18 config compatibility; pzdesign split.
  - Disposition: M26 config hygiene fix. Renamed the checked-in Summer example
    to `etc/summer.example.json` and replaced local runtime secrets with
    placeholders while keeping generated `etc/summer.local.json`.
  - Verification: `./scripts/aofei-doc-check.sh` guards the example.

## Medium Findings

- `[+]` M1 - Guard empty MaxMind subdivision slices.
  - Area: `maxmind/ipsearch.go`.
  - Disposition: M26 correctness fix. `CreatePzGeo` now checks
    `len(Subdivisions) > 0` before indexing.
  - Verification: `GOWORK=off go test ./maxmind`.

- `[+]` M2 - Evaluate hot-path allocation pooling.
  - Area: `dsp` OpenRTB response and audit encoding.
  - Disposition: M26 measurement baseline. Added a bid response marshal
    benchmark and deferred pooling until measurements justify the extra
    complexity.
  - Verification: `GOWORK=off go test ./dsp`.

- `[+]` M3 - Add audit drop observability.
  - Area: `dsp/audit.go`.
  - Disposition: M26 observability fix. Audit enqueue, drop, and publish error
    counters are exported via expvar.
  - Verification: existing audit drop test plus `GOWORK=off go test ./dsp`.

- `[+]` M4 - Add a minimal metrics endpoint.
  - Area: `dsp` counters and `../pzdesign/cmd/unify` mux.
  - Disposition: M26 observability baseline. `../pzdesign/cmd/unify` registers
    stdlib `/debug/vars`.
  - Verification: `GOWORK=off go test ./cmd/unify` in pzdesign.

- `[+]` M5 - Isolate middleman feature gating.
  - Area: `dsp/controller.go`, `dsp/middleman.go`.
  - Lineage: M20-M25 middleman runtime.
  - Disposition: M26 refactor. Added a middleman runtime interface with a
    disabled no-op runtime while preserving existing active behavior.
  - Verification: `GOWORK=off go test ./dsp`.

- `[+]` M6 - Stabilize cross-module Aofei to pzdesign API boundary.
  - Area: `match` types consumed by `../pzdesign/summer`.
  - Lineage: pzdesign split; module boundary architecture gap.
  - Disposition: M26 boundary refactor. Added `adminapi` as the Summer-facing
    Aofei facade and switched pzdesign Summer packages to it; `cmd/unify`
    remains the DSP service integration point.
  - Verification: pzdesign Summer and unify targeted tests pass.

- `[+]` M7 - Instrument local cache reload latency and size.
  - Area: `dsp/local_cache.go`.
  - Disposition: M26 observability fix. Local static cache reloads export last
    duration and entry count via expvar.
  - Verification: `GOWORK=off go test ./dsp`.

- `[+]` M8 - Clarify and test `dh` hour semantics.
  - Area: `dh`.
  - Disposition: M26 documentation/test fix. `dh` now documents and tests the
    legacy 1-based fullhour bucket.
  - Verification: `GOWORK=off go test ./dh`.

- `[+]` M9 - Standardize schema charset.
  - Area: `etc/step4_init.sql`.
  - Disposition: M26 schema hygiene fix. Active baseline now uses `utf8mb4`
    with `utf8mb4_0900_ai_ci` collation.
  - Verification: schema reset/load/check/diff required before milestone close.

- `[+]` M10 - Tune MySQL connection pools.
  - Area: `dsp.Config.GetRedisDB` and command modes.
  - Lineage: H1 config validation.
  - Disposition: M26 operations reliability fix. Added DB pool config keys and
    mode-aware defaults for service and singleton job profiles.
  - Verification: config tests cover retry defaults and overrides.

- `[+]` M11 - Add ACL package documentation.
  - Area: `acl/doc.go`.
  - Disposition: M26 onboarding documentation.
  - Verification: `GOWORK=off go test ./acl` and doc review.

## Low Findings

- `[+]` L1 - Improve controller test error assertions.
  - Area: `dsp/controller_test.go`.
  - Disposition: M26 test cleanup. Controller tests now cover injected seams and
    continue to use explicit error behavior for callback/idempotency paths.
  - Verification: `GOWORK=off go test ./dsp`.

- `[+]` L2 - Add safety around destructive local script paths.
  - Area: `scripts/aofei-local.sh`.
  - Disposition: M26 tooling hardening. Destructive local script commands now
    refuse custom Docker/database targets unless explicitly allowed.
  - Verification: shell syntax/doc checks.

- `[+]` L3 - Add DB-backed job schema smoke tests.
  - Area: ledger and middleman retry jobs.
  - Disposition: M26 coverage improvement. Added a DB-backed schema smoke for
    retry and ledger job tables, skipped when local DB is unavailable.
  - Verification: `GOWORK=off go test ./internal/jobs/ledger`.

- `[+]` L4 - Replace magic ledger timestamps in tests.
  - Area: `internal/jobs/ledger/ledger_test.go`.
  - Disposition: M26 test cleanup. Ledger DB tests use fixed `time.Date`
    anchors for active/timely values.
  - Verification: `GOWORK=off go test ./internal/jobs/ledger`.

- `[+]` L5 - Re-audit MySQL dependency and pool tuning for new ops commands.
  - Area: command bootstrap and config.
  - Disposition: Closed through M10/H8. DB-using commands now share validation,
    signal handling, locks, and mode-aware pool defaults where applicable.
  - Verification: targeted command packages build.

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

## Verification For M26 Closure

- `[X]` `GOWORK=off go test ./...`
- `[X]` `(cd ../pzdesign && GOWORK=off go test ./...)`
- `[X]` `GOWORK=off staticcheck ./dsp ./match ./adminapi ./internal/cmdboot ./internal/jobs/cache ./internal/jobs/ledger ./internal/jobs/midcallback ./cmd/redis-cache ./cmd/mid-callback-retry ./cmd/ledger ./cmd/winloss`
- `[X]` `(cd ../pzdesign && GOWORK=off staticcheck -checks=all,-ST1000,-ST1003,-ST1006 ./cmd/unify ./summer ./summer/pub ./summer/slot ./summer/targetname ./summer/midroute)`
- `[X]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load && ./scripts/aofei-local.sh check-sql && ./scripts/aofei-local.sh diff-schema`
- `[X]` `./scripts/aofei-local.sh reset-sample && ./scripts/aofei-cache-smoke.sh`
- `[X]` `./scripts/aofei-doc-check.sh`
- `[X]` `git diff --check`
- `[X]` `GOWORK=off staticcheck ./dsp ./internal/jobs/midcallback ./internal/safehttp ./cmd/redis-cache ./cmd/mid-callback-retry ./cmd/maxmind ./cmd/spread ./cmd/nats-client`
- `[X]` `./scripts/aofei-doc-check.sh`
- `[X]` `git diff --check`
