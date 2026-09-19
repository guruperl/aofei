# Retired milestone M32 - SSP Mobile/API Contract And Response Formats

**Milestone.** M32
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M32.md
**Source specification.** memory-bank/milestone.md#m32---ssp-mobileapi-contract-and-response-formats
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M32 - SSP Mobile/API Contract And Response Formats `[+]`

Add mobile/API serving to the existing `/pz` endpoint without changing browser
defaults or account/schema/cache boundaries.

Scope:

- Accept explicit `responseFormat` values: omitted/`"html"`, `"json"`, and
  `"openrtb"`.
- For `platform:"sdk"`, accept OpenRTB-like body `app`, `device`, and `user`
  fields.
- Synthesize app traffic from the validated direct SSP cache while keeping
  `site` and `adUnits[].slot` tokens authoritative.
- Keep `ads.js`, schema, cache shape, CORS credentials, and account roles out of
  scope.

Acceptance:

- Existing browser `/pz` HTML-array behavior remains compatible.
- SDK requests synthesize `BidRequest.App`, do not use cookies, and honor body
  device/user identity.
- App identity mismatches against the validated site token return `400`.
- JSON responses return ordered fill/no-fill objects with native JSON where
  applicable.
- OpenRTB responses return a valid `BidResponse`, including `200` with empty
  `seatbid` on all-no-fill.
````

## Status record

````markdown
# Status M32 - SSP Mobile/API Contract And Response Formats

State: `[+]` Completed

Add mobile/API serving to the existing direct SSP `/pz` endpoint while
preserving browser defaults, `ads.js`, the `pub` account boundary, schema, and
cache shape.

## Tasks

- `[+]` Add parser/runtime support for `responseFormat`, SDK `app`, `device`,
  and `user`.
  - Omitted or `"html"` preserves the existing ordered HTML-string array.
  - `platform:"sdk"` synthesizes `BidRequest.App`, leaves `BidRequest.Site` nil,
    rejects mismatched supplied app id/bundle/domain, merges body device fields
    with header IP/UA fallback, and honors body user IDs.
  - Browser traffic keeps browser-only `aofei_pz_uid` behavior and ignores SDK
    body identity fields.
- `[+]` Add JSON and OpenRTB response renderers from local SSP winners.
  - JSON responses return ordered fill/no-fill objects with markup, native JSON
    when applicable, tracker URLs, price/currency, ids, and dimensions.
  - OpenRTB responses return a standard `BidResponse`; all-no-fill returns
    `200` with an empty `seatbid`.
- `[+]` Update publisher API samples and direct SSP docs.
  - Publisher slot API snippets now show `responseFormat:"json"`, SDK app,
    device, and user objects, JSON fill metadata, and mention
    `responseFormat:"openrtb"`.
- `[+]` Check evolution history.
  - Added `evolution/result-v16.md` because M32 changes the public `/pz`
    response contract direction.
- `[+]` Run closeout verification.
- `[+]` Mark milestone complete after verification.

## Acceptance

- `[+]` Existing browser `/pz` HTML behavior remains byte-shape compatible.
- `[+]` SDK request with `app`/`device`/`user` produces `BidRequest.App`, no
  `Site`, explicit device/user identity, and no cookie.
- `[+]` App identity mismatch against validated site token returns `400`.
- `[+]` `responseFormat:"json"` returns ordered fill/no-fill objects and parsed
  native JSON.
- `[+]` `responseFormat:"openrtb"` returns a valid `BidResponse` with expected
  `impid`, `adm`, ids, and empty `seatbid` on all-no-fill.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./acl ./match ./internal/jobs/ledger ./internal/jobs/cache -count=1`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `(cd ../pzdesign && GOWORK=off go test ./summer/slot -count=1)`
- `[+]` `(cd ../pzdesign && GOWORK=off go test ./...)`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`

## Notes

- No schema drift checks are required for M32 because this milestone does not
  intentionally change schema.
- M33 remains the milestone for SSP middleman fallback.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Add parser/runtime support for `responseFormat`, SDK `app`, `device`, | `[+]` | |
| Add JSON and OpenRTB response renderers from local SSP winners. | `[+]` | |
| Update publisher API samples and direct SSP docs. | `[+]` | |
| Check evolution history. | `[+]` | |
| Run closeout verification. | `[+]` | |
| Mark milestone complete after verification. | `[+]` | |
| Existing browser `/pz` HTML behavior remains byte-shape compatible. | `[+]` | |
| SDK request with `app`/`device`/`user` produces `BidRequest.App`, no | `[+]` | |
| App identity mismatch against validated site token returns `400`. | `[+]` | |
| `responseFormat:"json"` returns ordered fill/no-fill objects and parsed | `[+]` | |
| `responseFormat:"openrtb"` returns a valid `BidResponse` with expected | `[+]` | |
| `GOWORK=off go test ./dsp ./acl ./match ./internal/jobs/ledger ./internal/jobs/cache -count=1` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./summer/slot -count=1)` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./...)` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |

````
