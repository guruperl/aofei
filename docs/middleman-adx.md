# Middleman AdX

Middleman AdX fallback is the planned path for bid requests that do not match a
local campaign. The current M16 implementation establishes the database and
admin identity boundary only; bid serving still returns `204 No Content` when no
local campaign produces a bid.

## Account Boundary

Downstream DSP partners are not advertisers. Advertisers (`adv`) own local
campaigns, items, creatives, balances, and targeting. Downstream DSPs use the
separate Summer/Genelet `dsp` role backed by `mid_dsp`.

DSP users can manage their own `mid_bidder` endpoint metadata and, in a later
milestone, view their own reports. Operators manage credential references,
endpoint activation, route groups, inventory assignment, and margin settings.
Secrets are not stored in MySQL; `mid_bidder.credential_ref` points to
environment-managed secret material.

## Schema

The middleman tables are:

| Table | Role |
|---|---|
| `mid_dsp` | Downstream DSP partner account for the `dsp` role. |
| `mid_bidder` | OpenRTB endpoint metadata owned by a downstream DSP. |
| `mid_route_group` | Operator-defined fallback group with timeout and margin defaults. |
| `mid_route_bidder` | Active bidders in a route group with optional overrides. |
| `mid_route_target` | Route assignment to global, publisher, site, slot, and optional size. |
| `daily_mid_bidder` | Future daily aggregate reporting by DSP and bidder endpoint. |

`def_entitytype` includes `(6, 'mid_dsp', 'dsp_id')`. Route targets may point at
existing publisher entities: `3=pub`, `31=pub_site`, and `32=pub_slot`; `NULL`
means global.

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

- M16: schema, `dsp` role, endpoint metadata modules, docs.
- M17: route cache and OpenRTB downstream client.
- M18: fallback auction integration.
- M19: callback proxying, audit, and operations.
- M20: middleman reporting for DSP users and operators.
