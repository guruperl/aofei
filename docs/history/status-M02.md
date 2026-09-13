# Retired milestone M02 - Schema Baseline Stewardship

**Milestone.** M02
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M02.md
**Source specification.** memory-bank/milestone.md#m02---schema-baseline-stewardship
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M02 - Schema Baseline Stewardship `[+]`

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
````

## Status record

````markdown
# Status M02 - Schema Baseline Stewardship

Milestone status: `[+]` Completed

Goal: Make `etc/step4_init.sql` the durable schema and baseline-data contract.

## Tasks

- `[+]` Document the exact baseline object inventory.
  - Files: `docs/database-baseline.md`, `memory-bank/status-M02.md`.
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

- `[+]` Run M02 verification.
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
  the Docker schema without being caught by current tests. Moved to M08
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
  current local database contract. Completed under M01; active configs and
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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Document the exact baseline object inventory. | `[+]` | |
| Add a schema dump command to the local workflow. | `[+]` | |
| Add a schema comparison command to the local workflow. | `[+]` | |
| Verify table definitions are covered. | `[+]` | |
| Verify views are covered. | `[+]` | |
| Verify routines are covered. | `[+]` | |
| Verify triggers are covered. | `[+]` | |
| Add a legacy-auth guard for baseline SQL. | `[+]` | |
| Document the intentional schema-change workflow. | `[+]` | |
| Run M02 verification. | `[+]` | |
| Make `etc/step4_init.sql` the only baseline loader source. The helper's | `[+]` | |
| Add schema-contract coverage for SQL embedded outside the baseline file. | `[X]` | |
| Keep the SQL guard specific: fail on explicit `DEFINER=` clauses and | `[+]` | |
| Make baseline loading replay-safe. `scripts/aofei-local.sh load` | `[+]` | |
| Convert checked-in active config examples into Docker-safe templates or | `[+]` | |

````
