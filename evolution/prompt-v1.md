# Prompt V1

Initial direction:

```text
Set up the agentic harness for this existing package using the sample package in
../skills and the "start up harness in an existing package" approach. Build
AGENTS.md, memory-bank/, evolution/, and docs/. Keep the root README clean, move
or merge useful existing documents into docs/, and make memory-bank/milestone.md
contain a detailed milestone-level list. Detailed task breakdowns will be
discussed later.
```

Relevant existing project context at the time of this prompt:

- Docker MySQL is the active local database backend.
- Docker Redis and Docker NATS are part of the local runtime.
- `etc/step4_init.sql` is the active schema/data baseline.
- The retired root config directory has been replaced by active `etc/` files.
- Legacy named MySQL users should not be active.
