# Prompt V3

Direction change for M12/M13:

```text
Run a deep OpenRTB, audience matching, and DSP workflow review without runtime
refactors in M12. Capture concrete findings in documentation and status memory.
Create M13 as the explicit implementation backlog for the code/design work found
during the review.
```

Relevant project context at this prompt:

- Local Docker MySQL, Redis, and NATS remain the supported development runtime.
- M12 does not change OpenRTB wire behavior, Redis/spread payloads, schema, or
  production config shape.
- M13 is the next backlog for ACL, geo/date-hour, creative-format,
  multi-impression, selection, cache, and measurement fixes found during M12.
