# Single-Region Availability, Recovery, And SLO

This is the O02 operating contract for one W8M region. It defines the topology,
failure behavior, recovery order, and evidence required before anyone may
describe the service as 99.9% available. It does not authorize multi-region
state or traffic; that remains deferred.

## Topology And Ownership

Run at least two identical `cmd/unify` nodes in separate failure domains behind
one health-checking regional load balancer. Nodes use the same reviewed binary,
schema/cache contract, clock source, public `server_url`, tracking key
generation, and secret/config version. The load balancer checks `GET /readyz`
without caching and sends public traffic only to ready nodes. `GET /healthz` is
a process-liveness check for the service manager; neither endpoint returns
dependency, account, or configuration detail and neither is the protected
`/debug/vars` metrics surface.

`/readyz` is lifecycle/process-local by design. It is 204 only after controller
and UI initialization, becomes 503 before graceful HTTP drain, and is 503 when
local/spread mode has no generation or its generation exceeds the tightest
configured cache age. It does not withdraw every HTTP node for one shared
MySQL/Redis/NATS outage; those dependencies have explicit per-feature failure
semantics and protected health metrics. `/healthz` remains 204 during drain
until the process exits so a service manager does not kill a correctly draining
node.

Per-process O01 QPS and concurrency limits are not a regional quota. Size the
region so `N-1` nodes can carry the contracted peak at no more than 70% of their
measured safe ceiling, including expected traffic unevenness. Adding node
limits together is not capacity evidence. The load balancer must use connection
draining longer than the application's 15-second shutdown window and must not
retry non-idempotent account/action writes without their documented
idempotency identity.

Singleton roles run on designated operations nodes, never every HTTP node:

| Role | Ownership and failover |
|---|---|
| Redis cache publication | One `cmd/redis-cache` timer; all mutation modes share `aofei:redis-cache`. |
| Interval/daily ledger | One `cmd/ledger` timer on the node with complete closed log files. |
| Middleman callback retry | One `cmd/mid-callback-retry` timer. |
| Action reconcile/prune | One operations timer; reconciliation also uses DB row locks and is idempotent. |
| Experiment retention | One authorized operations/privacy timer; transitions and exact deletion remain audited. |
| Traffic-quality maintenance | One restricted quality-operations owner; serving reads detached bounded snapshots. |
| Hosted-payment event retention | One restricted maintenance owner; the command cannot move money. |
| Manual accounting | Authorized named principal; A01 request keys and maker/checker rules, not an unattended failover timer. |

Redis singleton locks renew at one-third of their configured lease and commands
report an error if renewal is lost. Token-checked release cannot delete a
successor's lease. `ledger_log.timely` and `daily_log.daily` are unique durable
identities, so a Redis partition or failover cannot create two interval/day
source rows. Retry/action workers retain their database idempotency and row
claim rules. Lease safety reduces overlap; durable idempotency remains required
because no distributed lock can prove ownership through every partition.

## Dependency Failure Matrix

| Failure | Serving behavior | Recovery and evidence |
|---|---|---|
| One HTTP node | Readiness is withdrawn before drain; load balancer uses another ready node. In-flight requests receive up to 15 seconds to finish. | Alert on target loss and regional remaining headroom; replace node from reviewed artifact/config. |
| MySQL | Cache-backed auctions continue; management reads/writes and `/action` durable inserts fail. Impression/click publication remains primary and action-touch writes fail open. Cache/ledger/accounting jobs stop without reporting success. | Restore connectivity, inspect DB pool/dependency metrics, reconcile actions and delayed jobs in source order. Do not clear Redis budget floors. |
| Redis | Redis-cache-mode auction reads fail/no-bid; limited local demand fails closed at D01 reservation, tracking replay/caps fail open, and middleman callback state is unavailable. Local/spread static reads may serve eligible unlimited local demand but features requiring Redis remain unavailable. | Restore Redis, republish full static cache from MySQL, retain/reconcile conservative delivery floors, inspect callback and measurement gaps. |
| NATS | Auctions continue and returned bids are not rolled back; audit publish errors/drops grow and new ledger input may be incomplete. | Restore NATS/consumer, preserve the known gap, delay statements, reconcile against surviving tracker/source evidence. Never invent zero rows for missing log intervals. |
| Stale/missing static cache | D01 candidates with stale policy fail closed. Local/spread node readiness is 503 when the process generation is missing/stale. | Stop rollout, publish a compatible full generation, verify age/version/media/floor samples, then re-add nodes. |
| Log disk full/unwritable | HTTP bidding continues; NATS consumer cannot durably build complete ledger input, so ledger/accounting freshness is violated. | Stop statement creation, restore capacity/permissions, verify closed files and replay source; record data loss if source no longer exists. |
| Bidder DNS/proxy/endpoint | Bounded middleman call fails with fixed outcome; valid local winner is preserved, or fallback remains no-bid. | Disable route/credential, inspect safe fixed diagnostics, retry only callback rows classified retryable. |
| Regional proxy/load balancer | Origin nodes may remain healthy but public availability is lost. | Use provider health, independent synthetic probes, reviewed DNS/TLS rollback, and incident timing; origin liveness alone is not availability evidence. |

## Backup And Restore Contract

MySQL is authoritative for schema, accounts, campaign/publisher configuration,
ledger/daily and R01/R02 facts, S02 identity/audit state, S03 quality evidence,
I03 API state, and A01/A02 financial records. Redis static caches and
local/spread snapshots are derived and must be rebuilt. Redis mutable
delivery/cap/callback state needs a separately reviewed encrypted persistence
and recovery policy; a MySQL restore does not pretend to recreate lost in-flight
reservations or callbacks. NATS/log files are accounting inputs and require
encrypted storage/retention consistent with the ledger recovery point.

Production requirements:

- encrypted MySQL full backup plus binlog/incremental stream to a separate
  access-controlled failure domain;
- checksum, size, schema/accounting contract version, backup time, source
  version, key generation, and retention metadata;
- daily backup success verification, at least quarterly clean-room restore, and
  a named key/restore owner;
- 35 daily and 13 monthly generations unless legal/accounting/privacy policy
  specifies a reviewed stricter schedule; expired generations are destroyed and
  deletion cases are reapplied before exposure;
- target database recovery point no greater than 15 minutes and clean-room
  service recovery time no greater than 60 minutes. These are objectives, not
  claims, until the production backup provider and restore drill record them.

Restore order is fixed:

1. Declare incident scope, stop singleton timers and configuration writes, and
   select a checksum-verified encrypted generation plus binlog point.
2. Restore into an isolated MySQL service. Verify all schema objects,
   `acct_contract=usd-cpm-impression-v3`, statement/adjustment/audit counts and
   immutable triggers, correction links, ledger/daily uniqueness, R01/R02
   facts, S02/I03 security state, S03 quality evidence, A02 hosted-payment
   mappings/evidence, and required account/inventory rows. Reject mixed
   contract versions.
3. Reapply approved deletion cases and verify backup/source timestamps against
   the RPO. Preserve discrepancies; do not “repair” by deleting immutable
   accounting evidence.
4. Start fresh Redis/NATS dependencies. Rebuild the full compatible Redis cache
   from restored MySQL, validate publisher tokens, D01 policy age, D02 creative
   media, middleman route health, and conservative mutable-state decisions.
5. Start one HTTP node outside the load balancer, check `/healthz`, `/readyz`,
   protected dependencies, representative Banner/Video/Native no-bid/fill, and
   tracking/action behavior. Add nodes gradually, then restart singleton jobs
   in cache, retry/reconcile, interval ledger, daily ledger, and accounting
   order.
6. Record actual RPO/RTO, lost/deferred intervals, cache generation, binary and
   config/key versions, canary evidence, and the follow-up owner.

`scripts/aofei-recovery-drill.sh` is a safe local rehearsal. It creates unique
disposable MySQL/Redis containers, loads the baseline and synthetic fixture,
adds accounting/action evidence, takes a checksummed logical dump with routines
and triggers, restores it into a clean MySQL instance, proves accounting
immutability, ledger uniqueness, current 95-table/6-routine/57-trigger inventory,
reporting/experiment/security/quality/API/hosted-payment evidence, a restored
D03 route and credential-safe fallback preflight, and rebuilt Redis. Its
ephemeral unencrypted dump never leaves an owner-only temporary directory and
is destroyed on exit; it is not a production backup mechanism or production
RTO result.

## Deployment, Rotation, And Rollback

Use the cache-first D02/P01 order: audit populated rows and freeze edits; apply
reviewed schema migration; install compiler; publish/inspect additive cache;
canary one new HTTP node; expand; then unfreeze. Readiness and independent
synthetic requests gate each step. Rollback removes new nodes, restores the
complete prior binary/config set, and retains compatible new cache data until
no new reader remains. Never replay `step4_init.sql` over populated data.

Rotate one dependency credential/key class at a time. Database, Redis, NATS,
bidder/provider headers, and TLS keys require overlapping credential support or
a documented maintenance window plus canary. `tracking_secret` currently has
no dual-key window: rotation invalidates outstanding tracker/action/middleman
signatures and changes new privacy pseudonyms, so use an explicitly approved
window and monitor rejection/cap discontinuity. A rollback must not restore a
secret already revoked for compromise.

## Service Objectives And Error Budget

The proposed public-auction SLO is **99.9% over a rolling 30-day production
window**, measured by independent regional probes plus server counters at the
contracted request mix and offered load. This document does not claim W8M has
already achieved it; a claim requires a named measurement window and retained
evidence.

A good auction event is an eligible `POST /bid/{domain}` or `POST /pz` request
that completes within the partner timeout with a protocol-valid 2xx response;
204/no-fill is good. Malformed/unauthorized client 4xx is excluded. 429,
server-generated 5xx, timeout, invalid response, or network/TLS/proxy failure is
bad when offered load is within the contracted O01 profile. Planned maintenance
is not silently excluded; any approved exclusion must be named in the report.
At 99.9%, the 30-day bad-event budget is 0.1% (43 minutes 49 seconds only when a
minute-based approximation is also reported).

Supporting objectives:

| Indicator | Objective |
|---|---|
| Auction latency | p95 <= 100 ms and p99 <= 250 ms within contracted profile; partner `tmax`/configured timeout remains authoritative when tighter. |
| Ready regional capacity | At least two nodes and `N-1` measured headroom; zero stale local generations admitted. |
| Full cache freshness | Publication every <=300 s with serving age <=900 s by default; failed generation leaves the prior compatible generation. |
| Callback retry | 99% of retryable callbacks succeed or reach explicit terminal state within 15 minutes; no silent queue loss. |
| Action reconciliation | Touch-error backlog reconciled within 15 minutes after MySQL recovery. |
| Interval/daily ledger | Closed 10-minute interval complete within 20 minutes; UTC daily complete by 00:30. Missing input is a freshness failure, not zero success. |
| Database recovery | RPO <=15 minutes, RTO <=60 minutes after declared restore start, measured by production-style clean-room drills. |

Burn alerts use fixed O01 dimensions: page at >=14.4x budget burn over 1 hour or
>=6x over 6 hours; ticket at >=1x over 3 days. Freeze rollout/route expansion
when either page condition, stale cache, incomplete ledger input, callback
backlog, mixed contract version, or failed restore evidence is active. Reports
must state node count, hardware, binary/config/cache versions, traffic mix,
dependency state, exclusions, numerator/denominator, percentiles, error-budget
consumption, and incident links. Microbenchmarks and one clean-room drill are
regression/recovery evidence, not a production 99.9% claim.

## Required Exercises

At least quarterly and before major traffic expansion, record:

- remove one canary HTTP node during sustained within-profile traffic and prove
  readiness withdrawal, drain, retry policy, remaining headroom, and no error
  budget breach;
- pause each shared dependency in a non-production environment and compare
  observed HTTP/metric/ledger behavior to the matrix above;
- age/withhold one cache generation, fill the log disk quota, create a callback
  backlog, and delay one ledger interval without fabricating success;
- run the clean-room restore drill, then a production-provider restore with
  encrypted media and measured RPO/RTO;
- exercise cache-first canary/rollback and one non-compromise credential
  rotation; separately tabletop tracking-key compromise and revocation.

Each record names date, owner, environment, artifact/config versions, starting
state, injected fault, expected/actual behavior, timestamps, RPO/RTO or SLO
impact, discrepancies, corrective action, and retest evidence.
