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
- `[ ]` Build route/bidder cache, synthetic item eligibility checks, downstream
  OpenRTB fanout, and fallback auction integration in M20.
- `[ ]` Add callback proxying, win/loss reconciliation, middleman reporting,
  and operations after runtime fanout is stable.

## Verification

- `[+]` `GOWORK=off go test ./summer/bidder ./summer/registry ./genelet ./cmd/unify`
- `[+]` `GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer/bidder ./summer`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./summer ./summer/registry ./summer/bidder ./cmd/unify`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
