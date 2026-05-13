# Result V7

Resulting direction after M20:

- Middleman fallback is an explicit DSP runtime feature gated by
  `middleman_enabled`.
- Local campaign bids win first; middleman fanout is per local no-bid
  impression.
- `cmd/redis-cache` compiles the Redis-only `middleman:routes` payload from
  active `adv_bidder` and `mid_route_*` rows. `cmd/unify` does not refresh this
  cache.
- Bidder credentials remain outside MySQL and Redis. `credential_ref` names an
  environment variable containing a JSON outbound header map.
- Forwarded requests preserve original OpenRTB fields and the full impression
  list, then override `ext.request_domain`; only locally unfilled impressions
  can win from downstream responses.
- Upstream middleman bids preserve downstream markup/tracking fields but use the
  approved synthetic campaign and creative identifiers.
- Callback proxying, downstream win/loss reconciliation, and middleman
  reporting remain later milestones.
