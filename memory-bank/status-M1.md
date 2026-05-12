# Status M1 - Local Docker Runtime Stabilization

Milestone status: `[ ]` Pending

Goal: Make the Docker-backed development runtime the unquestioned local
baseline.

## Tasks

- `[ ]` Verify a clean local service start from stopped containers.
  - Files: `scripts/aofei-local.sh`, `docs/local-docker-runtime.md`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh down
    ./scripts/aofei-local.sh up
    ./scripts/aofei-local.sh status
    ```
  - Acceptance: MySQL, Redis, and NATS containers are running and status returns
    without errors.

- `[ ]` Verify database reset and baseline load on Docker MySQL.
  - Files: `scripts/aofei-local.sh`, `etc/step4_init.sql`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset
    ./scripts/aofei-local.sh load
    ./scripts/aofei-local.sh status
    ```
  - Acceptance: the `aofei` database exists, object counts are reported, and no
    legacy MySQL user is required.

- `[ ]` Verify sample data load.
  - Files: `etc/demand.sql`, `etc/main.go`, `scripts/aofei-local.sh`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh sample
    ./scripts/aofei-local.sh status
    ```
  - Acceptance: sample advertiser and publisher counts are nonzero or the
    command reports an intentional idempotent skip.

- `[ ]` Verify generated local configs are complete and ignored.
  - Files: `.gitignore`, `etc/aofei.local.json`, `etc/summer.local.json`.
  - Command:
    ```bash
    git status --short --ignored etc/aofei.local.json etc/summer.local.json
    ```
  - Acceptance: both files exist locally, are ignored by git, and include Docker
    MySQL, Redis, and NATS endpoints.

- `[ ]` Confirm no active workflow requires `conf/`.
  - Files: `README.md`, `AGENTS.md`, `docs/local-docker-runtime.md`,
    `docs/database-baseline.md`, `scripts/aofei-local.sh`, `etc/main.go`.
  - Command:
    ```bash
    rg -n 'conf/' README.md AGENTS.md docs scripts etc --glob '!docs/legacy-operations.md'
    ```
  - Acceptance: any remaining `conf/` reference is explicitly historical or a
    warning not to recreate it.

- `[ ]` Confirm Docker MySQL auth is the active auth.
  - Files: `scripts/aofei-local.sh`, `etc/*.json`, `docs/*.md`,
    `memory-bank/*.md`.
  - Command:
    ```bash
    rg -n 'eightran' . --glob '!backup/**' --glob '!docs/legacy-operations.md'
    ```
  - Acceptance: no active runtime file requires `eightran_*`; references are
    limited to warnings or historical notes.

- `[ ]` Verify Redis and NATS direct status commands.
  - Files: `scripts/aofei-local.sh`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh redis-status
    ./scripts/aofei-local.sh nats-status
    ```
  - Acceptance: Redis keyspace/DB size and NATS `/varz` output are visible.

- `[ ]` Update runtime documentation after verification.
  - Files: `README.md`, `docs/local-docker-runtime.md`,
    `memory-bank/tech-stack.md`, `memory-bank/status-M1.md`.
  - Acceptance: command output expectations and any discovered deviations are
    documented.

- `[ ]` Run M1 verification.
  - Command:
    ```bash
    git diff --check
    GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
    ```
  - Acceptance: both commands pass after documentation and script updates.

## Review Findings

- `[ ]` Remove active local-runtime reliance on legacy `eightran_*` database
  auth. Review `etc/aofei.json`, generated local configs, scripts, and docs so
  Docker MySQL auth is the only current development path.

- `[ ]` Audit tracked runtime configs for credential-like values before they are
  treated as active templates. Cover `etc/summer.json`, `genelet/test.conf`, and
  SMTP/AWS-like test fixtures without copying secrets into documentation.

- `[ ]` Fix or document the generated Summer template path. The helper writes a
  `Template` path under `tmpls`, but that directory is not present in the
  package.

- `[ ]` Decide the local config contract across DSP, Summer, and Genelet.
  Current tests mix lower-case DSP config JSON with Genelet's `ConnectArray`
  config shape, which makes local runtime verification ambiguous.

### Second Review Pass - 2026-05-12

- `[ ]` Make DSP config defaulting nil-safe. `dsp.NewConfig` dereferences
  `parsed.Redis` even when a config omits the `redis` block, so minimal configs
  can panic before validation returns an actionable error.

- `[ ]` Add usage/error handling to the `etc` helper command. `go run ./etc`
  indexes `os.Args[1]` directly, so missing or unsupported subcommands panic
  instead of explaining supported setup actions.

- `[ ]` Make `reset-sample` deterministic. The sample loader skips
  `etc/demand.sql` whenever `adv` already has rows, but the schema baseline can
  already contain advertiser rows, making sample-demand loading a no-op.
