# DSP Workflow

This document follows the current end-to-end bid workflow from HTTP request to
response and measurement publishing.

## Request Entry

`../pzdesign/cmd/unify` wires `POST /bid/{domain}` to
`dsp.Controller.ServeBid`. The route domain is the primary publisher key. The
bid body is limited to 1 MiB before JSON unmarshalling.

The active validation boundary requires:

- a nonnil bid request,
- at least one impression,
- a nonnil device.

Malformed JSON returns `400`. Missing required runtime shape returns no content.

## Publisher And Attribute Resolution

`ServeBid` looks up `acl.Pub` by route domain from either:

- the in-process local static cache backed by spread files when
  `Config.IsLocal` is true, or
- Redis `pubmap` otherwise.

`match.NewAttributeForImp` then derives these values per impression:

- `RPub` ids from publisher site/slot maps,
- creative size and native format from the current impression,
- app/video booleans,
- user ID and IFA,
- demographic, geo, user-agent, date/hour, and ACL attributes.

Unknown site or slot strings fall back to publisher default site/slot ids.

## Candidate Loading

For each impression, the controller loads `match.RAdvs` for the resolved
`size_id` and `slot_id` from the local static cache or Redis. Each candidate
contains advertiser, campaign, item, creative, cost, weight, and frequency-cap
fields.

If no candidates exist for an impression's size/slot pair, that impression is
eligible for middleman fallback when `middleman_enabled` is true. `Fallback`
routes only fan out for local no-bid impressions. `Always` routes can also fan
out for locally filled impressions when `middleman_always_enabled` is true.
Candidate bidders must match an active route and pass the synthetic item
ACL/channel check for the original publisher/site/slot.

Middleman fanout forwards the full original request shape to each selected
bidder and overrides `ext.request_domain` with `middleman_exchange_domain`. The
auction accepts downstream bids only for impressions eligible under the selected
route trigger mode. Marked-up `Always` bids compete with local bids on effective
CPM; unsafe local price comparisons keep the local winner. It does not add
user/profile enrichment yet. Credential references resolve to environment
variables containing JSON header maps; no secret material is stored in MySQL or
Redis.

For final middleman winners, callback metadata is stored in Redis and returned
`nurl`, `lurl`, and optional `burl` fields are replaced with signed `/mid/*`
proxy URLs. `burl` is the preferred billable event; win notification is the
billable fallback only when no downstream `burl` exists. The forwarded request
also includes cooperative click notify URLs under
`ext.aofei_middleman.click_notify_urls`; downstream markup must opt in because
the DSP does not rewrite arbitrary middleman ad markup.

## Filtering

Frequency caps are checked first by reading `bothcap:<user_id>` for capped item
ids. Expired cap entries are deleted from Redis. This mutable cap state remains
Redis-backed in both Redis and local/spread bid modes.

Audiences are loaded for remaining candidates. The path then:

1. tries uploaded-audience direct matches,
2. falls back to combined audience predicates only when no direct upload match
   exists,
3. skips the impression if no candidates remain.

Combined predicates cover geo, demographic, user-agent, date/hour, and ACL
audiences. Nil audience objects are treated as wildcard matches in Redis and
spread/IO modes.

In local/spread bid mode, requests with no frequency caps and no uploaded
audience predicates can complete from the in-process static cache without Redis.
Candidates that require caps or uploaded memberships fail closed when Redis
mutable state is unavailable.

## Selection

`RAdvs.PickIndexPrice` computes a selection weight for each surviving candidate:

- local cost semantics are converted to USD eCPM,
- candidates below the current impression's `bidfloor` receive weight zero,
- surviving values are multiplied by campaign/item weight,
- a weighted random index is selected.

The response currency is always `USD`. Empty or `USD` `bidfloorcur` is
accepted. Unsupported currencies are not converted and produce no bid for that
impression.

## Creative And Response

The selected creative is loaded from the local static cache or Redis `creative`.
`match.Creative.AdM` expands landing, impression, click redirect, and configured
tracker URLs and returns one of:

- default native image markup,
- default native video markup,
- banner iframe markup with DSP impression pixels.

Native is preferred when an impression offers multiple formats, followed by
video, then banner. App inventory no longer forces native markup; app banner
inventory returns banner markup.

Native markup uses a DSP `/clk` redirect URL as the primary native link and the
creative failback as the direct fallback URL. Banner iframe markup does not wrap
arbitrary HTML or iframe content; banner creatives that want DSP click redirect
measurement must include `{CLICK_URL}` in the creative content URL/template.
`{LANDING_URL}` remains available as the direct advertiser destination.
Creative URL macro expansion preserves existing query parameters, repeated
query values, empty values, and non-macro values while replacing supported
macros per query value.

`dsp.DSP.NewBid` creates one OpenRTB bid per served impression with:

- bid id derived from request time, creative id, and impression index,
- `impid` from the current impression id,
- price from selected USD eCPM,
- win and loss URLs,
- ad markup,
- campaign and creative ids,
- bundle and categories from the matched ACL audience,
- width and height from creative size.

`ServeBid` groups bids by campaign seat. Audit events are enqueued after the
response body is written: request and response once, and one attribute event per
served impression. A bounded background publisher sends them to NATS best
effort and counts queue drops.

Middleman fallback responses are grouped under synthetic campaign seats. The
downstream ad markup is preserved, but upstream-facing callback URLs and the
campaign and creative identifiers are replaced with Aofei proxy URLs and the
approved synthetic reporting IDs. Middleman callback proxying records win/loss,
billable, cooperative click, and charge/pay audit facts; advertiser/operator
reporting remains later milestone work.

## Win, Loss, Impression, And Click

`GET /win`, `/loss`, `/imp`, and `/clk` all enter `ServeWinLoss`. The handler
unpacks demand and supply identifiers from query parameters, parses
`auction_price`, builds a `WinLoss` record, and publishes it to NATS when NATS is
available.

For `/imp` and `/clk`, the handler also unpacks cap state and refreshes Redis
frequency caps for the user/item pair. Cap refresh uses Redis `WATCH`/`MULTI`/
`EXEC` with bounded retry so concurrent tracker callbacks update the existing
`bothcap:<user_id>` binary payload atomically.

When `/clk` receives a valid HTTP(S) `redirect` query parameter and the normal
packed tracking fields, it records the click best-effort and returns `302` to
that target. The redirect URL must carry a valid HMAC `sig` generated from the
full concrete `/clk` query payload, including `redirect`; unsigned or modified
redirects return `400`. Without `redirect`, `/clk` remains a tracking-only
endpoint and returns no content.

## Known Workflow Boundaries

- Request, response, and attribute logs are best effort after response write.
- Ledger spend is based on impression tracker records, not win records.
- Redis remains the shared mutable-state backend. Redis mode can also serve
  static cache reads, while local/spread mode serves static publisher, slot,
  audience, and creative data from in-process snapshots backed by spread files.
  Local snapshots are loaded at controller startup and refreshed through the
  explicit reload hook; request handlers read the current in-memory snapshot.
