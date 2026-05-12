# Status M2 - Schema Baseline Stewardship

Milestone status: `[ ]` Pending

Goal: Make `etc/step4_init.sql` the durable schema and baseline-data contract.

## Tasks

- `[ ]` Document the exact baseline object inventory.
  - Files: `docs/database-baseline.md`, `memory-bank/status-M2.md`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset
    ./scripts/aofei-local.sh load
    ./scripts/aofei-local.sh status
    ```
  - Acceptance: expected table, view, routine, trigger, advertiser, and
    publisher counts are recorded.

- `[ ]` Add a schema dump command to the local workflow.
  - Files: `scripts/aofei-local.sh`, `docs/database-baseline.md`.
  - Command to support:
    ```bash
    ./scripts/aofei-local.sh dump-schema
    ```
  - Acceptance: command writes a normalized schema dump to a temporary or
    ignored path without modifying `etc/step4_init.sql`.

- `[ ]` Add a schema comparison command to the local workflow.
  - Files: `scripts/aofei-local.sh`, `docs/database-baseline.md`.
  - Command to support:
    ```bash
    ./scripts/aofei-local.sh diff-schema
    ```
  - Acceptance: command compares Docker MySQL schema against a database rebuilt
    from `etc/step4_init.sql` and exits nonzero on drift.

- `[ ]` Verify table definitions are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    ./scripts/aofei-local.sh diff-schema
    ```
  - Acceptance: base table definitions compare cleanly or differences are
    listed in the task file.

- `[ ]` Verify views are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    rg -n '^CREATE .*VIEW|^CREATE VIEW' etc/step4_init.sql
    ```
  - Acceptance: views in Docker MySQL and baseline SQL match after normalized
    definer removal.

- `[ ]` Verify routines are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    rg -n '^CREATE .*PROCEDURE|^CREATE .*FUNCTION' etc/step4_init.sql
    ```
  - Acceptance: routines in Docker MySQL and baseline SQL match after normalized
    definer removal.

- `[ ]` Verify triggers are covered.
  - Files: `etc/step4_init.sql`, generated schema dump.
  - Command:
    ```bash
    rg -n '^CREATE .*TRIGGER' etc/step4_init.sql
    ```
  - Acceptance: triggers in Docker MySQL and baseline SQL match after normalized
    definer removal.

- `[ ]` Add a legacy-auth guard for baseline SQL.
  - Files: `scripts/aofei-local.sh`, `docs/database-baseline.md`.
  - Command to support:
    ```bash
    ./scripts/aofei-local.sh check-sql
    ```
  - Acceptance: command fails if `etc/step4_init.sql` contains explicit
    `DEFINER=` clauses or `eightran` references.

- `[ ]` Document the intentional schema-change workflow.
  - Files: `docs/database-baseline.md`, `memory-bank/tech-stack.md`,
    `AGENTS.md`.
  - Acceptance: future schema edits have a single documented path from Docker
    change to baseline update to verification.

- `[ ]` Run M2 verification.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset
    ./scripts/aofei-local.sh load
    ./scripts/aofei-local.sh check-sql
    ./scripts/aofei-local.sh diff-schema
    git diff --check
    ```
  - Acceptance: all commands pass or any drift is intentionally captured before
    closing the milestone.

## Review Findings

- `[ ]` Make `etc/step4_init.sql` the only baseline loader source. The helper's
  baseline selection still silently prefers a root `ref.sql` if that file
  reappears.

- `[ ]` Add schema-contract coverage for SQL embedded outside the baseline file.
  Queries in `acl`, `match`, `summer`, and operational commands can drift from
  the Docker schema without being caught by current tests.

- `[ ]` Keep the SQL guard specific: fail on explicit `DEFINER=` clauses and
  legacy auth references, while allowing intentional `SQL SECURITY DEFINER`
  syntax when it has no user-bound definer.
