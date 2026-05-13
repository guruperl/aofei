# Status M29 - Publisher Tag UI And Download

## Goal

Make direct publisher tags usable from the existing `pub` UI by generating
working M28 `/pz` requests, exposing copy/download actions, preserving slot
sizes, and allowing external browser embeds to call `/pz`.

## Tasks

- `[+]` Create the M29 status file.

- `[+]` Persist publisher slot size.
  - Add `pub_slot.size_id` to the SQL baseline.
  - Include `size_id` in Summer slot insert, update, topics, and edit metadata.
  - Use create/edit width and height fields to persist the packed size.

- `[+]` Generate M28-compatible publisher tag data.
  - Use absolute Aofei `/pz` URLs from configured `ServerURL`.
  - Pack `site` and `slot` direct tokens.
  - Use DOM-only ad unit `code` values.
  - Emit supported banner media types and JSON array response examples.

- `[+]` Update `www/js/ads.js` for external embeds.
  - Default the endpoint from the loaded script origin.
  - Accept explicit endpoint override options.
  - Preserve ad-unit order and omit credentials by default.

- `[+]` Add minimal `/pz` CORS in `../pzdesign/cmd/unify`.
  - Handle `OPTIONS /pz`.
  - Add permissive CORS headers for `/pz` only.

- `[+]` Add publisher copy and download actions in `.g` and `.e` slot topics
  templates.

- `[+]` Add focused tests for slot filtering/tag output and `/pz` CORS.

- `[+]` Update SSP docs, database docs, and memory bank files.

- `[+]` Run required verification and record results.
  - Checked `evolution/`; no new version is needed because M29 completes the
    already recorded direct SSP publisher-tag direction without changing the
    product or architecture boundary.

## Acceptance

- `[+]` Publishers can copy or download a working browser tag for each slot.
- `[+]` External sites can embed the sample tag and call Aofei `/pz`.
- `[+]` Existing publisher admin tests still pass.

## Verification

- `[+]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load`
- `[+]` `./scripts/aofei-local.sh check-sql`
- `[+]` `./scripts/aofei-local.sh diff-schema`
- `[+]` `GOWORK=off go test ./acl ./dsp ./internal/jobs/cache`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `(cd ../pzdesign && GOWORK=off go test ./cmd/unify ./summer/slot)`
- `[+]` `(cd ../pzdesign && go run ./tools/check-templates.go -ext=.g && go run ./tools/check-templates.go -ext=.e)`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `./scripts/aofei-cache-smoke.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`
