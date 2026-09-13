# Retired milestone M15 - DSP Serving Hardening

**Milestone.** M15
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M15.md
**Source specification.** memory-bank/milestone.md#m15---dsp-serving-hardening
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M15 - DSP Serving Hardening `[+]`

Resolve the post-M14 DSP serving review findings without reopening M14.

Scope:

- Sign DSP-generated tracker and click URLs and require valid signatures for
  click redirects and cap mutations.
- Remove request-path filesystem generation checks from local static cache
  reads.
- Make frequency-cap refresh atomic while preserving Redis key and payload
  shape.
- Move request/response/attribute audit publishing off the request goroutine.
- Preserve creative URL query parameters and repeated values during macro
  replacement.
- Clean the focused staticcheck finding in DSP tests.

Acceptance:

- Focused tests cover signed click redirect acceptance/rejection, local static
  cache read behavior, async audit queue drops, repeated-query macro expansion,
  and cap refresh behavior.
- Measurement, workflow, cache, audience, and memory-bank docs describe the new
  serving contracts.
````

## Status record

````markdown
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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Tracker and click integrity. | `[+]` | |
| Static cache hot path. | `[+]` | |
| Frequency caps. | `[+]` | |
| NATS audit publishing. | `[+]` | |
| URL macro expansion. | `[+]` | |
| Staticcheck cleanup. | `[+]` | |
| Docs and memory bank. | `[+]` | |
| `GOWORK=off go test ./dsp ./match ./uploaded ./acl ./cmd/spread ./cmd/winloss` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |

````
