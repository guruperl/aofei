# Status M2 - Schema Baseline Stewardship

Milestone status: `[+]` Completed

Goal: Make `etc/step4_init.sql` the durable schema and baseline-data contract.

## Tasks

- `[+]` Document the exact baseline object inventory.
  - Files: `docs/database-baseline.md`, `memory-bank/status-M2.md`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset
    ./scripts/aofei-local.sh load
    ./scripts/aofei-local.sh status
    ```
  - Result after `reset && load`: 50 base tables, 1 view, 6 routines,
    18 triggers, 0 events, 1 advertiser, 14 publishers.

- `[+]` Add a schema dump command to the local workflow.
  - Files: `scripts/aofei-local.sh`, `docs/database-baseline.md`.
  - Command to support:
    ```bash
    ./scripts/aofei-local.sh dump-schema
    ```
  - Result: command writes normalized current schema to ignored
    `.local/schema/aofei.schema.sql` without modifying `etc/step4_init.sql`.

- `[+]` Add a schema comparison command to the local workflow.
  - Files: `scripts/aofei-local.sh`, `docs/database-baseline.md`.
  - Command to support:
    ```bash
    ./scripts/aofei-local.sh diff-schema
    ```
  - Result: command compares Docker MySQL schema against a temporary database
    rebuilt from `etc/step4_init.sql`; the temp database is dropped on exit.

- `[+]` Verify table definitions are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    ./scripts/aofei-local.sh diff-schema
    ```
  - Result: `diff-schema` compares cleanly.

- `[+]` Verify views are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    rg -n '^CREATE .*VIEW|^CREATE VIEW' etc/step4_init.sql
    ```
  - Result: `diff-schema` covers `view_payment` and compares cleanly after
    normalized definer removal.

- `[+]` Verify routines are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    rg -n '^CREATE .*PROCEDURE|^CREATE .*FUNCTION' etc/step4_init.sql
    ```
  - Result: `diff-schema` covers 6 procedures and compares cleanly after
    normalized definer removal.

- `[+]` Verify triggers are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    rg -n '^CREATE .*TRIGGER' etc/step4_init.sql
    ```
  - Result: `diff-schema` covers 18 triggers and compares cleanly after
    normalized definer removal.

- `[+]` Add a legacy-auth guard for baseline SQL.
  - Files: `scripts/aofei-local.sh`, `docs/database-baseline.md`.
  - Command to support:
    ```bash
    ./scripts/aofei-local.sh check-sql
    ```
  - Result: command fails on explicit `DEFINER=` clauses or legacy account-name
    references, while allowing `SQL SECURITY DEFINER`.

- `[+]` Document the intentional schema-change workflow.
  - Files: `docs/database-baseline.md`, `memory-bank/tech-stack.md`,
    `AGENTS.md`.
  - Result: future schema edits have a documented path from Docker change to
    baseline update to `reset && load`, `check-sql`, and `diff-schema`.

- `[+]` Run M2 verification.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset
    ./scripts/aofei-local.sh load
    ./scripts/aofei-local.sh check-sql
    ./scripts/aofei-local.sh diff-schema
    GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
    git diff --check
    ```
  - Result: passed on 2026-05-12.

## Review Findings

- `[+]` Make `etc/step4_init.sql` the only baseline loader source. The helper's
  default baseline selection now uses `etc/step4_init.sql`; only the explicit
  `AOFEI_MYSQL_BASELINE_SQL` override can change the loaded file.

- `[X]` Add schema-contract coverage for SQL embedded outside the baseline file.
  Queries in `acl`, `match`, `summer`, and operational commands can drift from
  the Docker schema without being caught by current tests. Moved to M8
  repository test hygiene because it is broader than baseline stewardship.

- `[+]` Keep the SQL guard specific: fail on explicit `DEFINER=` clauses and
  legacy auth references, while allowing intentional `SQL SECURITY DEFINER`
  syntax when it has no user-bound definer.

### Second Review Pass - 2026-05-12

- `[+]` Make baseline loading replay-safe. `scripts/aofei-local.sh load`
  imports SQL into the current database without a reset or duplicate-state
  guard, so reruns can fail partway through or create confusing local state.
  The helper now exits before import when the target database already has schema
  objects; a second `load` returned nonzero with the reset-first message and no
  partial import.

- `[+]` Convert checked-in active config examples into Docker-safe templates or
  quarantine them as historical references. `etc/aofei.json` still points at
  non-Docker Redis/MySQL endpoints and legacy auth, which conflicts with the
  current local database contract. Completed under M1; active configs and
  Genelet fixtures no longer contain legacy local-runtime credentials.

## Verification Notes

- `bash -n scripts/aofei-local.sh`: passed.
- `./scripts/aofei-local.sh check-sql`: passed.
- `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load`: passed.
- `./scripts/aofei-local.sh status`: reported 50 base tables, 1 view,
  6 routines, 18 triggers, 0 events, 1 advertiser, and 14 publishers.
- Replay guard: a second `load` exited nonzero before import with
  `Database aofei already has 75 schema objects; run './scripts/aofei-local.sh reset' before load.`
- `./scripts/aofei-local.sh dump-schema`: wrote ignored
  `.local/schema/aofei.schema.sql`.
- `./scripts/aofei-local.sh diff-schema`: passed with no drift.
