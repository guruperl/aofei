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
- `slot/<size_id>/<slot_id>` compiled `match.RAdvs` candidates.
- `audience/<item_id>` compiled audience predicates.
- `creative/<creative_id>` creative metadata, trackers, landing URL, failback,
  size, and content.

Disk snapshots under `.local/spread/` are the durable local recovery format, not
the hot lookup path. The DSP loads those files into in-process maps and serves
static lookups from memory.

## Implemented M14 Behavior

`cmd/redis-cache` now treats full refreshes as replacement operations:

- Redis mode deletes `pubmap`, `audience`, `creative`, and existing `slot:*`
  keys before repopulating static cache state.
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
  maps between generations.
- Directory mtimes form the local reload generation.
- Missing audience entries remain wildcard matches.
- Bids with no caps/uploads can complete without Redis.
- Bids that require frequency caps or uploaded audience membership fail closed
  when Redis mutable state is unavailable.

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

No-cap/no-upload traffic will likely be limited by CPU, allocations, Go GC, and
NATS audit publishing. Work here includes lower-allocation OpenRTB parsing,
candidate pruning, asynchronous audit queues, and NATS publish batching/drop
metrics.

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

NATS audit/log publishing can still affect bid p99 if request handlers publish
or flush synchronously. Analytical logs should use bounded local queues,
background publishers, batch writes on dedicated log nodes, and drop counters.
Core NATS is acceptable for best-effort analytics; use JetStream only if replay
or durability becomes a product requirement.

Cache reloads can cause CPU and IO spikes during full snapshot decode and map
swap. Work here includes typed/versioned payloads, incremental generation
builds, and reload metrics.

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
