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
