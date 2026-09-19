# Result V4

Resulting direction after M14:

- Static cache families are publisher maps, slot candidate `RAdvs`, audience
  predicates, and creative metadata/content.
- Local/spread bid serving reads those static families through an in-process
  generation cache backed by `.local/spread/` snapshots.
- Redis remains the mutable-state backend for frequency caps, uploaded audience
  memberships, and future pacing, budget, throttling, or distributed counters.
- `cmd/redis-cache` full refreshes remove stale static Redis keys and publish
  spread reset subjects before repopulating snapshots.
- `cmd/spread` persists snapshots atomically, receives dotted publisher subjects,
  and best-effort bootstraps local static files from Redis on startup.
- Core NATS remains the default cache distribution path; replayable JetStream/KV
  cache distribution is a future option, not the M14 implementation baseline.
