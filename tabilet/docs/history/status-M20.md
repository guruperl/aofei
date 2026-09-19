# Retired milestone M20 - Middleman Bidder Runtime

**Milestone.** M20
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M20.md
**Source specification.** memory-bank/milestone.md#m20---middleman-bidder-runtime
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M20 - Middleman Bidder Runtime `[+]`

Wire approved bidders into fallback runtime after local campaign matching
returns no bid.

Scope:

- Build the route/bidder cache from `adv_bidder` plus `mid_route_*`.
- Require active route group, active route bidder, active credential-ready
  bidder, valid synthetic chain, route match, and synthetic item ACL/channel
  eligibility before fanout.
- Fan out downstream OpenRTB requests within the minimum of request `tmax`,
  route timeout, and DSP config timeout.
- Discard late, invalid, inactive, and non-USD responses.
- Aggregate surviving responses by price, apply route/bidder margin, and return
  the best upstream bid only after local no-bid fallback.

Acceptance:

- Approved bidders are considered only through configured routes and existing
  ACL/channel eligibility.
- Local campaign bids still win before any fallback fanout.
- No callback proxying, win/loss reconciliation, or middleman reporting changes
  are introduced until later milestones.
````

## Status record

````markdown
# Status M20 - Middleman Bidder Runtime

Current note: S01 supersedes the historical forwarding shape below. Runtime
now requires the independent privacy disclosure gate and sends each bidder only
its assigned impressions in a newly built contextual, extension-scrubbed view.

## Goal

Wire approved advertiser-owned bidders into DSP fallback runtime after local
campaign matching cannot fill an impression.

## Completed

- `[+]` Redis route/bidder cache.
  - Added a versioned `middleman:routes` Redis payload built by the singleton
    `cmd/redis-cache` job from active `adv_bidder` and `mid_route_*` rows.
  - The cache includes synthetic item ACL payloads and validates the synthetic
    advertiser/campaign/item/creative chain through SQL joins.

- `[+]` Per-impression fallback runtime.
  - Local campaign bids still win first.
  - The full original request is forwarded to eligible downstream bidders, but
    only local no-bid impressions can be accepted from downstream responses.
  - Candidate pooling merges all matching active routes, filters by synthetic
    ACL/channel rules, dedupes bidders, and applies
    `middleman_max_bidders_per_imp`.

- `[+]` Downstream OpenRTB client.
  - Forwarded requests preserve original request fields and the full impression
    list, then override `ext.request_domain`.
  - `credential_ref` resolves to an environment variable containing JSON
    outbound headers.
  - Fanout uses the minimum of remaining request `tmax`, route/group/bidder
    timeout, and DSP config timeout.

- `[+]` Response normalization.
  - Late, invalid, non-USD, below-floor, or wrong-impression responses are
    discarded.
  - Surviving bids are marked up by route/bidder margin and returned with
    synthetic reporting IDs while preserving downstream markup and tracking.

## Carry Forward

- `[+]` Callback proxying and downstream win/loss reconciliation were forwarded
  to M21 and completed there.
- `[+]` Middleman advertiser/operator reporting was forwarded through M21 and
  completed in M22.
- `[+]` Optional spread/local route-snapshot ownership is consolidated in D03;
  the runtime remains Redis-only until D03 records its decision.

## Verification

- `[+]` `GOWORK=off go test ./match ./internal/jobs/cache ./dsp ./cmd/redis-cache ./cmd/unify`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache ./cmd/ledger ./internal/jobs/cache ./internal/jobs/ledger`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Redis route/bidder cache. | `[+]` | |
| Per-impression fallback runtime. | `[+]` | |
| Downstream OpenRTB client. | `[+]` | |
| Response normalization. | `[+]` | |
| Callback proxying and downstream win/loss reconciliation were forwarded | `[+]` | |
| Middleman advertiser/operator reporting was forwarded through M21 and | `[+]` | |
| Optional spread/local route-snapshot ownership is consolidated in D03; | `[+]` | |
| `GOWORK=off go test ./match ./internal/jobs/cache ./dsp ./cmd/redis-cache ./cmd/unify` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache ./cmd/ledger ./internal/jobs/cache ./internal/jobs/ledger` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |

````
