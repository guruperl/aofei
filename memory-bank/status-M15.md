# Status M15 - DSP Serving Hardening

## Goal

Resolve the post-M14 DSP serving review findings around tracker integrity,
request-path cache IO, cap refresh atomicity, audit publish latency, URL macro
preservation, and staticcheck hygiene.

## Completed

- `[+]` Tracker and click integrity.
  - Added `tracking_secret` with `TRACKING_SECRET` fallback.
  - DSP-generated `/imp` and `/clk` URLs are HMAC signed over concrete query
    payloads, including click `redirect`.
  - `/win` and `/loss` URLs sign immutable packed fields while leaving exchange
    auction macros replaceable.
  - `/clk` redirects and `/imp`/`/clk` cap mutations require valid signatures.
  - Files: `dsp/config.go`, `dsp/tracking.go`, `dsp/winloss.go`,
    `dsp/dsp.go`, `dsp/controller.go`, `etc/aofei.json`,
    `scripts/aofei-local.sh`.

- `[+]` Static cache hot path.
  - Local/spread static snapshots load at controller startup when
    `is_local=true`.
  - Request-path publisher, slot, audience, and creative getters read only the
    current in-memory snapshot.
  - Added an explicit local static-cache reload hook.
  - Files: `dsp/local_cache.go`, `dsp/local_cache_test.go`,
    `dsp/m13_test.go`.

- `[+]` Frequency caps.
  - `match.MustRefreshBothCap` now uses Redis `WATCH`, `HGET`, `MULTI`, `HSET`,
    `EXEC`, and bounded retry through `radix.WithConn`.
  - Kept the existing `bothcap:<user_id>` hash and binary `BothCap` payload.
  - Files: `match/fcap.go`, `match/fcap_test.go`.

- `[+]` NATS audit publishing.
  - Added a bounded async audit queue owned by `dsp.Controller`.
  - `ServeBid` enqueues request/response/attribute audit logs after writing the
    HTTP response and no longer flushes NATS in the request goroutine.
  - Queue drops are counted and controller close drains cleanly.
  - Files: `dsp/audit.go`, `dsp/controller.go`, `dsp/controller_test.go`.

- `[+]` URL macro expansion.
  - Creative URL macro replacement now preserves existing query parameters,
    repeated values, empty values, and non-macro values while replacing macros
    per query value.
  - Files: `match/creative.go`, `match/creative_m13_test.go`.

- `[+]` Staticcheck cleanup.
  - Replaced the test-only nil context in `dsp/controller_test.go` with
    `context.TODO()`.

- `[+]` Docs and memory bank.
  - Updated measurement, workflow, cache, audience, local runtime, production
    runbook, architecture, tech-stack, milestone, and M14 carry-forward docs.
  - Files: `docs/openrtb-measurement.md`, `docs/dsp-workflow.md`,
    `docs/multiple-cache.md`, `docs/audience-matching.md`,
    `docs/local-docker-runtime.md`, `docs/production-runbook.md`,
    `memory-bank/architecture.md`, `memory-bank/tech-stack.md`,
    `memory-bank/milestone.md`, `memory-bank/status-M14.md`.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./match ./uploaded ./acl ./cmd/spread ./cmd/winloss`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
