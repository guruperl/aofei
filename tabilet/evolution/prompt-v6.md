# Prompt V6

Operators want reusable cache and ledger command logic, but `cmd/unify` should
remain an HTTP UI and ADX service rather than a per-node maintenance scheduler.

Refactor Redis cache refresh and ledger aggregation so the command code is
reusable. Keep Redis cache refresh as a standalone singleton cron/timer job on
one dedicated node. Keep ledger as a standalone singleton cron/timer job on the
node where `nats-client` aggregates win/loss log files. Keep `nats-client`,
`spread`, and `winloss` outside `unify`.
