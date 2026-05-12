# Middleman AdX

Middleman AdX fallback is the planned path for bid requests that do not match a
local campaign. The current M16 implementation establishes the advertiser-owned
bidder endpoint and route schema only; bid serving still returns
`204 No Content` when no local campaign produces a bid.

## Account Boundary

Downstream bidder endpoints are owned by advertisers (`adv`). A downstream
partner uses the existing advertiser login and account tooling instead of a
separate Summer/Genelet role.

Advertiser users can manage their own `adv_bidder` endpoint metadata and, in a
later milestone, view middleman results through advertiser reports. Operators
manage credential references, endpoint activation, route groups, inventory
assignment, and margin settings. Secrets are not stored in MySQL;
`adv_bidder.credential_ref` points to environment-managed secret material.

Each `adv_bidder` may reference synthetic campaign, item, and creative rows.
Those rows are reporting identities, not normal local demand. They let existing
ledger and advertiser reports roll up spend, impressions, clicks, and publisher
breakdowns through the current `adv_campaign`, `adv_item`, `adv_creative`,
`ledger_adv`, and `daily_adv` joins.

Operator tooling must ensure the synthetic IDs form one chain owned by the same
advertiser: `adv_bidder.adv_id -> adv_campaign.adv_id`,
`adv_item.campaign_id -> adv_campaign.campaign_id`, and
`adv_creative.item_id -> adv_item.item_id`.

## Schema

The middleman tables are:

| Table | Role |
|---|---|
| `adv_bidder` | OpenRTB endpoint metadata owned by an advertiser, with optional synthetic reporting IDs. |
| `mid_route_group` | Operator-defined fallback group with timeout and margin defaults. |
| `mid_route_bidder` | Active bidders in a route group with optional overrides. |
| `mid_route_target` | Route assignment to global, publisher, site, slot, and optional size. |

Route targets may point at existing publisher entities: `3=pub`,
`31=pub_site`, and `32=pub_slot`; `NULL` means global.

## Planned Runtime Contract

Fallback routing will run only after internal campaign matching produces no bid.
The fanout budget will be the minimum of the incoming OpenRTB `tmax`, the route
group timeout, and the DSP config default. Late, invalid, inactive, or non-USD
downstream responses will be discarded.

The first auction integration will preserve incoming bid floors when forwarding
and apply markup only on the response returned upstream:

```text
upstream_price = downstream_price + max(downstream_price * margin_pct, min_margin_cpm)
```

If no downstream bid survives validation and markup checks, the response remains
`204 No Content`.

## Milestone Sequence

- M16: advertiser-owned endpoint schema, synthetic reporting IDs, route tables,
  docs.
- M17: route cache and OpenRTB downstream client.
- M18: fallback auction integration.
- M19: callback proxying, audit, and operations.
- M20: advertiser and operator reporting using synthetic campaign/item/creative
  rows.
