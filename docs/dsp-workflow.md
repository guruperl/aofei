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

After validation and before publisher, cache, cap, or audience work, the
controller evaluates COPPA, GDPR/TCF, GPP/US Privacy, GPC, DNT, and LMT. Missing
or non-granting signals become contextual; COPPA becomes restricted. The
request is minimized before matching, and any accepted local identity is
HMAC-pseudonymized before mutable cap or tracking state. See
[privacy-data-governance.md](privacy-data-governance.md).

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
contains advertiser, campaign, item, creative, cost, weight, frequency-cap, and
versioned delivery-policy fields.

If no candidates exist for an impression's size/slot pair, that impression is
eligible for middleman fallback when `middleman_enabled` is true. `Fallback`
routes only fan out for local no-bid impressions. `Always` routes can also fan
out for locally filled impressions when `middleman_always_enabled` is true.
Candidate bidders must match an active route and pass the synthetic item
ACL/channel check for the original publisher/site/slot.

Middleman fanout additionally requires
`privacy_contextual_middleman_enabled`. It builds a fresh contextual OpenRTB
view for each selected bidder, includes only that bidder's assigned
impressions, removes identities, precise device/user data, unknown fields, and
all uncontrolled extensions, then adds controlled `ext.request_domain` and
click metadata. Marked-up `Always` bids compete with local bids on effective
CPM; unsafe local price comparisons keep the local winner. Credential
references resolve to environment variables containing JSON header maps; no
secret material is stored in MySQL or Redis.

For final middleman winners, callback metadata is stored in Redis and returned
`nurl`, `lurl`, and optional `burl` fields are replaced with signed `/mid/*`
proxy URLs. `burl` is the preferred billable event; win notification is the
billable fallback only when no downstream `burl` exists. The forwarded request
also includes cooperative click notify URLs under
`ext.aofei_middleman.click_notify_urls`; downstream markup must opt in because
the DSP does not rewrite arbitrary middleman ad markup.

I01 fixes that downstream contract at OpenRTB 2.5. Each outbound request uses
one uniquely identified, assigned impression scope per partner request, exactly
one supported media intent per impression, USD CPM floors, a bounded deadline,
and only controlled extensions. Responses must echo request/seat identity and
pass raw-price/floor, lateness, callback, media, size, secure-markup, and active
synthetic reporting checks before they can compete. Gzip transport is bounded
on both encoded and decoded bytes. See [middleman-adx.md](middleman-adx.md).

## Filtering

Campaign/ad-group UTC windows, weekly calendars, cached hard limits, and the
delivery snapshot deadline are checked before shared mutable-state reads. This
means out-of-window or stale demand cannot consume cap or reservation state.

Frequency caps are then checked by reading `bothcap:<user_id>` for capped item
ids. Expired numbered windows stop blocking immediately and that individual
counter resets on the next accepted callback; bid-time matching does not delete
the shared item record because its sibling click window or throttle timestamp
may still be active. Idle hash retention remains governed by
`cap_state_ttl_seconds`. This mutable cap state remains Redis-backed in both
Redis and local/spread bid modes.

A numbered impression or click cap requires a positive corresponding period.
The cache compiler rejects missing, negative, or out-of-wire-range cap values;
runtime matching also fails closed if invalid cap data is supplied by an older
cache. A standalone positive impression throttle remains supported.

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

`RAdvs.PickIndexPrice` accepts only positive finite USD CPM local demand. It
first chooses the highest CPM demand unit that satisfies the synthesized floor.
Equal-price demand is ordered deterministically by campaign, item, then
advertiser ID. Positive finite creative weights rotate only among creatives in
that winning advertiser/campaign/item unit; weight cannot make lower-priced
demand win. Legacy CPC, CPA, and ROI rows are disabled until a commercial owner
assigns a reviewed CPM value.

Before reserving delivery, the selected creative is loaded and checked for
media type, exact dimensions, request MIME, secure inventory, structured native
assets, and safe source/landing/tracker URLs. A failed creative is removed and
the auction is re-evaluated without touching delivery state. The validated
candidate's selected USD CPM is then
converted to one-impression USD spend (`CPM / 1000`). One Redis Lua operation
atomically reserves that amount and one impression across all configured
campaign/ad-group total/daily scopes. A hard-limit or deterministic pacing
rejection removes the whole selected demand unit and repeats selection. Any
other reservation error makes limited local demand fail closed and permits the
existing middleman-fallback decision. Unlimited demand does not create delivery
keys.

The response currency is always `USD`. Empty or `USD` `bidfloorcur` is
accepted. Unsupported currencies are not converted and produce no bid for that
impression.

## Creative And Response

The already validated creative is materialized from the local static cache or
Redis `creative`. Banner sources are absolute image or HTML-document URLs,
Video sources are absolute media URLs materialized as VAST 3.0, and Native
sources are versioned structured JSON mapped to the requested native assets.
Raw local HTML/VAST source is not accepted. `match.Creative.AdM` expands
landing, impression, click redirect, and configured tracker URLs.

Privacy-sensitive legacy macros for raw IP, UA, city, carrier, precise device
details, and device identifiers expand to empty values. Coarse supply, country,
region, OS/device class, language, and campaign/creative macros remain.

The impression must request exactly the matching Banner, Video, or Native
format. App inventory no longer forces native markup; app Banner inventory
returns Banner markup.

Native markup uses a DSP `/clk` redirect URL as its primary native link. The
legacy optional creative-cache fallback field remains decodable, but the
database compiler leaves it empty instead of misinterpreting the campaign's
external business identifier as a URL. Banner iframe markup does not wrap
arbitrary HTML or iframe content; banner creatives that want DSP click redirect
measurement must include `{CLICK_URL}` in the creative content URL/template.
`{LANDING_URL}` remains available as the direct advertiser destination.
Creative URL macro expansion preserves existing query parameters, repeated
query values, empty values, and non-macro values while replacing supported
macros per query value. Overlapping names are resolved longest first and then
lexically, with standard macros taking precedence over duplicate custom names,
so output does not depend on Go map iteration order.

Middleman winners pass a parallel downstream-response gate for positive finite
USD price, floor, exact size, media type, callbacks, secure inventory, and
format-specific markup/native assets before competing with local demand. The
full contract and migration sequence are in
[auction-pricing-creatives.md](auction-pricing-creatives.md).

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
billable, cooperative click, and charge/pay audit facts. A01 ledger/accounting
and R02 account-scoped reports expose the resulting advertiser/operator
charge, pay, margin, delivery, and callback views under their documented
freshness and authorization rules.

## Win, Loss, Impression, And Click

`GET /win`, `/loss`, `/imp`, and `/clk` all enter `ServeWinLoss`. The handler
unpacks demand and supply identifiers from query parameters, parses
`auction_price`, builds a `WinLoss` record, and publishes it to NATS when NATS is
available.

For `/imp` and `/clk`, the handler also unpacks cap state and refreshes Redis
frequency caps for the user/item pair. Cap refresh uses Redis `WATCH`/`MULTI`/
`EXEC` with bounded retry so concurrent tracker callbacks update the existing
`bothcap:<user_id>` binary payload atomically. Signature validation precedes all
Redis work. Claim and cap Redis failures are fail-open for valid measurement
events, while malformed payloads remain errors. Keyed events use a transactional
per-event cap marker even when replay-claim acquisition fails; unkeyed events
publish without cap mutation. Redis operations run for at most two seconds on a
context detached from HTTP cancellation.

Each successful cap mutation rewrites that item to the version-2 `BothCap`
wire format. Its UTC epoch-minute trailer is authoritative through the 90-day
retention window; its leading legacy view saturates elapsed minutes rather than
wrapping. New workers read both legacy-only and version-2 values, while old
workers can read the version-2 prefix during a bounded rolling deployment.

When `/clk` receives a valid HTTP(S) `redirect` query parameter and the normal
packed tracking fields, it records the click best-effort and returns `302` to
that target. The redirect URL must carry a valid HMAC `sig` generated from the
full concrete `/clk` query payload, including `redirect`; unsigned or modified
redirects return `400`. Without `redirect`, `/clk` remains a tracking-only
endpoint and returns no content.

## Known Workflow Boundaries

- Request, response, and attribute logs are best effort after response write.
- Routine successful bids and expected no-bids do not emit process logs. Bid
  request rejections use structured debug fields, while response, audit, and
  middleman operational failures use structured warning or error fields.
- Ledger spend is based on impression tracker records, not win records.
- Redis remains the shared mutable-state backend. Redis mode can also serve
  static cache reads, while local/spread mode serves static publisher, slot,
  audience, and creative data from in-process snapshots backed by spread files.
  Local snapshots are loaded at controller startup and refreshed through the
  bounded background reload loop (with an explicit reload hook retained for
  controlled use); request handlers read the current in-memory snapshot.
