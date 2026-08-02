# Status M16 - Middleman AdX Advertiser-Owned Bidder Schema

## Goal

Create the advertiser-owned bidder endpoint, route, and synthetic reporting
schema foundation for middleman AdX fallback without changing bid serving
behavior yet.

## Completed

- `[+]` Advertiser-owned endpoint metadata.
  - Added `adv_bidder` for OpenRTB endpoint metadata owned by `adv`.
  - Advertiser users can manage endpoint metadata through the existing `adv`
    role; admins control credential refs, synthetic reporting IDs, credential
    status, and activation.
  - Registered the Summer `bidder` component module.

- `[+]` Synthetic reporting row contract.
  - `adv_bidder` can point at synthetic campaign, item, and creative rows.
  - Existing advertiser ledger and daily reporting can later roll up middleman
    spend through `creative_id -> item_id -> campaign_id -> adv_id`.
  - The synthetic campaign/item chain is also the planned bidder eligibility
    surface, reusing existing ACL and channel matching instead of a separate
    bidder/site allowlist.

- `[+]` Route and reporting schema.
  - Added `mid_route_group`, `mid_route_bidder`, and `mid_route_target` for
    future fallback route assignment.

- `[+]` Docs and memory bank.
  - Added `docs/middleman-adx.md`.
  - Updated product, architecture, tech-stack, database, DSP workflow,
    Summer UI, milestone, and evolution docs.

## Carry Forward

- `[+]` Build the Summer/Genelet bidder portal and admin approval backend in
  M17.
- `[+]` Complete Summer template modernization in M18.
- `[+]` Route cache, synthetic-item eligibility, downstream OpenRTB fanout, and
  fallback auction behavior were completed in M20.
- `[+]` Callback proxying, audit, operations, and advertiser reporting were
  completed in M21/M22.

## Verification

- `[+]` `GOWORK=off go test ./summer ./summer/bidder ./summer/registry ./genelet`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `./scripts/aofei-local.sh check-sql`
- `[+]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load && ./scripts/aofei-local.sh diff-schema`
- `[+]` `GOWORK=off staticcheck ./summer ./summer/registry ./summer/bidder`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

Focused staticcheck for changed Summer packages passes. A broader exploratory
`GOWORK=off staticcheck ./summer ./summer/registry ./genelet` still reports
pre-existing Genelet findings and is not used as the M16 gate.
