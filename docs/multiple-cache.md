# Multiple Cache Architecture

This document records the current cache split, what M14 already changed, and
the likely next bottlenecks after static bid-time reads move out of Redis.

## Cache Roles

MySQL remains the source of truth. Cache commands compile MySQL rows into
runtime payloads.

Redis owns shared mutable state:

- `bothcap:<user_id>` frequency cap hashes.
- `upload:<adv_id>:<marker>` uploaded audience membership sets.
- Future pacing, budget, throttling, and distributed counters.

Static bid-serving data is local:

- `pubmap/<domain>` publisher/site/slot routing.
- Direct SSP derives publisher-by-id routing from `pubmap`, including reverse
  active site/slot and configured slot-size metadata for `/pz` validation.
  Redis mode also publishes the additive `pubmap:by-id` hash for `/pz`
  serving; local mode derives the same lookup in memory and does not add a
  spread directory.
- `slot/<size_id>/<slot_id>` compiled `match.RAdvs` candidates.
- `audience/<item_id>` compiled audience predicates.
- `creative/<creative_id>` creative metadata, trackers, landing URL, failback,
  size, and content.

Middleman bidder routes are Redis static data in M20:

- `middleman:routes:v2` stores the preferred M25 versioned JSON payload
  compiled from active `adv_bidder` and `mid_route_*` rows, including trigger
  mode and synthetic item ACL payloads. `middleman:routes` remains a legacy
  fallback-only key for M24 rolling-deploy compatibility.
- It is populated by the singleton `cmd/redis-cache -cache=redis|all|routes`
  job. Route-only mode leaves the other cache families untouched.
- M24 route payloads include additive metadata for generated time, entry count,
  source, route DB high-water timestamp, and checksum. Older version-1 payloads
  without metadata are still readable with unknown freshness.
- M25 route payloads include `trigger_mode`; older entries without it are
  treated as `Fallback`.
- It is not mirrored into spread/local snapshots yet.

Disk snapshots under `.local/spread/` are the durable local recovery format, not
the hot lookup path. The DSP loads those files into in-process maps and serves
static lookups from memory.

## Implemented M14 Behavior

`cmd/redis-cache` now treats full refreshes as replacement operations:

- Redis mode builds `pubmap`, `pubmap:by-id`, `audience`, `creative`,
  `middleman:routes:v2`, legacy `middleman:routes`, and every active `slot:*`
  family under internal `:next` keys. One `MULTI/EXEC` renames populated
  shadows over live keys and deletes empty or obsolete live families, so a
  build failure leaves the old generation intact and readers never observe the
  former delete-then-repopulate window.
- Spread mode publishes `__reset__` family subjects before publishing new
  snapshots.
- Item-level RAdv refreshes recompute affected creative sizes from MySQL slot
  state rather than merging against old Redis or spread-file state.

`cmd/spread` now acts as a local static snapshot receiver:

- It subscribes with the NATS tail wildcard so dotted publisher subjects such as
  `pubmap:example.com` are received.
- It writes snapshots by temp file, fsync, atomic rename, and directory fsync.
- It handles full-family reset subjects and slot cleanup subjects explicitly.
- On startup, it best-effort bootstraps static spread files from Redis when
  Redis and MySQL are reachable.

`dsp.Controller` local/spread mode now uses an in-process static cache:

- Publisher, slot candidate, audience, and creative reads are served from Go
  maps without request-path filesystem checks.
- Direct SSP publisher-by-id and slot-size validation reads are derived from the
  same publisher map in memory.
- The initial snapshot is loaded at controller startup when `is_local=true`.
- Later refreshes use the explicit local static-cache reload hook after spread
  files have been replaced.
- Local cache staleness is alert-only: `local_cache_max_age_seconds` sets the
  freshness threshold for the scrape-time `aofei_local_cache_stale` expvar,
  while `aofei_local_cache_loaded_at_unix` records the loaded snapshot timestamp
  and `aofei_local_cache_age_seconds` reports the current snapshot age. The bid
  path does not fail closed solely because a static snapshot is old; operators
  should alert and reload or restart the affected node.
- Missing audience entries remain wildcard matches.
- Bids with no caps/uploads can complete without Redis.
- Bids that require frequency caps or uploaded audience membership fail closed
  when Redis mutable state is unavailable.
- Redis-mode middleman route reads use an immutable controller snapshot for
  `middleman_route_cache_ttl_ms` (default 5000 ms). Concurrent misses share one
  refresh; refresh errors are cached for the same short interval and do not
  authorize fanout from an expired snapshot. The shared refresh has its own
  `middleman_timeout_ms` deadline and is detached from each waiting request.
  Canceling the initiating request therefore does not cancel the Redis load,
  fail other waiters, or prevent the loaded/error snapshot from being cached.
- Frequency-cap refresh keeps the existing `bothcap:<user_id>` hash and binary
  `BothCap` payload, but writes through a Redis optimistic transaction so
  concurrent `/imp` and `/clk` callbacks do not lose increments. Counters
  saturate at 255, and each write ensures at least the configured
  `cap_state_ttl_seconds` remains without shortening a longer existing TTL.
  The default 90-day idle retention exceeds the packed format's maximum active
  cap period and bounds abandoned user keys without changing payload layout.
  Bulk compatibility writes use one Redis script to commit hash fields and add
  expiry only for new, persistent, or shorter-lived keys, so they preserve a
  longer TTL without exposing written data without its required expiry.
- Impression and click replay keys are separate Redis mutable-state records.
  Each starts as a short owned processing claim and becomes a
  exact signature-deadline completion marker only after publication succeeds.
  Pre-publication failures release the claim, while a transactional per-event
  cap marker prevents retries from repeating cap mutation. Duplicate signed
  events skip cap refresh and ledger publication. Claim and cap failures are
  fail-open; keyed claim failures still use the idempotent cap marker, while
  unkeyed events publish without cap mutation.

## Hardware Affinity Option

A hardware load balancer can reduce mutable Redis pressure when it keeps a
visitor's traffic on the same physical DSP node. The DSP should still key caps by
the full anonymous user/profile identity, not by IP alone. Source-IP affinity is
only used to choose the node.

That changes cap semantics from exact global to mostly node-local,
best-effort-global:

- Same user, stable IP, same node: local cap state is accurate.
- Same user, changed IP or rebalance: cap state may split.
- Shared NAT users do not collide if the cap key remains the user/profile id;
  they only co-locate on the same node.

If this mode is implemented later, make it explicit in config and docs, for
example `cap_mode=local_affinity_best_effort`, and bound local cap maps by TTL
based on the longest active cap period.

## Likely Future Bottlenecks

After static data is local, the first bottleneck depends on traffic mix.

No-cap/no-upload traffic will likely be limited by CPU, allocations, and Go GC.
Work here includes lower-allocation OpenRTB parsing, candidate pruning, and
audit queue observability.

Cap-heavy traffic with Redis caps will likely be limited by Redis mutable-state
latency and throughput. Work here includes local affinity cap mode, local
short-TTL cap caches, Redis sharding by user/profile id, batching, and clear
failover semantics.

Cap-heavy traffic with local affinity caps shifts pressure to node memory, TTL
cleanup, lock contention, restart recovery, and traffic imbalance. Work here
includes sharded in-memory maps, periodic checkpointing, bounded queues for
replication, and load-balancer hash-ring stability.

Upload-heavy traffic will likely be limited by Redis `SISMEMBER` calls and set
memory. Work here includes immutable local membership indexes, sharded hash
sets, Bloom filters with Redis confirmation, or compressed bitmap files loaded
into memory.

NATS request/response/attribute audit publishing now uses a bounded local queue
and background publisher with drop counters. Core NATS is acceptable for
best-effort analytics; use JetStream only if replay or durability becomes a
product requirement.

Cache reloads can cause CPU and IO spikes during full snapshot decode and map
swap. Current reload/freshness metrics expose reload duration, reload errors,
entry count, snapshot age, and stale status. Future work includes
typed/versioned payloads and incremental generation builds.

MySQL and cache compiler work should stay off the bid path. Bottlenecks there
affect admin/cache freshness rather than live bid latency.

## Measurement Checklist

Measure before optimizing further:

- Redis commands per request by family.
- Redis operation latency histograms in the DSP process.
- Bid p50/p95/p99 split by no-cap, cap-heavy, and upload-heavy traffic.
- NATS publish queue depth, publish latency, and dropped-log counters.
- Go allocation rate, heap size, GC pause time, and candidate counts.
- Per-node request distribution under load-balancer affinity.
- Static cache reload duration and generation swap frequency.
