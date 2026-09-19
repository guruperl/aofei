# Retired milestone M16 - Middleman AdX Advertiser-Owned Bidder Schema

**Milestone.** M16
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M16.md
**Source specification.** memory-bank/milestone.md#m16---middleman-adx-advertiser-owned-bidder-schema
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M16 - Middleman AdX Advertiser-Owned Bidder Schema `[+]`

Establish the advertiser-owned endpoint, routing, and synthetic reporting schema
needed before fallback fanout changes the bid path.

Scope:

- Add downstream endpoint metadata under `adv_bidder`, owned by `adv`.
- Link each bidder endpoint to optional synthetic campaign, item, and creative
  IDs so existing advertiser ledger/report joins can be reused.
- Add `mid_route_*` tables for future fallback route groups and inventory
  assignment.
- Document that runtime fanout is still disabled after this milestone.

Acceptance:

- The active schema baseline recreates empty `adv_bidder` and `mid_route_*`
  middleman tables.
- Summer registry includes the advertiser-owned bidder endpoint module.
- Docs and memory bank describe the advertiser-owned endpoint/reporting boundary
  ACL/channel eligibility reuse, and future milestone sequence.
````

## Status record

````markdown
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

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Advertiser-owned endpoint metadata. | `[+]` | |
| Synthetic reporting row contract. | `[+]` | |
| Route and reporting schema. | `[+]` | |
| Docs and memory bank. | `[+]` | |
| Build the Summer/Genelet bidder portal and admin approval backend in | `[+]` | |
| Complete Summer template modernization in M18. | `[+]` | |
| Route cache, synthetic-item eligibility, downstream OpenRTB fanout, and | `[+]` | |
| Callback proxying, audit, operations, and advertiser reporting were | `[+]` | |
| `GOWORK=off go test ./summer ./summer/bidder ./summer/registry ./genelet` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `./scripts/aofei-local.sh check-sql` | `[+]` | |
| `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load && ./scripts/aofei-local.sh diff-schema` | `[+]` | |
| `GOWORK=off staticcheck ./summer ./summer/registry ./summer/bidder` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |

````
