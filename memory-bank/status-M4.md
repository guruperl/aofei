# Status M4 - Bid Path Smoke Coverage

Milestone status: `[ ]` Pending

Goal: Add a reliable local proof that the DSP request path still works.

## Tasks

- `[ ]` Inventory available OpenRTB sample inputs and expected outputs.
  - Files: `etc/samples/*`, `dsp/*`, `match/*`.
  - Command:
    ```bash
    find etc/samples -maxdepth 1 -type f | sort
    ```
  - Acceptance: samples are mapped to bid request, response, win, native, and
    targeting fixture roles.

- `[ ]` Identify the narrowest bid-path function for a deterministic smoke test.
  - Files: `dsp/controller.go`, `dsp/dsp.go`, `dsp/dsp_test.go`,
    `dsp/config.go`.
  - Acceptance: test entrypoint is chosen without requiring a live HTTP server
    unless the controller design makes that the cleanest proof.

- `[ ]` Build a seeded runtime fixture setup.
  - Files: `dsp/dsp_test.go`, `etc/samples/*`, `scripts/aofei-local.sh`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    ```
  - Acceptance: the test can rely on Docker MySQL/Redis state created by
    documented commands.

- `[ ]` Add a smoke test for valid bid request handling.
  - Files: `dsp/dsp_test.go` or a new focused test file under `dsp/`.
  - Acceptance: test reads a sample request, calls the selected bid path, and
    asserts a valid response shape or a documented no-bid result.

- `[ ]` Add a smoke test for config loading.
  - Files: `dsp/config_test.go`, `etc/aofei.local.json`.
  - Acceptance: test verifies MySQL, Redis, NATS, spread, and log path fields
    needed by the bid path are present when using generated local config.

- `[ ]` Add a smoke test for cache-dependent matching.
  - Files: `dsp/*`, `match/*`, `acl/*`.
  - Acceptance: test demonstrates the bid path can read the Redis cache
    structures populated in M3.

- `[ ]` Add a failure-mode smoke test for missing cache or malformed request.
  - Files: `dsp/*_test.go`.
  - Acceptance: test returns a controlled no-bid/error path rather than a panic.

- `[ ]` Document the bid-path smoke command.
  - Files: `README.md`, `docs/local-docker-runtime.md`,
    `memory-bank/tech-stack.md`.
  - Command to document:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
    ```
  - Acceptance: docs identify prerequisites and expected pass/fail boundaries.

- `[ ]` Run M4 verification.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
    git diff --check
    ```
  - Acceptance: bid-path smoke tests pass from the seeded Docker runtime.

## Review Findings

- `[ ]` Harden bid request parsing for missing device, missing impressions, and
  partial user identifiers. `match/attribute.go` can return a nil attribute or
  index `Imp[0]`, and anonymous requests with only IFA can share an empty fcap
  Redis key.

- `[ ]` Normalize generated user identifiers. The fallback MD5 value is
  converted from raw bytes to string instead of a stable text encoding.

- `[ ]` Guard `dsp.ServeBid` dependencies. Offline or test controllers can have
  nil logger or NATS client fields even though the bid path assumes both.

- `[ ]` Guard macro expansion against incomplete OpenRTB requests. Macro code
  assumes device, geo, app, and first impression data are always present.

- `[ ]` Fix fcap refresh semantics. Click-cap reset checks the impression
  period, and the must-refresh path updates counters without period-expiry
  handling.

- `[ ]` Split bid orchestration into testable units. `ServeBid` currently mixes
  HTTP parsing, pub lookup, geo/user-agent matching, audience checks, fcap,
  creative load, response writing, and NATS logging in one method.
