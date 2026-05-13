# Middleman AdX

Middleman AdX fallback is the planned path for bid requests that do not match a
local campaign. M16 established the advertiser-owned bidder endpoint and route
schema, and M17 adds the Summer/Genelet portal plus admin approval for endpoint
metadata and inactive synthetic reporting rows. Bid serving still returns
`204 No Content` when no local campaign produces a bid.

## Account Boundary

Downstream bidder endpoints are owned by advertisers (`adv`). A downstream
partner uses the existing advertiser login and account tooling instead of a
separate Summer/Genelet role.

Advertiser users can manage safe `adv_bidder` endpoint metadata and, in a later
milestone, view middleman results through advertiser reports. Their writable
fields are bidder name, endpoint URL, OpenRTB version, seat, and timeout.
Advertisers can see credential status and active status, but cannot set
credential refs, synthetic reporting IDs, or activation. Operators manage
credential references, endpoint activation, route groups, inventory assignment,
and margin settings. Secrets are not stored in MySQL;
`adv_bidder.credential_ref` points to environment-managed secret material.

Each `adv_bidder` may reference synthetic campaign, item, and creative rows.
Those rows are reporting identities, not normal local demand. They let existing
ledger joins preserve charge-side delivery through `adv_campaign`, `adv_item`,
`adv_creative`, `ledger_adv`, and `daily_adv`. M22 adds middleman-specific
ledger tables for pay-side advertiser reporting and admin settlement views.

Operator tooling must ensure the synthetic IDs form one chain owned by the same
advertiser: `adv_bidder.adv_id -> adv_campaign.adv_id`,
`adv_item.campaign_id -> adv_campaign.campaign_id`, and
`adv_creative.item_id -> adv_item.item_id`.

M17 approval creates the chain when all three synthetic IDs are empty, reuses an
existing complete chain after same-advertiser validation, and rejects partial
synthetic state. Created synthetic campaign, item, and creative rows are
inactive so they cannot become local demand.

The same synthetic chain is also the planned inventory eligibility surface.
Existing advertiser/campaign/item access-control and channel tables already
express whether demand may serve a publisher or site:

- `adv.access_order` plus `ac(entitytype_id=4, othertype_id=3|31)` controls
  advertiser-level publisher/site allow or block behavior when campaign and
  item inherit.
- `adv_campaign.access_order` plus
  `ac(entitytype_id=41, othertype_id=3|31)` controls campaign-level
  publisher/site allow or block behavior when item inherits.
- `adv_item.access_order`, `adv_item.fl_sitetypes`, and
  `ac(entitytype_id=42, othertype_id=31)` control item-level site/app
  eligibility.
- `ch_belong(entitytype_id=41)` and `ch_ac(entitytype_id=42)` keep the current
  campaign category and item channel allow/block model.

Because of this, M18+ should not add a separate bidder-vs-site ACL table unless
the existing model proves insufficient. Route groups should select the coarse
fanout pool for a publisher/site/slot, then runtime eligibility should filter
each `adv_bidder` through its synthetic item audience rules before the request
is forwarded downstream.

## Summer Portal

Tracked Summer UI templates live in the sibling `../pzdesign/tmpls` tree. The
HTML routes are:

- Advertiser: `/goto/adv/g/bidder?action=topics|startnew|insert|edit|update`
- Admin: `/goto/admin/g/bidder?action=topics|edit|update|approve`
- Admin routes:
  `/goto/admin/g/midroute?action=topics|startnew|insert|edit|update|delete`
- Admin route bidders:
  `/goto/admin/g/midroute?action=bidders|startnewBidder|insertBidder|editBidder|updateBidder|deleteBidder`
- Admin route targets:
  `/goto/admin/g/midroute?action=targets|startnewTarget|insertTarget|editTarget|updateTarget|deleteTarget`

JSON routes remain available under `/goto/{role}/json/bidder` and
`/goto/admin/json/midroute`.

## Schema

The middleman tables are:

| Table | Role |
|---|---|
| `adv_bidder` | OpenRTB endpoint metadata owned by an advertiser, with optional synthetic reporting IDs. |
| `mid_route_group` | Operator-defined fallback group with timeout and margin defaults. |
| `mid_route_bidder` | Active bidders in a route group with optional overrides. |
| `mid_route_target` | Route assignment to global, publisher, site, slot, and optional size. |
| `ledger_mid` | Interval middleman reporting facts from callback metadata. |
| `daily_mid` | Daily middleman reporting facts aggregated from `ledger_mid`. |

Route targets may point at existing publisher entities: `3=pub`,
`31=pub_site`, and `32=pub_slot`; `NULL` means global.

## Runtime Contract

M20 enables per-impression fallback behind `middleman_enabled`. Local campaign
bids still win first. Only impressions that local campaign matching cannot fill
are considered for downstream fanout. Route group `trigger_mode='Always'` is not
active in M20.

Operators manage `mid_route_group`, `mid_route_bidder`, and `mid_route_target`
through the admin `midroute` Summer/Genelet UI. The bidder route cache is
Redis-only under `middleman:routes`. The singleton
`cmd/redis-cache -cache=redis|all` job builds it from active `adv_bidder`,
`mid_route_group`, `mid_route_bidder`, and `mid_route_target` rows. `cmd/unify`
does not refresh this cache after route edits.

Route membership alone is not enough to fan out. A bidder must also be active,
credential-ready, mapped to a valid synthetic reporting chain, and eligible for
the original upstream publisher/site/slot under the existing ACL and channel
matching rules.

Credential refs are environment-backed. `adv_bidder.credential_ref` names an
environment variable whose value is a JSON object of outbound HTTP headers, for
example `{"Authorization":"Bearer ..."}`. Hop-by-hop headers such as `Host`,
`Connection`, and `Content-Length` are rejected.

The forwarded OpenRTB request preserves the original request fields and full
impression list. It overrides `ext.request_domain` with
`middleman_exchange_domain` and does not add user profile enrichment yet. The
auction accepts downstream bids only for impressions that local campaign
matching did not fill.

The fanout budget is the minimum of remaining nonzero incoming OpenRTB `tmax`
after local matching, group timeout, bidder or route-bidder timeout, and
`middleman_timeout_ms`. Late, invalid, inactive, below-floor, or non-USD
downstream responses are discarded.

The first auction integration will preserve incoming bid floors when forwarding
and apply markup only on the response returned upstream:

```text
upstream_price = downstream_price + max(downstream_price * margin_pct, min_margin_cpm)
```

If no downstream bid survives validation and markup checks, the response remains
`204 No Content`.

The upstream bid keeps downstream ad markup, but uses the approved synthetic
campaign and creative IDs for `seat`, `cid`, `crid`, and `adid`.

M21 keeps Aofei in the OpenRTB callback path for selected middleman winners.
After the final per-impression winner is selected, `cmd/unify` stores a
short-lived callback context in Redis under `middleman:cb:<token>` and replaces
returned callback fields with signed Aofei proxy URLs:

- `/mid/win` for win notification and win audit.
- `/mid/loss` for loss notification and loss audit.
- `/mid/bill` when the downstream bid provided `burl`.
- `/mid/click` for cooperative click notification when downstream markup opts in
  to the URL supplied in forwarded request `ext`.

Billing authority is `burl` first. When a downstream bid has `burl`, `/mid/bill`
is the billable event. When a downstream bid does not have `burl`, `/mid/win`
is the billable fallback. Billable middleman events are idempotent through
`middleman:bill:<token>`.

M21 records both sides of the middleman price:

```text
charge_price = upstream callback auction_price capped at upstream returned bid price,
               or upstream returned bid price when the callback price is absent
pay_price = min(downstream_bid_price, max(0, charge_price - margin_cpm))
margin_cpm = upstream returned bid price - downstream bid price
```

Downstream callbacks receive the net payable price in their
`${AUCTION_PRICE}` macro. Winloss logs store charge price in the existing
`RAdv.Cost` field so the legacy ledger can count charge-side CPM delivery, and
store downstream bid, upstream bid, pay price, margin, callback source, and
forward status in middleman metadata.

M22 consumes that metadata in `cmd/ledger`. `ledger_mid` and `daily_mid`
preserve bidder, route, synthetic demand, and publisher dimensions. Advertiser
middleman reports show pay-side spend. Admin reports show charge spend, pay
spend, margin, win/loss counts, billable impressions, clicks, and callback
forward health.

Forwarded requests still preserve the full original impression list and add
cooperative click notify URLs under `ext.aofei_middleman.click_notify_urls`.
M21 does not rewrite arbitrary downstream `adm`, impression pixels, or click
markup.

## Milestone Sequence

- M16: advertiser-owned endpoint schema, synthetic reporting IDs, route tables,
  docs.
- M17: Summer/Genelet bidder portal, advertiser-safe writes, admin approval, and
  synthetic reporting chain creation.
- M18: Summer template modernization on `../pzdesign/tmpls`, `html/template`,
  and bidder page integration.
- M20: route cache, synthetic eligibility checks, downstream OpenRTB client, and
  per-impression fallback auction integration after local no-bid.
- M21: callback proxying, cooperative click notify, audit, and charge/pay price
  reconciliation.
- M22: advertiser pay-side reports and admin charge/pay/margin settlement views
  from middleman callback metadata.
- M23: admin route-operations UI for route groups, route bidders, and route
  targets; route-cache refresh remains a singleton `cmd/redis-cache` job.

## Post-M23 Backlog

The live middleman TODOs after M23 are route cache rollout automation beyond the
singleton `cmd/redis-cache` job, optional spread/local route snapshots,
`trigger_mode='Always'` runtime behavior, durable downstream callback retry
queues, and real invoicing/payment execution from settlement facts. Arbitrary
downstream `adm` impression/click rewriting remains closed unless a future
reporting requirement makes cooperative click notify insufficient.
