# Status M28 - SSP Runtime Adapter

## Goal

Serve direct browser ad-tag requests through `/pz` by validating packed direct
SSP tokens, converting ad units into internal OpenRTB impressions, and reusing
the existing local Aofei bid path.

## Tasks

- `[+]` Create the M28 status file.

- `[+]` Add SSP runtime conversion and handler in `dsp`.
  - Read request bodies with the existing bid body limit.
  - Return HTTP errors for malformed or invalid requests.
  - Return `200 application/json` arrays for valid requests, preserving input
    ad-unit order and using `""` for no-fill units.
  - Reuse local candidate, cap, audience, creative rendering, tracker, and audit
    paths without middleman fallback.

- `[+]` Wire `POST /pz` in `../pzdesign/cmd/unify`.

- `[+]` Add focused Aofei and pzdesign tests.
  - Aofei covers media conversion with token size, request-order HTML arrays,
    partial fill, validation failures, and header-derived metadata.
  - pzdesign covers `/pz`, `/bid/{domain}`, and `/debug/vars` route handling
    before the Genelet catch-all.

- `[+]` Update SSP docs and memory-bank files.
  - Updated `docs/ssp-direct-traffic.md`, product, architecture, tech stack, and
    milestone state.

- `[+]` Resolve M27-M29 deep-review runtime findings.
  - `/pz` now validates the full site/slot/size tuple against cache metadata.
  - `cmd/unify -local` no longer overrides config `is_local` when the flag is
    omitted, and enabling it explicitly reloads local static snapshots.

- `[+]` Run required verification and record results.

## Acceptance

- `[+]` Multi-ad-unit SSP requests return renderable HTML in request order.
- `[+]` Direct SSP impressions and clicks carry real publisher, site, slot, and
  size IDs.
- `[+]` `/pz` does not write MySQL or refresh caches.

## Verification

- `[+]` `GOWORK=off go test ./acl ./dsp ./internal/jobs/cache`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `(cd ../pzdesign && GOWORK=off go test ./cmd/unify)`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `./scripts/aofei-cache-smoke.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`
