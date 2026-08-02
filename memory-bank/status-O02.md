# Status O02 - Single-Region Availability, Recovery, And SLO

State: `[+]` Complete

## Goal

Prove a recoverable, measurable single-region service before accepting
multi-region consistency and operations complexity.

## Dependencies

- O01 observability, traffic controls, and capacity baseline.
- A01 accounting recovery and reconciliation requirements.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Node topology | `[+]` | `cmd/unify` exposes separate lifecycle liveness/readiness, withdraws readiness before graceful drain, rejects missing/stale/future local generations, and has a two-node health-checked failover test. The operating contract requires at least two nodes with measured N-1 headroom. Cache, ledger, callback retry, and simulator leases renew while active; token-checked release cannot remove a successor. |
| Dependency failures | `[+]` | The failure matrix fixes MySQL, Redis, NATS, static-cache, disk/log, bidder DNS/proxy, HTTP-node, and regional-edge semantics. Focused tests exercise dependency metrics/failure, Redis fail-open/fail-closed paths, stale routes/cache, log write/queue/drain failures, unsafe DNS, missing ledger input, callback backlog, and node failover. |
| Backup and restore | `[+]` | `docs/single-region-availability.md` defines encrypted off-Git backup inventory, 35-daily/13-monthly retention, checksum/version evidence, ordered restore, RPO <=15m and RTO <=60m objectives, and quarterly drills. The disposable clean-room script restores all 63 tables, 6 routines, 21 triggers, A01/R01 facts, immutable triggers, interval/day uniqueness, and rebuilt Redis cache. |
| Deployment safety | `[+]` | Cache-first canary/rollback, readiness/drain, 15-second shutdown, schema/cache compatibility, secret/config rotation, and compromise handling are documented. `ledger_log.timely` and `daily_log.daily` are unique durable split-brain backstops. |
| SLO | `[+]` | The contract defines an unclaimed 99.9% rolling-30-day auction objective, exact good/bad/excluded events, 0.1% budget, p95/p99, freshness/callback/ledger/recovery objectives, burn alerts, and mandatory evidence fields. No production claim is made because no named production measurement window was authorized or observed. |
| Incident exercises | `[+]` | Automated failure tests and the 32-second isolated restore/cache-rebuild rehearsal passed. The contract keeps edge-inclusive node/dependency, disk, backlog, ledger-delay, canary/rollback, rotation, and provider-backed restore exercises on the quarterly/pre-expansion operator schedule. |

## Acceptance Criteria

- Loss of one HTTP node does not interrupt service beyond the documented SLO.
- Singleton jobs cannot double-write during failover.
- A clean environment can restore authoritative data and rebuild derived caches
  within recorded recovery objectives.
- A 99.9% claim is made only from measured SLO evidence over an agreed window.

## Verification

- Go 1.23.5 full tests and vet passed in Aofei and pzdesign; pinned staticcheck
  passed in both repositories; the documented Aofei and `cmd/unify` race suites
  passed.
- Two-node health-checked failover, graceful readiness withdrawal, local-cache
  readiness, renewable lease/loss/successor safety, dependency failure, disk,
  backlog, delayed-ledger, and route-expiry tests passed.
- `scripts/aofei-recovery-drill.sh` passed again in 31 seconds with inventory
  `63:6:21:1:1:1:1:1:1:1:1:1:1:1:1:usd-cpm-impression-v2`, 7 rebuilt Redis
  keys, immutable A01 audit enforcement, duplicate interval/day rejection, and
  a restored D03 fallback preflight. It used unique disposable containers; the
  configured live stack was untouched.
- Documentation/public-data/template/public-copy guards, actionlint, SQL
  baseline guard, benchmarks, and both repository diff checks passed.
- A production 99.9% statement and production-provider RPO/RTO evidence remain
  deliberately unclaimed; the operating contract requires a named production
  window and retained evidence before either assertion.

## Closeout Review

- Review added future-dated cache rejection, direct load-balancer failover
  coverage, signal-aware simulator requests, a disk-path failure test, restored
  ledger uniqueness checks, and expanded scoped race coverage.
- O02 requirements were reconciled into D03, R02, P02, I03, S03, A02, and the
  conditional I02 plan. The existing single-region direction was implemented;
  no product/ownership boundary changed, so no evolution entry was required.
- No commits or production/deployed-system mutations were authorized by this
  goal; changes remain in the working trees for final goal-level handoff.

## Exclusions

- Multi-region deployment remains deferred in [docs/defer.md](../docs/defer.md).

## Reconciliation From O01

- O01 admission limits are per process, not a distributed global quota. O02
  capacity and load-balancer calculations must account for node count,
  unevenness, draining, and failover rather than multiplying limits blindly.
- Health/readiness endpoints must remain separate from the protected expvar
  surface. Canary, rollback, and SLO evidence should reuse O01's fixed
  rejection, latency, dependency, and capacity contracts.

## Reconciliation From A01

- Backup/restore drills must preserve the `acct_contract` singleton,
  statements, immutable adjustments/audits, correction links, external opaque
  evidence references, and the matching v2 ledger/daily facts. A restore must
  reject mixed accounting versions before auction admission resumes.
- Singleton failover must include ledger, daily aggregation, and manual
  accounting ownership. Recovery evidence must reconcile advertiser charge,
  publisher pay, middleman charge/pay/margin, D01 conservative Redis floors,
  and source snapshots without deleting immutable discrepancies.

## Reconciliation From D02

- Canary and rollback drills must use D02's edit freeze, reviewed populated-data
  audit, new compiler/full-cache publication, HTTP rollout, and additive-cache
  rollback order. A D02 reader must never receive an old creative generation
  without media metadata.
- Readiness evidence includes representative Banner/Video/Native validation,
  legacy-price rejection, creative-rejection/no-bid signals, and reservation
  release after response failure or middleman displacement.
