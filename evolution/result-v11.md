# Result V11

Resulting direction after M24:

- `cmd/redis-cache -cache=routes` is the route-only publisher and
  `-cache=routes -read` is the route-only inspection path.
- `middleman:routes` remains version-1 JSON, with optional metadata for
  generation time, entry count, source, route-table high-water timestamp, and
  checksum. Older version-1 payloads remain readable with unknown freshness.
- Summer/Genelet `midroute` shows route-cache freshness and route health, but
  route refresh remains owned by the singleton cache node.
- `mid_callback_retry` stores retryable downstream `/mid/win`, `/mid/loss`, and
  `/mid/bill` forwarding failures. Missing URLs, invalid URLs, duplicate
  callbacks, and HTTP 4xx other than 429 are not queued.
- `cmd/mid-callback-retry` forwards downstream retry rows only; it does not
  republish ledger events.
- `trigger_mode='Always'`, spread/local route snapshots, and real settlement
  execution remain future work.
