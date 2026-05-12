# Milestones

This file stays at milestone level. Detailed task rows live in the matching
`memory-bank/status-M*.md` files.

Status markers:

| Symbol | Meaning |
|---|---|
| `[ ]` | Pending |
| `[+]` | Completed |
| `[~]` | In progress |
| `[!]` | Blocked |
| `[X]` | Cancelled |

## M0 - Agentic Harness Bootstrap `[+]`

Create the project operating layer for future agent work.

Scope:

- Add `AGENTS.md`, `memory-bank/`, `evolution/`, and `docs/`.
- Rewrite the root README as a clean current operator entry point.
- Move or preserve old long-form documentation under `docs/`.
- Record milestone-level direction without inventing detailed task breakdowns.

Acceptance:

- A new agent can read `AGENTS.md` and know where to start.
- Product, architecture, tech stack, milestones, and coarse status are captured.
- Historical docs are not lost, but the root README no longer presents legacy
  manual setup as the active workflow.

## M1 - Local Docker Runtime Stabilization `[ ]`

Make the Docker-backed development runtime the unquestioned local baseline.

Scope:

- MySQL, Redis, and NATS run from Docker using repository helpers.
- `etc/aofei.local.json` and `etc/summer.local.json` are generated and ignored.
- Local service auth uses Docker helper credentials only.
- Reset/load/sample/status commands are documented and repeatable.

Acceptance:

- A clean checkout can start local services, load the database baseline, load
  sample data, populate Redis, and report service status.
- No active workflow depends on `conf/` or `eightran_*` credentials.

## M2 - Schema Baseline Stewardship `[ ]`

Make `etc/step4_init.sql` the durable schema and baseline-data contract.

Scope:

- Define a repeatable schema comparison path between Docker MySQL and
  `etc/step4_init.sql`.
- Keep views, routines, triggers, and table definitions covered.
- Strip legacy definers and production auth from baseline SQL.
- Document how to update the baseline after intentional schema changes.

Acceptance:

- Docker MySQL schema can be recreated from `etc/step4_init.sql`.
- Drift between Docker MySQL and the baseline can be detected and reviewed.

## M3 - Redis And NATS Cache Pipeline Reliability `[ ]`

Prove that cache and message-bus flows work from the Docker services.

Scope:

- Validate `cmd/redis-cache` Redis mode, spread mode, and combined mode.
- Verify `PubMap`, `RAdv`, audience, and creative cache structures after sample
  data is loaded.
- Confirm NATS connectivity for paths that require it.
- Keep Redis key inspection and cache-read commands documented.

Acceptance:

- Cache population succeeds against Docker MySQL and Docker Redis.
- NATS-required cache/spread flows connect to Docker NATS.
- Cache inspection commands show expected sample objects.

## M4 - Bid Path Smoke Coverage `[ ]`

Add a reliable local proof that the DSP request path still works.

Scope:

- Use existing `etc/samples/` OpenRTB fixtures where possible.
- Exercise config loading, Redis cache reads, matching, bid response generation,
  and win/loss-adjacent paths.
- Keep tests deterministic and independent of production services.

Acceptance:

- A documented command proves the seeded local runtime can process representative
  bid-path samples.
- Failures point to config, schema, cache, fixture, or matching boundaries.

## M5 - Summer/Genelet Admin Compatibility `[ ]`

Align admin models, filters, and components with the active Docker schema.

Scope:

- Check Summer model/filter tests against the current schema.
- Confirm active tables, views, and stored routines still match admin model
  expectations.
- Keep generated local config and upload/template paths out of git.

Acceptance:

- Admin model smoke tests run against the Docker schema.
- Known admin/schema mismatches are either fixed or explicitly listed for later
  product decisions.

## M6 - Ledger, Logs, And Operational Commands `[ ]`

Clarify and verify the non-bid operational commands.

Scope:

- Review `cmd/ledger`, `cmd/nats-client`, `cmd/winloss`, `cmd/spread`, and
  related SQL/log paths.
- Define what local data each command needs.
- Separate active command workflows from historical deployment notes.

Acceptance:

- Each active operational command has a documented local invocation or a clearly
  recorded blocker.
- Log and ledger flows do not require production credentials.

## M7 - MaxMind And Geo Runtime `[ ]`

Make geodata expectations explicit and locally testable.

Scope:

- Keep `etc/maxmind.json` as the active geodata config reference.
- Document which MaxMind database files are external runtime inputs.
- Verify existing maxmind/ipsearch tests with local fixtures or clear skips.

Acceptance:

- Developers know which geo assets are required and where config points.
- Geodata tests are either runnable locally or marked with explicit input
  requirements.

## M8 - Full Repository Test Hygiene `[ ]`

Move from scoped smoke checks to a clean repository-level verification target.

Scope:

- Resolve the `backup/` Go package discovery issue.
- Decide the canonical command for full Go verification.
- Add or update CI-style verification notes.
- Keep historical files preserved without polluting active package tests.

Acceptance:

- `GOWORK=off go test ./...` is either clean or replaced by an explicitly
  justified canonical test command.
- The root README and `AGENTS.md` point to the same verification target.

## M9 - Production Deployment Runbook `[ ]`

Rebuild production/operator docs from current reality rather than legacy notes.

Scope:

- Review historical systemd/NATS/Redis/TDengine notes under
  `docs/legacy-operations.md`.
- Decide what still applies to production.
- Separate local Docker development from production deployment.
- Document secrets, backup/restore, service ownership, and rollout assumptions
  without committing credentials.

Acceptance:

- Operators have one current deployment/runbook document.
- Historical notes are either retired or clearly marked as retained context.

## M10 - Documentation And Agent Stewardship `[ ]`

Keep the harness useful as the project changes.

Scope:

- Maintain `README.md`, `AGENTS.md`, `memory-bank/`, `evolution/`, and `docs/`.
- Add new evolution versions only for real direction or boundary changes.
- Keep milestone docs summary-level and detailed task lists in `status-M*.md`.

Acceptance:

- A future agent can resume from the memory bank without rediscovering local
  runtime, schema, or documentation decisions.
- Milestones, status, and docs remain consistent after each substantial change.
