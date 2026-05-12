# Status M9 - Production Deployment Runbook

Milestone status: `[+]` Completed

Goal: Rebuild production/operator docs from current reality rather than legacy
notes.

## Tasks

- `[+]` Review historical deployment notes.
  - Files: `docs/legacy-operations.md`, `backup/*`.
  - Command:
    ```bash
    sed -n '1,220p' docs/legacy-operations.md
    find backup -maxdepth 2 -type f | sort
    ```
  - Acceptance: historical NATS, Redis, systemd, config, test DB, and TDengine
    notes are categorized as current, obsolete, or unknown.

- `[+]` Define production service inventory.
  - Files: `docs/production-runbook.md`, `memory-bank/architecture.md`.
  - Acceptance: runbook identifies service binaries, MySQL, Redis, NATS,
    MaxMind assets, log directories, upload directories, and ports.

- `[+]` Define production configuration strategy.
  - Files: `docs/production-runbook.md`, `etc/aofei.json`,
    `etc/summer.json`.
  - Acceptance: runbook explains config templates, secret injection, local
    generated configs, and prohibited committed secrets.

- `[+]` Define production database lifecycle.
  - Files: `docs/production-runbook.md`, `docs/database-baseline.md`.
  - Acceptance: runbook covers baseline load, migrations/schema updates,
    backups, restores, and drift checks without embedding credentials.

- `[+]` Define production Redis and NATS lifecycle.
  - Files: `docs/production-runbook.md`, `docs/local-docker-runtime.md`.
  - Acceptance: runbook separates local Docker services from production service
    ownership, persistence, monitoring, and restart behavior.

- `[+]` Define deployment and rollback flow.
  - Files: `docs/production-runbook.md`.
  - Acceptance: runbook lists build command, artifact location, service restart,
    smoke checks, and rollback criteria.

- `[+]` Define observability and log handling.
  - Files: `docs/production-runbook.md`, `cmd/nats-client/*`, `cmd/ledger/*`.
  - Acceptance: request, response, attribute, win/loss, NATS, and ledger flows
    are documented at operator level.

- `[+]` Retire or isolate obsolete historical notes.
  - Files: `docs/legacy-operations.md`, `docs/production-runbook.md`.
  - Acceptance: root README links only to current runbooks; historical docs are
    clearly labeled and not required for active setup.

- `[+]` Run M9 verification.
  - Command:
    ```bash
    rg -n 'eightr[a]n|12pass3[4]|co[n]f/' README.md AGENTS.md docs memory-bank --glob '!docs/legacy-operations.md'
    git diff --check
    ```
  - Acceptance: no active production/local docs rely on legacy credentials or
    retired config directories.

## Review Findings

- `[+]` Remove or quarantine credential-like values from tracked active files.
  Cover runtime configs and Genelet SMTP/AWS-like test fixtures without copying
  sensitive values into the runbook.

- `[+]` Separate local Docker assumptions from production service ownership.
  Production docs need explicit MySQL, Redis, NATS, MaxMind, log, and restart
  responsibilities.

- `[+]` Add secret-handling guidance for config templates, generated local
  files, and production injection so future agents do not commit live values.

- `[+]` Add graceful shutdown expectations for server commands. Current command
  review did not find a documented signal/shutdown contract.

### Second Review Pass - 2026-05-12

- `[+]` Remove active tracked credential-like values and legacy DSNs from config
  examples and tests, or clearly quarantine them as historical-only fixtures.

- `[+]` Define a production CORS policy for the admin service. Genelet currently
  reflects any request `Origin` while allowing credentials.

- `[+]` Fix static-file path rejection. The static handler writes `404` for
  paths containing `..` but does not return before calling `ServeFile`.

- `[+]` Make legacy password hashing an explicit migration decision. Summer
  login/reset paths still use SHA1-era password hashes that should not be
  treated as the long-term production auth contract.
