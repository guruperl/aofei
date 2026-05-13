# Status M20 - Middleman Bidder Runtime

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
- `[ ]` Middleman advertiser/operator reporting was forwarded through M21 and
  remains M22.
- `[ ]` Spread/local snapshots for bidder routes remain deferred; M20 route cache
  is Redis-only.

## Verification

- `[+]` `GOWORK=off go test ./match ./internal/jobs/cache ./dsp ./cmd/redis-cache ./cmd/unify`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache ./cmd/ledger ./internal/jobs/cache ./internal/jobs/ledger`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
