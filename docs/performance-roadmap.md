# Performance Roadmap

This document is an advisory roadmap for scaling `aofei` after M26. It is not
a committed milestone list. Pick work from it only after the measurement
baseline shows which path is actually limiting production traffic.

Current runtime shape:

- MySQL is the source of truth for admin and campaign state.
- Redis stores mutable bid-path state and Redis-mode cache payloads.
- NATS carries best-effort log and spread messages.
- Spread disk snapshots and in-process maps serve static publisher, slot,
  audience, creative, and route data when local/spread mode is used.
- `cmd/unify` serves the HTTP bid, tracker, middleman callback, admin, and
  `/debug/vars` endpoints from the sibling `../pzdesign` checkout.

The high-level order is:

1. Build a repeatable measurement baseline.
2. Remove avoidable Redis round trips on the cap path.
3. Consider a bounded local cap cache only if cap traffic justifies it.
4. Reduce allocation and JSON overhead where benchmarks prove value.
5. Split cache delta work into its own milestone before implementation.

## Already Addressed By M26

The earlier performance draft predated M26. These items are now done or partly
covered and should not be re-opened as roadmap work without a new finding.

| Area | Current state |
|---|---|
| Protected metrics endpoint | `../pzdesign/cmd/unify` registers stdlib expvar at `/debug/vars`; O01 restricts direct peers, requires an edge deny, and adds fixed traffic/dependency/latency evidence. |
| Bid and audit counters | Bid outcomes, ECPM errors, audit enqueue/drop/publish-error counters, middleman callback failures, and local cache reload status are exported. |
| Singleton operations | Mutating `cmd/redis-cache`, `cmd/ledger`, `cmd/mid-callback-retry`, and `cmd/winloss` runs acquire Redis singleton locks. Read-only and dry-run modes skip locks where appropriate. |
| Cache payload compatibility | RAdvs, audience, and creative payloads use typed version envelopes while preserving legacy decode support. The middleman route cache remains versioned JSON. |
| HTTP service hardening | `cmd/unify` uses an explicit `http.Server` with read and write timeouts plus a bounded header size. Further timeout tuning can still be measured. |
| Middleman callback forwarding | Downstream callback URLs are expanded, validated, sent with context timeouts, guarded by safe callback HTTP behavior by default, and retried through `mid_callback_retry` for retryable failures. |
| Hot-path pooling decision | M26 added bid-response marshal benchmarking and deferred pooling until measurements justify the extra complexity. |
| Controller test seams | Redis, DB, NATS, IP search, HTTP client, callback guard, callback store, and logger dependencies can be injected for focused tests. |

## M44 Microbenchmark Baseline

M44 added parallel benchmarks before changing weighted selection or request
path allocation behavior:

```bash
GOWORK=off go test ./dsp ./match -run '^$' \
  -bench 'Benchmark(ServeBidLocalTwoImpressions|SelectOneParallel)$' \
  -benchmem -count=3
```

On 2026-07-31 with Go 1.26.1 on linux/amd64 and the repository test fixture:

| Benchmark | Observed range |
|---|---|
| `BenchmarkSelectOneParallel-8` | 150.8-156.8 ns/op, 0 B/op, 0 allocs/op |
| `BenchmarkServeBidLocalTwoImpressions-8` | 145.7-212.7 us/op, 121.4-123.9 KB/op, 1,062-1,066 allocs/op |

The bid benchmark covers a successful two-impression local-static-cache HTTP
request, including request construction, OpenRTB decode, matching, creative
materialization, tracking URLs, response encoding, and response recording.
These numbers are a same-machine before/after baseline, not a production SLO.
The parallel selection result does not justify replacing the default top-level
`math/rand` source; any future RNG change still requires a measured improvement
under the same benchmark.

## Near-Term Measurable Work

These items fit the current stack and should be evaluated in this order. Each
item needs before/after numbers under the same traffic shape and hardware.

### 1. Measurement Baseline (O01 Baseline Established)

Before changing hot paths, capture the baseline for at least these dimensions:

- Bid p50/p95/p99 split by no-cap, cap-heavy, upload-heavy, and middleman-heavy
  traffic.
- Redis command rate and latency by operation family.
- Cap mutation latency and retry/fail-closed rate.
- Audit queue depth, drop counters, and publish-error counters.
- Local cache reload duration and entry count.
- Go allocation rate, heap size, and GC pause time under representative load.
- Per-node request distribution when a load balancer is in front of multiple
  `cmd/unify` nodes.

O01 added `scripts/aofei-capacity-baseline.sh`, ADX/SSP/admission benchmarks,
and fixed runtime percentile buckets. The local-static baseline and its strict
non-production limitations are recorded in
[production-traffic-observability.md](production-traffic-observability.md).
Redis cap, upload, middleman, compression, rejection, and saturation profiles
still require staging measurements before a capacity claim.

### 2. Cap-Path Lua

Current frequency-cap refresh uses Redis hash state under `bothcap:<user_id>`.
If cap-heavy measurements show Redis round trips or optimistic transaction
retries driving tail latency, replace the client-side `WATCH` flow with a Redis
Lua script or Redis 7 `FCALL` path that performs read, recompute, and write
atomically in one call.

Keep the existing binary `BothCap` payload format so Redis state does not need a
dump/reload migration. Stage the script by config flag, load/cache the SHA, and
fall back to `EVAL` on `NOSCRIPT`.

Expected value: lower cap-path RTTs and fewer retry-driven tail spikes without
changing cap semantics.

### 3. Optional Local Cap Cache

If cap-path Lua is still not enough, add a short-TTL per-node cap cache behind
an explicit config mode, for example `cap_mode=local_ttl_best_effort`.

The safer version keeps Redis authoritative:

- Serve very recent local reads from an in-process sharded map.
- Update the local entry on cap mutation.
- Flush mutations to Redis through the Lua path.
- Reconcile from Redis on a short interval so writes from other nodes are
  observed.

This is a deliberate consistency trade-off. It can reduce Redis pressure
substantially, but it must be paired with documented TTL bounds, memory limits,
restart behavior, and over-delivery expectations.

### 4. Allocation And JSON Work

After cap pressure is understood, benchmark no-cap and upload-light traffic for
CPU, allocations, and marshal cost. Candidate work:

- Preallocate bid response maps and slices from known request sizes.
- Pool response buffers only if benchmarks show meaningful GC pressure.
- Evaluate generated OpenRTB JSON encoders or a faster JSON library in a
  contained package.
- Keep the existing correctness tests as the gate, because OpenRTB encode/decode
  behavior is an external contract.

Do not introduce pooling or alternate JSON libraries on theory alone. M26
already established that this needs measurement first.

### 5. Cache Delta Pipeline

Full static cache rebuilds are acceptable as a safety baseline but are not a
real-time campaign-management mechanism. A cache-delta pipeline could reduce
admin-to-bid-path freshness from minutes to seconds.

This should become its own milestone before implementation. The milestone needs
to decide:

- Whether deltas come from explicit Summer/Genelet dirty markers, a small
  change-log table, or MySQL binlog CDC.
- Which cache keys can be rebuilt independently.
- How spread/NATS subjects represent per-key updates.
- How bid nodes reload only changed in-process static entries.
- How full rebuilds remain a recovery backstop.

Expected value: faster campaign pause, creative swap, targeting, route, and
budget propagation. Main risk: partial invalidation bugs, so the design needs
tests and recovery semantics before code changes.

## Gated By Production Telemetry

These items are plausible but should wait for production metrics that show the
specific ceiling.

### Local Cap Authority And Affinity

A stronger local cap mode can combine source-IP or user-key load-balancer
affinity with per-node cap authority. This can drive Redis cap traffic close to
zero for stable traffic, but it changes failure semantics:

- Node loss can temporarily reset local cap state for users routed to a new
  node.
- Rebalance can split a user's cap state across nodes.
- Unstable client IPs can bypass affinity.
- Brief over-delivery is possible during loss, restart, or rebalance windows.

Only consider this if cap traffic remains a proven bottleneck after Lua and the
short-TTL cache. Document it as best-effort global capping, not exact global
capping.

### Upload Audience Local Indexes

Upload-heavy traffic may be limited by Redis membership checks. If metrics show
that `SISMEMBER` dominates, evaluate immutable local indexes or Bloom filters
generated by the cache job. Bloom negatives can skip Redis; positives should
still confirm against Redis to avoid false-positive targeting.

### Ledger Sharding And Pre-Aggregation

Ledger writes are already transactional and batched at the job level. Shard
`cmd/ledger` or add edge pre-aggregation only if interval files become too large
for the singleton job's SLA. Keep singleton-per-shard ownership so two workers
do not write the same interval rows.

### Additional HTTP Timeout Tuning

`cmd/unify` already uses an explicit server with read/write timeouts and header
limits. Further hardening such as `ReadHeaderTimeout`, `IdleTimeout`, or
request concurrency semaphores should follow observed slow-client or overload
symptoms.

## Technology Swaps And Decision Triggers

These are higher-cost moves. Treat them as decision records when their triggers
appear, not as default roadmap work.

| Technology | Consider when | Trigger to proceed |
|---|---|---|
| Dragonfly or KeyDB for Redis-compatible state | One Redis instance becomes the cap/upload bottleneck after cap Lua, local cache, and straightforward sharding have been evaluated. | Sustained Redis CPU or command latency saturation under measured production load. |
| ClickHouse or Doris for reporting | MySQL reporting and ledger queries miss operator SLA at production data volume. | Query latency and storage growth show OLAP work is the bottleneck, not missing indexes or job scheduling. |
| NATS JetStream | Win/loss, ledger feed, or CDC streams require replay or durable delivery. | Product or billing requirements cannot tolerate core NATS best-effort behavior for a subject family. Keep best-effort audit on core NATS unless requirements change. |
| MySQL binlog CDC or Debezium | Cache-delta freshness becomes a committed milestone and explicit dirty markers are insufficient. | Sub-second propagation is required and MySQL row-based binlog operations are acceptable production dependencies. |
| Aerospike or Scylla for cap state | Redis-compatible options and local cap modes are still insufficient. | Cap state needs multi-million ops/sec, cross-region replication, or tail latency beyond what Redis-compatible systems can provide. |
| gRPC or Connect for internal RPC | Internal service-to-service hops become material in latency or CPU profiles. | JSON/HTTP overhead on internal middleman, cache-delta, or ledger paths is measured as significant. Do not change exchange-facing OpenRTB HTTP/JSON. |
| Multi-region edge bidding | Exchange timeout budgets are dominated by network RTT from a single region. | Traffic opportunity is lost because p95/p99 wire latency consumes the bid budget. |

## Rollout Rules

For any item selected from this roadmap:

1. Record the baseline measurement and acceptance target first.
2. Implement behind a config flag when behavior or dependencies change.
3. Run the relevant local smoke and package tests.
4. Canary on one `cmd/unify` node before wider rollout.
5. Update `memory-bank/architecture.md`, `memory-bank/tech-stack.md`, and the
   matching `memory-bank/status-<lane><number>.md` file only when the work
   changes runtime behavior, operator workflow, cache contracts, schema, or
   milestone state.

Documentation-only roadmap changes do not create a new milestone by themselves.
