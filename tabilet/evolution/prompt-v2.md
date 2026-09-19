# Prompt V2

Direction change for M09:

```text
Make production operations current again by adding a Linux systemd-oriented
production runbook and keeping local Docker as the development workflow. Keep
code changes narrow: harden Genelet CORS defaults, stop static path traversal
after rejection, redact historical credential-like fixtures, and document
SHA1-era password hashes as compatibility behavior rather than the future auth
contract.
```

Relevant project context at this prompt:

- Local Docker MySQL, Redis, and NATS remain the supported development runtime.
- `etc/step4_init.sql` remains the local schema/data baseline.
- Historical deployment notes and `backup/` are retained only as quarantine
  context.
- Production deployment is now expected to be owned by Linux/systemd services
  and external MySQL, Redis, NATS, MaxMind, log, upload, and config lifecycles.
