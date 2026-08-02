# Status M03 - Redis And NATS Cache Pipeline Reliability

Milestone status: `[+]` Completed

Goal: Prove that cache and message-bus flows work from Docker services.

## Tasks

- `[+]` Reset the runtime to a known sample state.
  - Files: `scripts/aofei-local.sh`, `etc/step4_init.sql`, `etc/demand.sql`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    ./scripts/aofei-local.sh redis-flush
    ```
  - Result: sample MySQL data loads deterministically and Redis starts empty.

- `[+]` Populate Redis cache only.
  - Files: `cmd/redis-cache/main.go`, `acl/*`, `match/*`, `dsp/config.go`.
  - Command:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    ./scripts/aofei-local.sh redis-status
    ```
  - Result: Redis cache population exits without NATS and writes cache keys.

- `[+]` Read Redis cache through application code.
  - Files: `cmd/redis-cache/main.go`, `acl/*`, `match/*`.
  - Command:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis -read
    ```
  - Result: output includes `pubmap`, `Audiences`, `Creatives`, and RAdv
    output for active creative size IDs discovered from MySQL.

- `[+]` Verify expected Redis key families.
  - Files: `match/*`, `acl/*`, `docs/local-docker-runtime.md`.
  - Command:
    ```bash
    docker exec aofei-redis redis-cli --scan
    ```
  - Result: expected families are documented as `pubmap`, `audience`,
    `creative`, and `slot:<size_id>` hashes.

- `[+]` Populate spread/NATS path.
  - Files: `cmd/redis-cache/main.go`, `cmd/spread/main.go`,
    `scripts/aofei-cache-smoke.sh`.
  - Command:
    ```bash
    ./scripts/aofei-cache-smoke.sh
    ```
  - Result: the helper starts `cmd/spread`, publishes spread cache messages to
    Docker NATS, and verifies `.local/spread/` artifact families.

- `[+]` Populate combined cache mode.
  - Files: `cmd/redis-cache/main.go`, `docs/local-docker-runtime.md`.
  - Command:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=all
    ```
  - Result: combined mode publishes spread/NATS messages and writes Redis.

- `[+]` Add or update a cache smoke script if repeated commands stay manual.
  - Files: `scripts/aofei-cache-smoke.sh`, `docs/local-docker-runtime.md`.
  - Result: one command resets sample data, populates Redis, inspects cache
    output, starts spread, publishes spread/all modes, and reports pass/fail.

- `[+]` Document cache inspection expectations.
  - Files: `docs/local-docker-runtime.md`, `memory-bank/architecture.md`.
  - Result: docs explain Redis mode, spread mode, combined mode, expected key
    families, spread artifact families, and the spread receiver requirement.

- `[+]` Run M03 verification.
  - Command:
    ```bash
    bash -n scripts/aofei-cache-smoke.sh
    GOWORK=off go test ./cmd/redis-cache ./cmd/spread -run '^$'
    GOWORK=off go test ./cmd/spread -run 'Test'
    ./scripts/aofei-cache-smoke.sh
    ./scripts/aofei-local.sh reset-sample
    ./scripts/aofei-local.sh redis-flush
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis -read
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=all
    GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
    git diff --check
    ```
  - Result: all commands passed on 2026-05-12.

## Review Findings

- `[+]` Validate `cmd/redis-cache -cache` values explicitly. Unknown values now
  exit nonzero before config loading or cache writes.

- `[+]` Remove hardcoded RAdv creative sizes from Redis cache population and
  inspection. `cmd/redis-cache` now discovers active creative size IDs from
  MySQL with active advertiser/campaign/item/creative and item date filters.

- `[X]` Define a versioned cache payload contract for Redis and spread data.
  Deferred from M03 because the current milestone is cache-pipeline reliability,
  not a wire-format migration. This remains tracked as an architecture gap.

- `[+]` Review spread file writes and cleanup subjects. `cmd/spread` now routes
  subjects through a tested helper, writes full message snapshots, supports
  `DELETE`, preserves `slot:<size_id>:<slot_id>cleanup`, and ignores log or
  unsupported subjects.

### Second Review Pass - 2026-05-12

- `[X]` Move `cmd/nats-client` callback/backpressure, rotation, and
  ignored-subject observability findings to M06. These are operational-log
  reliability issues rather than M03 cache-population requirements.

## Milestone Review

- `[+]` Deep review completed on 2026-05-12. Reviewed explicit cache mode
  validation, active creative size discovery, Redis/spread RAdv loops, spread
  snapshot/cleanup/delete/ignore behavior, smoke-script cleanup, and docs.
  Residual versioned payload contracts remain an architecture gap, and
  `cmd/nats-client` operational-log findings remain in M06.
