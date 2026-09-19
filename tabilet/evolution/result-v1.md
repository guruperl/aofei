# Result V1

Resulting harness direction:

- `AGENTS.md` is the bootstrap file for future agents.
- `memory-bank/` is the active project source of truth.
- `evolution/` records material direction changes.
- `docs/` holds stable long-form references and historical notes.
- Root `README.md` is short and focused on current local operation.

Key decisions captured:

- Use Docker MySQL, Redis, and NATS for local development.
- Use `etc/step4_init.sql` as the schema baseline.
- Use generated `etc/aofei.local.json` and `etc/summer.local.json` for local
  runtime config.
- Preserve historical material in `backup/` and `docs/`, but keep it out of
  active runtime paths.
- Keep milestones at milestone level until detailed task planning is requested.
