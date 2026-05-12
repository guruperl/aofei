# Status M1 - Local Docker Runtime Stabilization

Milestone status: `[+]` Completed

Goal: Make the Docker-backed development runtime the unquestioned local
baseline.

Verified on 2026-05-12.

## Tasks

- `[+]` Verify a clean local service start from stopped containers.
  - Files: `scripts/aofei-local.sh`, `docs/local-docker-runtime.md`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh down
    ./scripts/aofei-local.sh up
    ./scripts/aofei-local.sh status
    ```
  - Result: `down`, `up`, and `status` passed. Status reported MySQL
    `mysql:8.0.41`, Redis `redis:7-alpine`, and NATS `nats:2-alpine` running.

- `[+]` Verify database reset and baseline load on Docker MySQL.
  - Files: `scripts/aofei-local.sh`, `etc/step4_init.sql`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset
    ./scripts/aofei-local.sh load
    ./scripts/aofei-local.sh status
    ```
  - Result: `reset`, `load`, and `status` passed. Status reported 50 base
    tables, 1 view, 6 routines, 1 advertiser, and 14 publishers.

- `[+]` Verify sample data load.
  - Files: `etc/demand.sql`, `etc/main.go`, `scripts/aofei-local.sh`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh sample
    ./scripts/aofei-local.sh status
    ```
  - Result: `sample` and `reset-sample` passed. `etc/demand.sql` was skipped
    because the default sample demand was already present in the baseline, and
    the default publisher helper still ran.

- `[+]` Verify generated local configs are complete and ignored.
  - Files: `.gitignore`, `etc/aofei.local.json`, `etc/summer.local.json`.
  - Command:
    ```bash
    git status --short --ignored etc/aofei.local.json etc/summer.local.json
    ```
  - Result: both generated files exist and `git status --short --ignored`
    reports them as ignored. They point at Docker MySQL
    `127.0.0.1:3307`, Redis `127.0.0.1:6379`, NATS
    `nats://127.0.0.1:4222`, and Summer templates under `.local/templates`.

- `[+]` Confirm no active workflow requires the retired root config directory.
  - Files: `README.md`, `AGENTS.md`, `docs/local-docker-runtime.md`,
    `docs/database-baseline.md`, `scripts/aofei-local.sh`, `etc/main.go`.
  - Command:
    ```bash
    rg -n 'co[n]f/' README.md AGENTS.md docs scripts etc --glob '!docs/legacy-operations.md'
    ```
  - Result: remaining matches are the `AGENTS.md` warnings not to recreate the
    retired root config directory; no active command or config depends on it.

- `[+]` Confirm Docker MySQL auth is the active auth.
  - Files: `scripts/aofei-local.sh`, `etc/*.json`, `docs/*.md`,
    `memory-bank/*.md`.
  - Command:
    ```bash
    rg -n 'eightr[a]n' . --glob '!backup/**' --glob '!docs/legacy-operations.md'
    ```
  - Result: checked-in active configs and Genelet fixtures no longer contain
    legacy DSNs. Remaining matches are warnings, historical notes, or
    `scripts/aofei-local.sh` cleanup statements that drop legacy users.

- `[+]` Verify Redis and NATS direct status commands.
  - Files: `scripts/aofei-local.sh`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh redis-status
    ./scripts/aofei-local.sh nats-status
    ```
  - Result: both commands passed. Redis reported keyspace/DB size, and NATS
    returned `/varz` JSON.

- `[+]` Update runtime documentation after verification.
  - Files: `README.md`, `docs/local-docker-runtime.md`,
    `memory-bank/tech-stack.md`, `memory-bank/status-M1.md`.
  - Result: README, local runtime docs, and database baseline docs now document
    deterministic sample loading, generated config endpoints, the
    lower-case/upper-case config distinction, and the M1 Summer template path
    boundary.

- `[+]` Run M1 verification.
  - Command:
    ```bash
    git diff --check
    GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
    ```
  - Result: `git diff --check` passed. The scoped Go smoke command passed for
    `cmd/redis-cache`, `cmd/nats-client`, `cmd/spread`, `etc`, `dsp`, `acl`,
    and `match`.

## Review Findings

- `[+]` Remove active local-runtime reliance on legacy database auth. Review
  `etc/aofei.json`, generated local configs, scripts, and docs so Docker MySQL
  auth is the only current development path.

- `[+]` Audit tracked runtime configs for credential-like values before they are
  treated as active templates. Cover `etc/summer.json`, `genelet/test.conf`, and
  SMTP/AWS-like test fixtures without copying secrets into documentation.

- `[+]` Fix or document the generated Summer template path. The helper writes a
  `Template` path under `tmpls`, but that directory is not present in the
  package.

- `[+]` Decide the local config contract across DSP, Summer, and Genelet.
  Current tests mix lower-case DSP config JSON with Genelet's `ConnectArray`
  config shape, which makes local runtime verification ambiguous.

### Second Review Pass - 2026-05-12

- `[+]` Make DSP config defaulting nil-safe. `dsp.NewConfig` dereferences
  `parsed.Redis` even when a config omits the `redis` block, so minimal configs
  can panic before validation returns an actionable error.

- `[+]` Add usage/error handling to the `etc` helper command. `go run ./etc`
  indexes `os.Args[1]` directly, so missing or unsupported subcommands panic
  instead of explaining supported setup actions.

- `[+]` Make `reset-sample` deterministic. The sample loader skips
  `etc/demand.sql` whenever `adv` already has rows, but the schema baseline can
  already contain advertiser rows, making sample-demand loading a no-op.

## Verification Notes

- `GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis`
  passed after `reset-sample`; Redis status reported 4 keys.
- `GOWORK=off go run ./etc` prints usage and exits without panic.
- `GOWORK=off go run ./etc unknown` prints an unknown-command error and usage
  instead of panicking.
- `GOWORK=off go test ./genelet -run '^TestConfig$'` passed after isolating the
  Genelet fixture from legacy DB/auth keys.
- Follow-up review findings were closed by making `dsp` config tests
  self-contained and by removing the stale root SQL baseline fallback.
