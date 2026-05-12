# Status M3 - Redis And NATS Cache Pipeline Reliability

Milestone status: `[ ]` Pending

Goal: Prove that cache and message-bus flows work from Docker services.

## Tasks

- `[ ]` Reset the runtime to a known sample state.
  - Files: `scripts/aofei-local.sh`, `etc/step4_init.sql`, `etc/demand.sql`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    ./scripts/aofei-local.sh redis-flush
    ```
  - Acceptance: MySQL sample data exists and Redis starts empty.

- `[ ]` Populate Redis cache only.
  - Files: `cmd/redis-cache/main.go`, `acl/*`, `match/*`, `dsp/config.go`.
  - Command:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    ./scripts/aofei-local.sh redis-status
    ```
  - Acceptance: Redis DB size is greater than zero and the command exits
    without NATS dependency.

- `[ ]` Read Redis cache through application code.
  - Files: `cmd/redis-cache/main.go`, `acl/*`, `match/*`.
  - Command:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis -read
    ```
  - Acceptance: output includes `pubmap`, `Audiences`, and `Creatives`.

- `[ ]` Verify expected Redis key families.
  - Files: `match/*`, `acl/*`, `docs/local-docker-runtime.md`.
  - Command:
    ```bash
    docker exec aofei-redis redis-cli --scan
    ```
  - Acceptance: task notes list the expected key prefixes or names for PubMap,
    RAdv, audience, and creative cache entries.

- `[ ]` Populate spread/NATS path.
  - Files: `cmd/redis-cache/main.go`, `cmd/spread/main.go`,
    `scripts/aofei-local.sh`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh nats-status
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=spread
    ```
  - Acceptance: command connects to Docker NATS and writes spread artifacts
    under the configured local spread directory.

- `[ ]` Populate combined cache mode.
  - Files: `cmd/redis-cache/main.go`, `docs/local-docker-runtime.md`.
  - Command:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=all
    ```
  - Acceptance: both spread and Redis writes complete successfully.

- `[ ]` Add or update a cache smoke script if repeated commands stay manual.
  - Files: `scripts/aofei-cache-smoke.sh` or `scripts/aofei-local.sh`,
    `docs/local-docker-runtime.md`.
  - Acceptance: one command can reset sample data, populate Redis, inspect cache,
    and report pass/fail.

- `[ ]` Document cache inspection expectations.
  - Files: `docs/local-docker-runtime.md`, `memory-bank/architecture.md`.
  - Acceptance: docs explain which commands prove Redis mode, spread mode, and
    NATS connectivity.

- `[ ]` Run M3 verification.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis -read
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=all
    git diff --check
    ```
  - Acceptance: all commands pass against Docker services.

## Review Findings

- `[ ]` Validate `cmd/redis-cache -cache` values explicitly. Today any unknown
  value falls into the combined spread-plus-Redis path, so typos can trigger
  unexpected writes.

- `[ ]` Remove hardcoded RAdv creative sizes from Redis cache population and
  inspection. The command only handles `64x64` and `100x100` even though
  creative size is schema-driven.

- `[ ]` Define a versioned cache payload contract for Redis and spread data.
  Current gob/binary serialization is tied directly to Go struct layout.

- `[ ]` Review spread file writes and cleanup subjects. Cache files are opened
  append-style, and cleanup is encoded as a string suffix rather than a tested
  subject contract.

### Second Review Pass - 2026-05-12

- `[ ]` Prevent NATS callback backpressure in `cmd/nats-client`. Subscription
  callbacks send to unbuffered success/error channels, so log delivery can block
  inside the NATS callback path under traffic or errors.

- `[ ]` Synchronize NATS log file rotation and writes. The log consumer mutates
  shared file handles from the callback path without a lock or single-writer
  queue, leaving rotation/write races untested.

- `[ ]` Make ignored NATS subjects observable. The wildcard subscription sends
  success even for unknown subjects, so dropped traffic is indistinguishable
  from processed request, response, attribute, or win/loss logs.
