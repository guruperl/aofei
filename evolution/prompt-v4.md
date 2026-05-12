# Prompt V4

Direction change for M14:

```text
Use disk-backed, NATS-distributed local static cache for publisher, slot
candidate, audience, and creative data. Keep Redis for shared mutable state such
as frequency caps, uploaded audience memberships, pacing, budgets, and counters.
The bid hot path should read static data from in-process immutable maps backed
by local spread snapshots, not from Redis or direct file IO.
```

Relevant project context at this prompt:

- Local Docker MySQL, Redis, and Core NATS remain the supported development
  runtime.
- Redis remains required for mutable bid-time features, but should no longer be
  the hot lookup path for static local/spread bid serving.
- Core NATS plus full refresh/bootstrap is the M14 reliability baseline;
  JetStream/KV is deferred unless replayable cache updates become necessary.
