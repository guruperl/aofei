# Retired milestone M28 - SSP Runtime Adapter

**Milestone.** M28
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M28.md
**Source specification.** memory-bank/milestone.md#m28---ssp-runtime-adapter
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M28 - SSP Runtime Adapter `[+]`

Serve direct browser ad-tag requests through the existing Aofei bid engine.

Scope:

- Add `dsp.Controller.ServeSSP` and wire `POST /pz` in `../pzdesign/cmd/unify`.
- Convert valid SSP ad units into internal OpenRTB impressions.
- Reuse the existing local bid flow for candidates, caps, audiences, creative
  rendering, trackers, and audit publishing.
- Return a JSON HTML-string array in input order.
- Add SSP request, malformed, fill, no-fill, and validation expvar counters.

Acceptance:

- Multi-ad-unit SSP requests return renderable HTML in request order.
- Direct SSP impressions and clicks carry real publisher, site, slot, and size
  IDs.
- `/pz` does not write MySQL or refresh caches.

Result:

- `dsp.Controller.ServeSSP` reads the v1 browser JSON body with the existing bid
  body limit, validates direct tokens through local/Redis `pubmap:by-id`, and
  returns a JSON HTML-string array in input order.
- The adapter synthesizes OpenRTB browser metadata from request headers and
  cache-derived site/slot strings, then reuses the existing local candidate,
  cap, audience, creative rendering, tracker, and audit paths.
- `../pzdesign/cmd/unify` registers `POST /pz` before the Genelet catch-all.
- M28 keeps middleman fallback, cookies, CORS/origin policy changes, publisher
  tag UI, and reporting-semantics separation out of scope.
````

## Status record

````markdown
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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Create the M28 status file. | `[+]` | |
| Add SSP runtime conversion and handler in `dsp`. | `[+]` | |
| Wire `POST /pz` in `../pzdesign/cmd/unify`. | `[+]` | |
| Add focused Aofei and pzdesign tests. | `[+]` | |
| Update SSP docs and memory-bank files. | `[+]` | |
| Resolve M27-M29 deep-review runtime findings. | `[+]` | |
| Run required verification and record results. | `[+]` | |
| Multi-ad-unit SSP requests return renderable HTML in request order. | `[+]` | |
| Direct SSP impressions and clicks carry real publisher, site, slot, and | `[+]` | |
| `/pz` does not write MySQL or refresh caches. | `[+]` | |
| `GOWORK=off go test ./acl ./dsp ./internal/jobs/cache` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `(cd ../pzdesign && GOWORK=off go test ./cmd/unify)` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `./scripts/aofei-cache-smoke.sh` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |

````
