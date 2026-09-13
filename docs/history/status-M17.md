# Retired milestone M17 - Advertiser Bidder Portal

**Milestone.** M17
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M17.md
**Source specification.** memory-bank/milestone.md#m17---advertiser-bidder-portal
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M17 - Advertiser Bidder Portal `[+]`

Make `adv_bidder` usable from Summer/Genelet without enabling DSP runtime
fanout.

Scope:

- Expose advertiser HTML routes for bidder list, new, insert, edit, and update.
- Expose admin HTML routes for bidder list, edit, update, and approval.
- Keep JSON routes under `/goto/{role}/json/bidder`.
- Restrict advertiser writes to safe endpoint metadata and expose credential
  status plus active status as read-only.
- Validate bidder endpoint URLs and timeout values before writes.
- Let admin approval create or validate inactive synthetic reporting rows,
  store a credential ref, and mark the bidder active.
- Move active Summer templates to the sibling `../pzdesign/tmpls` tree and point
  generated local Summer config away from ignored `.local/templates`.

Acceptance:

- Advertisers cannot set credential refs, credential state, activation, or
  synthetic IDs.
- Admin approval requires `bidder_id` and `credential_ref`, runs in one DB
  transaction, creates or validates the synthetic campaign/item/creative chain,
  and rejects partial or wrong-advertiser synthetic state.
- DSP bid serving, route cache, downstream OpenRTB fanout, auctions, and
  callback proxying remain unchanged.
````

## Status record

````markdown
# Status M17 - Advertiser Bidder Portal

## Goal

Make advertiser-owned bidder endpoints usable from Summer/Genelet while keeping
DSP runtime fanout disabled.

## Completed

- `[+]` Bidder portal routes and templates.
  - Added advertiser/admin bidder list/new/edit/approve pages to the active
    Summer UI template tree.
  - Advertiser HTML routes cover
    `/goto/adv/g/bidder?action=topics|startnew|insert|edit|update`.
  - Admin HTML routes cover
    `/goto/admin/g/bidder?action=topics|edit|update|approve`.
  - JSON routes remain available under `/goto/{role}/json/bidder`.

- `[+]` Advertiser-safe endpoint writes.
  - Advertiser ownership now scopes bidder queries and edits by `adv_id`.
  - Advertiser writes are limited to `bidder_name`, `endpoint_url`,
    `openrtb_version`, `seat`, and `timeout_ms`.
  - Advertiser-provided credential refs, credential state, activation, and
    synthetic reporting IDs are stripped before writes.
  - Advertisers can see read-only `credential_status` and `active`.
  - Endpoint URLs must be absolute `http` or `https` URLs without userinfo.
  - Timeouts default to `100` ms and must be positive, bounded millisecond
    values.

- `[+]` Admin approval backend.
  - Added `summer/bidder.Model.Approve`.
  - Approval requires `bidder_id` and `credential_ref`.
  - Approval runs in one DB transaction, creates a missing inactive synthetic
    campaign/item/creative chain, validates existing complete same-advertiser
    chains, rejects partial or wrong-advertiser chains, sets
    `credential_status='Active'`, and activates the bidder.

- `[+]` Template path boundary.
  - Summer config examples and generated local configs no longer point
    `Template` at ignored `.local/templates`.

## Carry Forward

- `[+]` Move active templates to the sibling `pzdesign` tree and switch Genelet
  HTML rendering to `html/template` in M18.
- `[+]` Route/bidder cache, synthetic item eligibility, downstream OpenRTB
  fanout, and fallback auction integration were completed in M20.
- `[+]` Callback proxying, win/loss reconciliation, middleman reporting, and
  operations were completed in M21-M24.

## Verification

- `[+]` `GOWORK=off go test ./summer/bidder ./summer/registry ./genelet ./cmd/unify`
- `[+]` `GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer/bidder ./summer`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./summer ./summer/registry ./summer/bidder ./cmd/unify`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Bidder portal routes and templates. | `[+]` | |
| Advertiser-safe endpoint writes. | `[+]` | |
| Admin approval backend. | `[+]` | |
| Template path boundary. | `[+]` | |
| Move active templates to the sibling `pzdesign` tree and switch Genelet | `[+]` | |
| Route/bidder cache, synthetic item eligibility, downstream OpenRTB | `[+]` | |
| Callback proxying, win/loss reconciliation, middleman reporting, and | `[+]` | |
| `GOWORK=off go test ./summer/bidder ./summer/registry ./genelet ./cmd/unify` | `[+]` | |
| `GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer/bidder ./summer` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./summer ./summer/registry ./summer/bidder ./cmd/unify` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |

````
