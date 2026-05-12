# Result V2

Resulting direction after M9:

- [docs/production-runbook.md](../docs/production-runbook.md) is the active
  production/operator entry point.
- Production default config paths are `/etc/aofei/aofei.json` and
  `/etc/aofei/summer.json`, supplied through `AOFEI` and `SUMMER`.
- Local Docker remains the development workflow and is documented separately.
- Historical operations notes and `backup/` are explicitly historical-only and
  must not provide active credentials or setup commands.
- Summer/Genelet CORS now allows only the exact `ServerURL` origin plus exact
  `CORSOrigins` entries.
- Static file parent-path requests return immediately after rejection.
- SHA1-era password hashes are documented as compatibility behavior pending a
  future auth migration.
