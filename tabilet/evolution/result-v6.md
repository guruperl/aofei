# Result V6

Resulting direction after M19:

- Cache refresh and ledger aggregation have reusable internal job packages.
- `cmd/redis-cache` and `cmd/ledger` remain supported standalone command
  wrappers.
- `cmd/unify` remains HTTP UI and ADX serving only; no cache or ledger scheduler
  runs inside it.
- Redis cache refresh remains a standalone singleton cron/timer job on one
  dedicated node, not a `unify` background job.
- Ledger remains a standalone singleton cron/timer job on the log aggregation
  node, not a `unify` background job.
- Missing ledger input files remain retryable command input errors.
- `nats-client`, `spread`, and `winloss` remain separate operational commands.
