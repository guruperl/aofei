# Retired milestone M31 - SSP Hardening And Product Boundary

**Milestone.** M31
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M31.md
**Source specification.** memory-bank/milestone.md#m31---ssp-hardening-and-product-boundary
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M31 - SSP Hardening And Product Boundary `[+]`

Finish safety, operations, and product separation after the basic path works.

Scope:

- Add origin/referrer policy controls for browser tags.
- Validate token tampering, inactive inventory, mismatched size, and unsupported
  media-type cases.
- Decide whether publisher inventory needs an explicit supply-source field.
- Keep browser HTML arrays as stable v1 while planning optional API/mobile
  response formats.
- Update production runbook, local runtime docs, memory bank status, and
  closeout verification.

Acceptance:

- Direct SSP is safe to expose outside local development.
- ADX `/bid/{domain}` and direct `/pz` are documented as separate traffic
  entrypoints.
- Remaining advanced API/mobile/native response work is carried forward
  explicitly.

Result:

- `/pz` now enforces exact cached-site-host `Origin`/`Referer` policy after
  token/cache validation and before cookies, bidding, or audit publishing.
  Browser traffic must send a matching `Origin` or `Referer`; `platform:"sdk"`
  may omit both headers, but any supplied header must still match.
- Policy rejections return `403` and increment
  `aofei_ssp_policy_rejections_total`.
- Direct SSP remains bounded by the `/pz` entrypoint plus audit
  `source:"ssp"`/`contract:"pz-v1"` metadata; no supply-source schema/cache
  field, `ads.js` change, credentialed CORS, or response-format change was
  added.
- API/mobile/native response formats and richer supply taxonomy remain future
  product work.
````

## Status record

````markdown
# Status M31 - SSP Hardening And Product Boundary

## Goal

Harden direct SSP browser serving and close the M31 product-boundary decisions
without changing schema, cache shape, `ads.js`, or the v1 HTML-array response.

## Tasks

- `[+]` Create the M31 status file.

- `[+]` Enforce `/pz` browser origin/referrer policy.
  - Browser traffic is any request that is not `platform:"sdk"`.
  - Browser traffic must send `Origin` or `Referer` matching the validated cached
    site host exactly.
  - Any present `Origin` or `Referer`, including SDK requests, must parse as an
    absolute URL and match the cached site host.
  - `Origin: null`, malformed URLs, missing browser headers, mismatched hosts,
    and subdomain variants return `403`.
  - Policy rejections do not set `aofei_pz_uid`, bid, or publish audits.
  - `aofei_ssp_policy_rejections_total` tracks policy rejections.

- `[+]` Close remaining validation coverage.
  - Direct SSP tests cover matching `Origin`, matching `Referer`, mismatched
    headers, `Origin:null`, malformed headers, missing browser headers,
    SDK-without-headers success, and policy rejection side effects.
  - Existing malformed, invalid-token, inactive/unknown publisher,
    site/slot/size mismatch, missing media, and unsupported media cases remain
    `400`.
  - ACL coverage proves inactive and limited publishers are absent from the
    direct by-id lookup.

- `[+]` Decide direct SSP product boundary.
  - Do not add a publisher supply-source database or cache field in M31.
  - `/pz` plus audit `source:"ssp"`/`contract:"pz-v1"` remains the current
    direct SSP source boundary.
  - Richer supply taxonomy and API/mobile/native response formats remain future
    product work.

- `[+]` Keep v1 serving contracts stable.
  - `ads.js` remains unchanged.
  - `/pz` CORS remains permissive and credentialless.
  - Browser and SDK/API responses remain JSON arrays of HTML strings.
  - SDK/in-app `platform:"sdk"` requests remain credentialless and cookie-free;
    with the current request contract their identity uses existing device-ID or
    UA+IP fallback rather than `aofei_pz_uid`.

- `[+]` Update docs and memory bank files.

- `[+]` Run required verification and record results.

- `[+]` Check `evolution/`.
  - No new version is needed because M31 implements the planned hardening and
    boundary documentation without changing product direction, schema, cache
    contracts, or response format.

## Acceptance

- `[+]` Direct SSP browser traffic is rejected unless the validated page host
  matches the cached site host.
- `[+]` ADX `/bid/{domain}` and direct `/pz` remain documented as separate
  traffic entrypoints.
- `[+]` Advanced API/mobile/native response work is carried forward explicitly.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./acl ./match ./internal/jobs/cache -count=1`
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
| Create the M31 status file. | `[+]` | |
| Enforce `/pz` browser origin/referrer policy. | `[+]` | |
| Close remaining validation coverage. | `[+]` | |
| Decide direct SSP product boundary. | `[+]` | |
| Keep v1 serving contracts stable. | `[+]` | |
| Update docs and memory bank files. | `[+]` | |
| Run required verification and record results. | `[+]` | |
| Check `evolution/`. | `[+]` | |
| Direct SSP browser traffic is rejected unless the validated page host | `[+]` | |
| ADX `/bid/{domain}` and direct `/pz` remain documented as separate | `[+]` | |
| Advanced API/mobile/native response work is carried forward explicitly. | `[+]` | |
| `GOWORK=off go test ./dsp ./acl ./match ./internal/jobs/cache -count=1` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./cmd/unify ./summer/slot -count=1)` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./...)` | `[+]` | |
| `(cd ../pzdesign && go run ./tools/check-templates.go -ext=.g && go run ./tools/check-templates.go -ext=.e)` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `./scripts/aofei-cache-smoke.sh` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |

````
