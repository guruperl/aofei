# Status M4 - Bid Path Smoke Coverage

Milestone status: `[+]` Completed

Goal: Add a reliable local proof that the DSP request path still works.

## Tasks

- `[+]` Inventory available OpenRTB sample inputs and expected outputs.
  - Files: `etc/samples/*`, `dsp/*`, `match/*`.
  - Command:
    ```bash
    find etc/samples -maxdepth 1 -type f | sort
    ```
  - Result: `sample_bid.json` is the active bid-request fixture,
    `sample_resp.json` is historical response shape reference,
    `sample_adm.json` and `sample_native.json` cover native/ad markup,
    `sample_win.json` is win-log reference text, and `aofei_*.json` files are
    targeting value references.

- `[+]` Identify the narrowest bid-path function for a deterministic smoke test.
  - Files: `dsp/controller.go`, `dsp/dsp.go`, `dsp/dsp_test.go`,
    `dsp/config.go`.
  - Result: M4 uses `httptest` against `dsp.Controller.ServeBid`, not a live
    HTTP server and not a new exported orchestration layer.

- `[+]` Build a seeded runtime fixture setup.
  - Files: `dsp/dsp_test.go`, `etc/samples/*`, `scripts/aofei-local.sh`.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    ```
  - Result: the smoke relies on `reset-sample` plus
    `go run ./cmd/redis-cache -cache=redis`; tests fail when `AOFEI` points at a
    config but Redis/cache prerequisites are absent.

- `[+]` Add a smoke test for valid bid request handling.
  - Files: `dsp/dsp_test.go` or a new focused test file under `dsp/`.
  - Result: `TestServeBidSmoke` reads `etc/samples/sample_bid.json`, clears
    `device.ip`, sets path domain `default`, and asserts a
    valid OpenRTB bid response.

- `[+]` Add a smoke test for config loading.
  - Files: `dsp/config_test.go`, `etc/aofei.local.json`.
  - Result: `TestLocalConfigSmoke` verifies generated AOFEI MySQL, Redis, NATS,
    spread, and log-path fields.

- `[+]` Add a smoke test for cache-dependent matching.
  - Files: `dsp/*`, `match/*`, `acl/*`.
  - Result: `TestServeBidSmoke` exercises pubmap, slot/radv, audience, and
    creative reads from Docker Redis.

- `[+]` Add a failure-mode smoke test for missing cache or malformed request.
  - Files: `dsp/*_test.go`.
  - Result: unknown publisher returns `204`, malformed JSON returns `400`, and
    oversized bodies return `413`.

- `[+]` Document the bid-path smoke command.
  - Files: `README.md`, `docs/local-docker-runtime.md`,
    `memory-bank/tech-stack.md`.
  - Command to document:
    ```bash
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
    ```
  - Result: command and prerequisites are documented in README,
    `docs/local-docker-runtime.md`, and `memory-bank/tech-stack.md`.

- `[+]` Run M4 verification.
  - Command:
    ```bash
    ./scripts/aofei-local.sh reset-sample
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/redis-cache -cache=redis
    GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
    git diff --check
    ```
  - Result: verification passed on 2026-05-12.

## Review Findings

- `[+]` Harden bid request parsing for missing device, missing impressions, and
  partial user identifiers. `match/attribute.go` can return a nil attribute or
  index `Imp[0]`, and anonymous requests with only IFA can share an empty fcap
  Redis key.

- `[+]` Normalize generated user identifiers. The fallback MD5 value is
  converted from raw bytes to string instead of a stable text encoding.

- `[+]` Guard `dsp.ServeBid` dependencies. Offline or test controllers can have
  nil logger or NATS client fields even though the bid path assumes both.

- `[+]` Guard macro expansion against incomplete OpenRTB requests. Macro code
  assumes device, geo, app, and first impression data are always present.

- `[+]` Fix fcap refresh semantics. Click-cap reset checks the impression
  period, and the must-refresh path updates counters without period-expiry
  handling.

- `[X]` Split bid orchestration into testable units. `ServeBid` currently mixes
  HTTP parsing, pub lookup, geo/user-agent matching, audience checks, fcap,
  creative load, response writing, and NATS logging in one method.
  - Decision: deferred beyond M4; the smoke proof uses `httptest` without adding
    a new exported orchestration API.

### Second Review Pass - 2026-05-12

- `[+]` Limit bid request body size before reading. `ServeBid` currently uses
  `io.ReadAll` on the HTTP body, so public bid traffic can force unbounded
  memory allocation.

- `[+]` Decide the audit-log failure contract for accepted bids. `ServeBid`
  writes the OpenRTB response before publishing request, response, and attribute
  logs to NATS, so a bidder can receive `200 OK` while audit logs are lost.
  - Decision: NATS audit publish is best-effort and must not alter an already
    accepted bid response.

- `[+]` Harden win/loss URL macro expansion for no-bid responses.
  `dsp.UnpackURLString` assumes `SeatBid[0].Bid[0]` exists and can panic on
  empty or malformed simulator responses.

- `[+]` Fix the `WinLoss.BothCap` JSON tag. Staticcheck flags
  `json:"-,omitempty"` as invalid; use `json:"-"` if the cap should never be
  serialized.

## Milestone Review - 2026-05-12

- `[+]` Reviewed the M4 implementation after verification. No blocking
  regressions found in the smoke harness, nil dependency handling, no-bid
  handling, macro guards, upload-audience default behavior, or frequency-cap
  refresh changes.
