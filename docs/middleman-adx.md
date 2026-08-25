# Middleman AdX

Middleman AdX is the config-gated path for bid requests that do not match a
local campaign and, behind a separate gate, for effective-CPM competition with
local demand. The advertiser-owned bidder endpoint, route schema, Summer portal,
cache, OpenRTB client, callback proxy/retry, and settlement facts are
implemented. Production activation follows the staged, reversible evidence
contract in [middleman-activation.md](middleman-activation.md); checked-in
defaults keep external disclosure disabled.

## Account Boundary

Downstream bidder endpoints are owned by advertisers (`adv`). A downstream
partner uses the existing advertiser login and account tooling instead of a
separate Summer/Genelet role.

Advertiser users can manage safe `adv_bidder` endpoint metadata and view their
account-scoped middleman results through the implemented reports. Their writable
fields are bidder name, endpoint URL, OpenRTB version, seat, and timeout.
Advertisers can see credential status and active status, but cannot set
credential refs, synthetic reporting IDs, or activation. Operators manage
credential references, endpoint activation, route groups, inventory assignment,
and margin settings. Secrets are not stored in MySQL;
`adv_bidder.credential_ref` points to environment-managed secret material.

Each `adv_bidder` may reference synthetic campaign, item, and creative rows.
Those rows are reporting identities, not normal local demand. They let existing
ledger joins preserve charge-side delivery through `adv_campaign`, `adv_item`,
`adv_creative`, `ledger_adv`, and `daily_adv`. Middleman-specific `ledger_mid`
and `daily_mid` tables provide pay-side advertiser reporting and operator
charge/pay/margin settlement views.

Operator tooling must ensure the synthetic IDs form one chain owned by the same
advertiser: `adv_bidder.adv_id -> adv_campaign.adv_id`,
`adv_item.campaign_id -> adv_campaign.campaign_id`, and
`adv_creative.item_id -> adv_item.item_id`.

M17 approval creates the chain when all three synthetic IDs are empty, reuses an
existing complete chain after same-advertiser validation, and rejects partial
synthetic state. Created synthetic campaign, item, and creative rows are
inactive so they cannot become local demand.

The same synthetic chain is also the active inventory eligibility surface.
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

The runtime therefore does not add a separate bidder-vs-site ACL table. Route
groups select the coarse fanout pool for a publisher/site/slot, then runtime
eligibility filters each `adv_bidder` through its synthetic item audience rules
before forwarding the request downstream. A future schema may reopen that
choice only if the existing model proves insufficient.

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
- Admin route health:
  `/goto/admin/g/midroute?action=health`

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
| `mid_callback_retry` | Durable retry queue for retryable downstream post-auction callback forwarding failures. |
| `ledger_mid` | Interval middleman reporting facts from callback metadata. |
| `daily_mid` | Daily middleman reporting facts aggregated from `ledger_mid`. |

Route targets may point at existing publisher entities: `3=pub`,
`31=pub_site`, and `32=pub_slot`; `NULL` means global.

## Runtime Contract

Middleman runtime remains behind `middleman_enabled`. `Fallback` route groups
fan out only for impressions that local campaign matching cannot fill. M25 adds
`Always` route behavior behind the separate `middleman_always_enabled` gate,
default false. When that gate is off, `Always` routes are ignored at runtime.
When it is on, matching `Always` routes may fan out even for impressions where
local campaigns produced a bid.

Operators manage `mid_route_group`, `mid_route_bidder`, and `mid_route_target`
through the admin `midroute` Summer/Genelet UI. The bidder route cache is
Redis-only. M25 runtimes prefer `middleman:routes:v2`, while the legacy
`middleman:routes` key remains fallback-only for M24 rolling-deploy safety. The
singleton `cmd/redis-cache -cache=redis|all|routes` job builds both keys from
active `adv_bidder`, `mid_route_group`, `mid_route_bidder`, and
`mid_route_target` rows. `cmd/unify` does not refresh this cache after route
edits.

HTTP workers memoize the decoded preferred/fallback route result for
`middleman_route_cache_ttl_ms`, default 5000 ms. Concurrent cache misses share
one Redis fetch and decode. A failed refresh is cached for the same short
interval and disables middleman fanout rather than reusing an expired route
snapshot; an existing local winner remains eligible under the normal fallback
rules. The shared fetch has an independent `middleman_timeout_ms` deadline.
Each caller waits with its own request context, so cancellation of the caller
that starts a refresh does not cancel the load or propagate its cancellation to
other waiters; the eventual result or error is still cached. `/debug/vars`
exposes route-cache hit, miss, refresh, and refresh-error counters as
`aofei_middleman_route_cache_hits_total`,
`aofei_middleman_route_cache_misses_total`,
`aofei_middleman_route_cache_refreshes_total`, and
`aofei_middleman_route_cache_refresh_errors_total`.

M24 version-1 route payloads added optional metadata: generation time, entry
count, source, route-table high-water timestamp, and a checksum over route
entries. M25 writes version-2 payloads with `trigger_mode` under
`middleman:routes:v2`; older version-1 payloads without metadata or trigger
mode remain readable, and a missing trigger mode is treated as `Fallback`. The
admin `midroute` topics page and JSON output expose preferred-key Redis metadata
beside the current MySQL route high-water timestamp. The admin UI is
visibility-only; operators still run the singleton cache command on the cache
node to publish route changes.

`midroute?action=health` reports active route groups with no active targets or
route bidders, active route bidders whose downstream bidder is inactive or not
credential-approved, missing `credential_ref` names, and invalid synthetic
campaign/item/creative chains. Credential secret values are never read or shown;
the UI shows only `credential_ref` names and statuses.

Route membership alone is not enough to fan out. A bidder must also be active,
credential-ready, mapped to a valid synthetic reporting chain, and eligible for
the original upstream publisher/site/slot under the existing ACL and channel
matching rules.

Credential refs are environment-backed. `adv_bidder.credential_ref` names an
environment variable whose value is a JSON object of outbound HTTP headers, for
example `{"Authorization":"Bearer ..."}`. Hop-by-hop headers such as `Host`,
`Connection`, and `Content-Length` are rejected.

## OpenRTB 2.5 Partner Profile

Active bidders use one exact, operator-approved profile:

- `POST` to an absolute HTTP(S) endpoint without URL user info or a fragment;
- `Content-Type: application/openrtb+json`, OpenRTB/JSON `Accept`, and
  `x-openrtb-version: 2.5`;
- one independently scoped request containing only assigned, uniquely named
  impressions, exactly one supported Banner/Video/Native intent per imp, and
  USD CPM floors/currency;
- the minimum of remaining incoming `tmax`, configured middleman timeout,
  route-group timeout, bidder timeout, and optional route-bidder timeout;
- optional configured buyer `seat`, which must match every accepted seatbid;
- W8M-controlled `request_domain` and click-notification extension only;
- HTTPS callback and creative resources whenever `imp.secure=1`.

Response `id` must equal request `id`; response currency defaults to USD and
any explicit non-USD value is rejected. Group/all-or-none seatbids are not
supported. Raw downstream CPM must meet the forwarded floor before W8M margin
is added. Bid IDs must be nonempty and unique, and active synthetic reporting
IDs remain the only local campaign/item/creative identities. Partner win,
billing, loss, and cooperative click callbacks are proxied through the existing
signed callback contract; they do not create an additional W8M billable event.

Partner response JSON may be gzip encoded. Both compressed and decompressed
bodies remain capped at 1 MiB and under the call deadline. Credential-free
fixtures live in `dsp/testdata/openrtb/` and use example domains.

External disclosure requires both `middleman_enabled` and the separate
default-false `privacy_contextual_middleman_enabled` gate. Every bidder request
is rebuilt independently as typed OpenRTB, contains only impressions assigned
to that bidder, and is contextual even when local matching had a personalized
grant. User and device identifiers, IP, raw UA, precise geo, demographics,
search/query data, unknown fields, and every uncontrolled `ext` are removed.
W8M adds back only controlled `ext.request_domain` and cooperative click
notification metadata. The auction accepts downstream bids only for impressions
eligible under the selected route trigger mode.

The fanout budget is the minimum of remaining nonzero incoming OpenRTB `tmax`
after local matching, group timeout, bidder or route-bidder timeout, and
`middleman_timeout_ms`. Late, invalid, inactive, below-floor, or non-USD
downstream responses are discarded.

Discard reasons and bidder latency are exposed only through the fixed metrics
documented in [production-traffic-observability.md](production-traffic-observability.md).
Optional sampled diagnostics are hashed and metadata-only. Raw bid bodies,
consent strings, endpoint/callback URLs, credentials, and creative markup are
never logged or returned in a public debug body.

D02 validates each downstream winner before markup or competition: positive
finite USD price, synthesized floor, selected impression ID, approved synthetic
IDs, exact dimensions, compatible media type, safe callbacks, and secure
inventory. Banner markup cannot navigate or escape the approved container;
Video must be compatible VAST; Native must decode to requested assets with safe
click/tracker/image URLs and all required assets present. Rejected responses do
not displace a local winner. See
[auction-pricing-creatives.md](auction-pricing-creatives.md).

The first auction integration will preserve incoming bid floors when forwarding
and apply markup only on the response returned upstream:

```text
upstream_price = downstream_price + max(downstream_price * margin_pct, min_margin_cpm)
```

Under `usd-cpm-impression-v3`, downstream and minimum-margin CPM are six-place
integers and `margin_pct` is a four-place integer fraction. The percentage
product rounds half away from zero once to a micro-USD CPM before the exact
minimum is applied; over-scale downstream prices and marked-price overflow are
rejected.

If no downstream bid survives validation and markup checks, `Fallback`
impressions remain no-bid and `Always` impressions keep their local winner when
one exists.

The upstream bid keeps downstream ad markup, but uses the approved synthetic
campaign and creative IDs for `seat`, `cid`, `crid`, and `adid`.

M25 chooses one final winner per impression. Local matching still runs first.
For `Always` route candidates, marked-up middleman bids compete with local bids
on effective CPM. If the middleman bid is higher, the response uses the
synthetic reporting IDs and M21 callback proxying. If the local bid is higher or
price comparison is unsafe, the local bid remains the winner and existing local
callback and ledger behavior are preserved.

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
`middleman:bill:<token>`. Downstream notification results are retained under
`middleman:notify:<source>:<token>`, while local win/loss/click publication uses
the independent `middleman:publish:<source>:<token>` family. A local publication
retry therefore does not resend a successfully forwarded downstream callback,
and duplicate `/mid/win`, `/mid/loss`, or `/mid/click` requests do not republish
facts. If the downstream side effect succeeds but its result cannot be stored,
the request fails and clears notification ownership; the resulting downstream
retry window remains deliberately at-least-once and requires idempotent partner
callbacks.

Notify, publish, and bill markers begin as random-token-owned processing claims
with a lease that covers the configured downstream timeout. Successful state or
local publication atomically converts the matching owner claim to a completed
callback-TTL marker. Publication failure releases only the matching owner on a
bounded context detached from HTTP cancellation. A process exit leaves only a
short processing lease, so a retry can resume instead of suppressing an
unpublished fact for the full callback TTL; a late owner cannot clear or
complete a replacement claim.
`middleman_callback_ttl_seconds` must cover the complete accepted tracking
signature lifetime, including the five-minute future-clock-skew allowance, and
the processing lease. Its default is therefore 86,700 seconds when the tracking
signature TTL is 24 hours. `middleman_callback_timeout_ms` is bounded to
1..60,000 milliseconds.

Malformed signatures and expired, missing, or corrupt callback context return
HTTP `400`. Redis/context dependency failures, retryable local publication
failures, and retryable downstream failures that cannot be placed in the durable
queue return `503` so the exchange can retry. A retryable downstream failure
that is durably queued retains the completed notify marker and returns the normal
`204`; later queue processing forwards downstream only and never republishes the
local fact.

`aofei_middleman_callback_outcomes_total` exposes only fixed outcome keys for
in-flight/completed forward duplicates, local publication/billing duplicates,
retryable local publication, retry queue availability, and claim release or
release/completion failure. It never uses a callback token, auction identity, partner
endpoint, or inventory identity as a metric key. Callback retry warnings contain
only the bounded callback source and a fixed dependency reason, never the raw
callback token.

M21 records both sides of the middleman price:

```text
charge_price = exact upstream CPM stored in the Redis callback context
pay_price = exact downstream CPM stored in the same context
margin_cpm = charge_price - pay_price
```

Incoming middleman `auction_price` and `auction_currency` query parameters are
not trusted for ledger math. The callback context carries a v3 contract marker
and exact charge/pay/margin identity; an inconsistent or out-of-range context
fails closed. Downstream callbacks receive the net payable price
in their `${AUCTION_PRICE}` macro. Winloss logs store charge-side CPM in the
exact CPM field; `usd-cpm-impression-v3` maps each billable impression to
integer nano-USD charge, pay, and margin by dividing the respective CPM values
by 1000. The logs also preserve downstream bid, upstream bid, pay price,
margin, callback source, and forward status in middleman metadata.

M22 consumes that metadata in `cmd/ledger`. `ledger_mid` and `daily_mid`
preserve bidder, route, synthetic demand, and publisher dimensions. Advertiser
middleman reports show pay-side spend. Admin reports show charge spend, pay
spend, margin, win/loss counts, billable impressions, clicks, and callback
forward health.

M24 adds durable retry for retryable downstream callback forwarding failures.
Only post-auction `/mid/win`, `/mid/loss`, and `/mid/bill` forwards are
eligible. Network/request errors, HTTP 429, and HTTP 5xx responses enqueue a
row in `mid_callback_retry` with the expanded downstream URL and enough
auction, route, bidder, price, and currency context to retry after the Redis
callback context expires. Missing URLs, invalid URLs, duplicate notifications,
and HTTP 4xx responses other than 429 are not queued.

M26 signs middleman proxy URLs with `sig_ts` and rejects expired callbacks.
Bidder fanout and downstream callback forwarding share the `internal/safehttp`
address policy before requests and again on every dial. It rejects private,
loopback, link-local, unspecified, multicast, CGNAT, benchmarking,
documentation, protocol-transition, reserved, and future non-public ranges;
IPv4-mapped IPv6 addresses follow the IPv4 policy. A DNS answer containing any
denied address fails closed, including after re-resolution, so a mixed answer
cannot select only its public member. The retry command applies the same guard.
The reviewed prefix policy follows the IANA
[IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry/) and
[IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry/)
special-purpose registries; registry changes require a code and test review,
not an implicit expansion through `net.IP.IsGlobalUnicast`.

Injected HTTP clients cannot replace this boundary. A supported network client
must use `*http.Transport`; Aofei clones its TLS and connection-pool settings,
then removes proxy and custom TLS-dial hooks, installs checked/pinned dialing,
requires certificate verification and TLS 1.2 or newer, derives certificate
identity from each request host, and caps response headers at 64 KiB. Arbitrary
custom network `RoundTripper` implementations fail closed. The only alternative
is an explicitly marked, socket-free in-memory transport used by tests, and its
request URLs are still validated. Client timeout and caller context remain in
force, while the bidder parser retains its independent one-MiB encoded/decoded
response limit and callbacks retain their 1-KiB drain bound.

Every redirect target is validated after any injected redirect hook and again
by the transport. At most ten hops are followed. Any authority or scheme change
clears all request headers and URL credentials, and callback/bidder clients do
not retain a cookie jar. Application-specific URL guards are additive and
cannot relax the mandatory policy.

The singleton `cmd/mid-callback-retry` command claims due rows as `Processing`
before forwarding, then marks them `Succeeded`, `Retrying`, or `Abandoned` with
bounded attempts and exponential backoff. It forwards downstream callbacks only;
it does not republish win/loss or delivery records, so ledger counts remain
idempotent. Once a retryable `/mid/*` failure is durably queued, duplicate
exchange callbacks remain suppressed by the Redis notify key and are not
enqueued again.

Deploy the split notify/publication lifecycle as one callback-tier rollout:
drain or route `/mid/*` consistently to upgraded workers instead of allowing an
old worker and a new worker to alternate on one callback token. Preserve
`middleman:notify:*`, `middleman:publish:*`, `middleman:bill:*`, and queued retry
rows during rollout and rollback. If rollback is required, roll the callback
tier together and let existing marker TTLs expire naturally; deleting marker
families can refire downstream side effects and is not a rollback procedure.

Forwarded requests contain only that bidder's assigned impression list and add
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
- M24: route-cache freshness/health visibility, route-only cache refresh, and
  durable retry for retryable downstream callback forwards.
- M25: gated `trigger_mode='Always'` fanout and effective-CPM winner selection
  between local and middleman bids.

## Current Activation Boundary

D03 keeps routes Redis-only: their short shared refresh/revocation timeline is
safer than duplicating it into spread/local snapshots. The read-only
`cmd/redis-cache -validate-middleman` preflight compares active MySQL routes to
the published Redis v2 checksum/high-water mark, validates profiles and config,
and resolves environment credential references without exposing values.

Hosted funding/payout remains A02 work; current charge/pay/margin facts feed A01
manual accounting. Arbitrary downstream `adm` impression/click rewriting
remains closed unless a future reporting requirement makes cooperative click
notification insufficient.
