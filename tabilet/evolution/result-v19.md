# Result V19

M36 hardens runtime operations and verification without changing schema/cache
payloads or direct SSP response contracts.

Implemented direction:

- `cmd/spread` now runs under a signal-aware context, logs callback results
  without unbuffered reporting channels, and drains NATS on shutdown.
- Middleman bidder fanout no longer falls back to unwrapped
  `http.DefaultClient`; every fanout validates endpoint URLs before request
  creation, and nil-client fanout uses the safe callback HTTP client.
- Frequency-cap refresh records expvar counters for refreshes, retries,
  conflicts, and last refresh latency.
- Audit publishing exposes queue depth in addition to enqueue/drop/publish
  counters.
- The middleman callback retry command reports due and stale-processing backlog
  counts before processing.
- Local/spread cache staleness is alert-only: `local_cache_max_age_seconds`
  marks scrape-time `aofei_local_cache_stale`,
  `aofei_local_cache_loaded_at_unix` records the loaded snapshot timestamp, and
  old snapshots continue serving until operators reload or restart the node.
- README, AGENTS, and the memory bank now distinguish package, runtime
  hardening, Docker smoke, admin integration, and schema verification commands.

Deferred direction:

- `cron_halfhour` trigger/table cleanup remains deferred because it is a schema
  baseline decision and should be handled with the normal schema workflow.
- Larger architecture cleanup, such as moving all spread logic under
  `internal/jobs/spread`, remains optional future work.
