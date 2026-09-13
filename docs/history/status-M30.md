# Retired milestone M30 - SSP Measurement, Cookie, And Reporting Semantics

**Milestone.** M30
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M30.md
**Source specification.** memory-bank/milestone.md#m30---ssp-measurement-cookie-and-reporting-semantics
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M30 - SSP Measurement, Cookie, And Reporting Semantics `[+]`

Make direct SSP traffic observable and compatible with existing ledgers.

Scope:

- Identify direct SSP traffic separately from ADX `/bid` in request, response,
  and attribute audits.
- Add best-effort browser user-cookie handling with IP+UA fallback.
- Verify `/imp` and `/clk` records from SSP markup feed existing win/loss and
  ledger inputs.
- Add direct web tag, app-like API, partial-fill, all-no-fill, and invalid-token
  smoke fixtures.
- Document direct SSP measurement behavior.

Acceptance:

- Operators can distinguish ADX and direct SSP traffic in logs/audits.
- Existing ledger aggregation continues without schema change unless a concrete
  reporting gap is found.
- Browser cookie absence does not break serving.

Result:

- `/pz` request and response audits use explicit SSP envelopes with
  `source:"ssp"` and `contract:"pz-v1"` while ADX `/bid` request/response audits
  remain raw OpenRTB payloads.
- Attribute audits include additive `source` and `contract` fields for ADX and
  SSP traffic.
- `/pz` uses a valid browser-only `aofei_pz_uid` cookie as OpenRTB user identity
  when present; missing or invalid browser cookies keep current-request serving
  on the existing IP+UA fallback and set a best-effort cookie for later browser
  requests. `platform:"sdk"` requests do not read, set, rotate, or propagate the
  cookie.
- SSP markup reuses the existing signed `/imp` and `/clk` tracker path, and
  tracker `WinLoss` records aggregate through the current ledger schema.
````

## Status record

````markdown
# Status M30 - SSP Measurement, Cookie, And Reporting Semantics

Current note: S01 supersedes the historical privacy behavior below. Browser
identity now requires an accepted configured personalization grant, IP+UA is no
longer an identity fallback, and all request/attribute audits are redacted.

## Goal

Make direct SSP traffic observable in audits, give browser traffic best-effort
cookie identity without breaking IP+UA fallback, and prove SSP trackers feed the
existing win/loss and ledger path without schema changes.

## Tasks

- `[+]` Create the M30 status file.

- `[+]` Identify SSP traffic separately from ADX audits.
  - ADX `/bid` request and response audit payloads remain raw OpenRTB JSON.
  - SSP `/pz` request and response audits use a `source:"ssp"` and
    `contract:"pz-v1"` JSON envelope.
  - Attribute audits include additive `source` and `contract` fields for both
    ADX and SSP traffic.

- `[+]` Add best-effort `/pz` browser cookie handling.
  - `/pz` accepts a valid `aofei_pz_uid` cookie as OpenRTB `user.id` and
    `buyeruid` only for browser-cookie traffic: missing or empty `platform` and
    `platform:"browser"`.
  - Missing or invalid browser cookies leave the current request on the existing
    IP+UA fallback path and set or rotate a best-effort cookie for later browser
    requests.
  - `platform:"sdk"` requests do not read, set, rotate, or propagate the
    browser cookie.
  - `ads.js` remains credentialless and `/pz` CORS remains permissive but not
    credentialed; origin/referrer policy is still M31.

- `[+]` Verify SSP markup trackers feed measurement.
  - Filled SSP markup carries signed `/imp` and `/clk` URLs from the existing
    `NewBid` rendering path.
  - `/imp` and `/clk` publish `WinLoss` records with direct publisher/site/slot
    and demand IDs.
  - Existing ledger aggregation counts SSP `StatusTrackImp` and
    `StatusTrackClk` records without schema changes.

- `[+]` Add direct SSP smoke fixtures.
  - Direct web tag and app-like API request shapes.
  - Partial-fill and all-no-fill responses.
  - Invalid-token rejection with no audit publish.

- `[+]` Update SSP measurement docs and memory bank files.

- `[+]` Run required verification and record results.
  - Checked `evolution/`; no new version is needed because M30 implements the
    already planned direct SSP measurement/cookie/reporting semantics without
    changing the product boundary.

## Acceptance

- `[+]` Operators can distinguish ADX and direct SSP traffic in logs/audits.
- `[+]` Existing ledger aggregation continues without schema change.
- `[+]` Browser cookie absence does not break serving.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./match ./internal/jobs/ledger ./internal/jobs/cache -count=1`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `(cd ../pzdesign && GOWORK=off go test ./cmd/unify ./summer/slot -count=1)`
- `[+]` `(cd ../pzdesign && GOWORK=off go test ./...)`
- `[+]` `(cd ../pzdesign && go run ./tools/check-templates.go -ext=.g && go run ./tools/check-templates.go -ext=.e)`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `./scripts/aofei-cache-smoke.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Create the M30 status file. | `[+]` | |
| Identify SSP traffic separately from ADX audits. | `[+]` | |
| Add best-effort `/pz` browser cookie handling. | `[+]` | |
| Verify SSP markup trackers feed measurement. | `[+]` | |
| Add direct SSP smoke fixtures. | `[+]` | |
| Update SSP measurement docs and memory bank files. | `[+]` | |
| Run required verification and record results. | `[+]` | |
| Operators can distinguish ADX and direct SSP traffic in logs/audits. | `[+]` | |
| Existing ledger aggregation continues without schema change. | `[+]` | |
| Browser cookie absence does not break serving. | `[+]` | |
| `GOWORK=off go test ./dsp ./match ./internal/jobs/ledger ./internal/jobs/cache -count=1` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./cmd/unify ./summer/slot -count=1)` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./...)` | `[+]` | |
| `(cd ../pzdesign && go run ./tools/check-templates.go -ext=.g && go run ./tools/check-templates.go -ext=.e)` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `./scripts/aofei-cache-smoke.sh` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |

````
