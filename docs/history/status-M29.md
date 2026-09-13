# Retired milestone M29 - Publisher Tag UI And Download

**Milestone.** M29
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M29.md
**Source specification.** memory-bank/milestone.md#m29---publisher-tag-ui-and-download
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M29 - Publisher Tag UI And Download `[+]`

Make direct publisher tags usable from the existing `pub` UI.

Scope:

- Fix publisher slot snippets to use an absolute Aofei endpoint.
- Update `www/js/ads.js` to derive the ad-server origin from the script URL or
  an explicit endpoint option.
- Add visible tag download/link actions on publisher slot pages.
- Align browser and API examples with the M28 request contract.
- Keep the existing `pub` role and site/slot CRUD.

Acceptance:

- Publishers can copy or download a working browser tag for each slot.
- External sites can embed the sample tag and call Aofei `/pz`.
- Existing publisher admin tests still pass.

Result:

- `pub_slot.size_id` is stored in the SQL baseline and Summer slot metadata, so
  generated direct tags use each slot's configured width/height.
- Publisher slot topics generate M28-compatible browser/API samples with
  absolute `/pz` endpoints, direct `site` and `slot` tokens, DOM-only ad-unit
  codes, and banner `mediaTypes`.
- `www/js/ads.js` defaults to the origin of the loaded script plus `/pz`,
  supports explicit endpoint overrides, preserves ad-unit order, and omits
  credentials by default.
- Publisher slot topics expose copy actions and downloadable
  `aofei-slot-<slot_id>.html` browser samples.
- `cmd/unify` handles `OPTIONS /pz` and applies permissive CORS headers only to
  `/pz`.
````

## Status record

````markdown
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

- `[+]` Resolve M27-M29 deep-review publisher sample finding.
  - Publisher API snippets no longer show body-level `ua` or `ip` fields,
    because M28 derives request metadata from HTTP headers.

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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Create the M29 status file. | `[+]` | |
| Persist publisher slot size. | `[+]` | |
| Generate M28-compatible publisher tag data. | `[+]` | |
| Update `www/js/ads.js` for external embeds. | `[+]` | |
| Add minimal `/pz` CORS in `../pzdesign/cmd/unify`. | `[+]` | |
| Add publisher copy and download actions in `.g` and `.e` slot topics | `[+]` | |
| Add focused tests for slot filtering/tag output and `/pz` CORS. | `[+]` | |
| Resolve M27-M29 deep-review publisher sample finding. | `[+]` | |
| Update SSP docs, database docs, and memory bank files. | `[+]` | |
| Run required verification and record results. | `[+]` | |
| Publishers can copy or download a working browser tag for each slot. | `[+]` | |
| External sites can embed the sample tag and call Aofei `/pz`. | `[+]` | |
| Existing publisher admin tests still pass. | `[+]` | |
| `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load` | `[+]` | |
| `./scripts/aofei-local.sh check-sql` | `[+]` | |
| `./scripts/aofei-local.sh diff-schema` | `[+]` | |
| `GOWORK=off go test ./acl ./dsp ./internal/jobs/cache` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./cmd/unify ./summer/slot)` | `[+]` | |
| `(cd ../pzdesign && go run ./tools/check-templates.go -ext=.g && go run ./tools/check-templates.go -ext=.e)` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `./scripts/aofei-cache-smoke.sh` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |

````
