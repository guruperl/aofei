# SSP Direct Publisher Traffic

Direct SSP traffic is the browser/API publisher tag entrypoint for publishers
managed by the existing `pub` role. It is separate from ADX/OpenRTB traffic on
`/bid/{domain}`.

M28 wires runtime serving on `POST /pz`. M29 makes publisher slot pages generate
working browser/API samples for that runtime path. M30 adds measurement
semantics. M31 adds browser origin/referrer enforcement and documents the
product boundary. M32 adds mobile/API request fields and explicit JSON/OpenRTB
response formats. M33 lets valid direct SSP auctions use the existing
middleman fallback and gated `Always` runtime after local matching. M34 records
the richer supply taxonomy direction in
[ADR 0001](adr/0001-richer-supply-taxonomy.md). M35 records the account/schema
boundary decision in [ADR 0002](adr/0002-ssp-account-schema-boundary.md): keep
the existing `pub` ownership model. P02 implements that taxonomy, approved
seller metadata, privacy-safe reporting dimensions, and server-generated
OpenRTB supply chains without changing the `/pz` request schema. The handler
validates packed direct SSP
tokens against the publisher-by-id cache, enforces
the browser page host against the cached site host, converts ad units into
internal OpenRTB impressions, serves local Aofei demand, optionally fans out to
middleman bidders, renders the requested response shape, and identifies SSP
traffic separately in audits.

The public O01/I01 gate accepts one `Content-Encoding: gzip` for `/pz`, with
independent encoded and decoded body limits. JSON responses are gzip encoded
only when the client explicitly accepts gzip; no-fill behavior is unchanged.
Any synthesized request sent to an external DSP uses the exact OpenRTB 2.5
profile in [middleman-adx.md](middleman-adx.md).

P03 is now in progress. Its accepted
[authenticity contract](direct-ssp-authenticity.md) records that the current
packed browser values are enumerable, that even future HMAC-protected browser
locators remain public and replayable, and that the current credentialless SDK
path is not publisher/App authentication. This document describes the current
v1 runtime; the P03 contract governs the target security boundary and rollout
invariants until later P03 rows implement it.

## V1 Browser Contract

Browser tags post JSON to `/pz` and receive a JSON array of HTML strings. The
array order matches the input `adUnits` order. A later no-fill response returns
an empty string at the matching array position.

Publisher slot pages generate a complete HTML sample per slot:

```html
<!doctype html>
<html>
<head>
<meta charset="utf-8">
<script src="https://aofei.example/js/ads.js"></script>
</head>
<body>
<div id="pz-slot-13" style="width:300px;height:250px"></div>
<script>
pzLoadAds({
  "platform": "browser",
  "site": "AAAACAH774AAA",
  "adUnits": [{
    "code": "pz-slot-13",
    "slot": "AAAACAAUAMAAA",
    "mediaTypes": {
      "banner": {
        "size": [300, 250]
      }
    }
  }]
});
</script>
</body>
</html>
```

Request fields:

```json
{
  "id": "optional-request-id",
  "platform": "browser",
  "responseFormat": "html",
  "site": "AAAACAH774AAA",
  "adUnits": [
    {
      "code": "pz-slot-dom-id",
      "slot": "AAAACAAUAMAAA",
      "floor": 1.2,
      "mediaTypes": {
        "banner": {
          "size": [300, 250]
        }
      }
    }
  ]
}
```

`site` is the historical base32 no-padding little-endian packing of
`(pub_id, site_id)`.

`adUnits[].slot` is the same packing of `(slot_id, size_id)`.
`pub_slot.size_id` stores the publisher-configured packed width/height used for
generated slot tags. `pub_slot.bidfloor` is the server-owned USD CPM minimum.
Runtime validation rejects a slot token whose `size_id` does not match the
cached configured size, rejects declared media dimensions that do not match
that size, and builds `imp.bidfloor` from the greater of the configured and
finite non-negative request floors.

`adUnits[].code` is only the publisher page DOM element id. It is echoed for UI
and debug use, but it is not trusted as supply identity in the v1 contract.
`code` is required, must be 1-128 URL/DOM-safe ASCII characters
(`A-Z`, `a-z`, `0-9`, `_`, `.`, `:`, or `-`), and must be unique within a
request because it becomes the OpenRTB impression id used by downstream bid
responses.
Older Holiday samples used `code` as the slot token; the parser can recognize
that shape for compatibility tests, but supply validation for the v1 contract
requires `slot`.

Supported `mediaTypes` keys for the v1 contract are `banner`, `video`, and
`native`, with exactly one required per ad unit. Runtime conversion uses the
trusted `size_id` from `adUnits[].slot`. Declared banner/video/native-image
dimensions must equal it; partial, zero, or negative dimensions are invalid.

Valid requests always return `200 application/json` with one string per input
ad unit when `responseFormat` is omitted or `"html"`. No-fill units are
represented by `""` at the matching array position:

```json
["<iframe ...></iframe><img ...>", ""]
```

`responseFormat:"json"` returns one object per input ad unit:

```json
[
  {
    "filled": true,
    "adm": "<iframe ...></iframe>",
    "impressionUrl": "https://aofei.example/imp?...",
    "clickUrl": "https://aofei.example/clk?...",
    "price": 1.2,
    "currency": "USD",
    "adId": "10000",
    "campaignId": "10",
    "creativeId": "10000",
    "width": 300,
    "height": 250
  },
  {
    "filled": false
  }
]
```

Native fills include a parsed `native` object alongside `adm`. Middleman fills
use the returned `adm`, price, ids, and dimensions; they omit local-only
`impressionUrl` and `clickUrl` fields.
`responseFormat:"openrtb"` returns an OpenRTB `BidResponse`; all-no-fill
requests still return `200` with an empty `seatbid`. Filled bids are grouped by
the final winner seat, including synthetic middleman seats when middleman wins.

API examples on publisher slot pages use the SDK JSON response shape:

```text
POST https://aofei.example/pz

{
  "platform": "sdk",
  "responseFormat": "json",
  "site": "AAAACAH774AAA",
  "app": {
    "bundle": "example.com"
  },
  "device": {
    "os": "Android",
    "language": "zh"
  },
  "adUnits": [{
    "code": "pz-slot-13",
    "slot": "AAAACAAUAMAAA",
    "mediaTypes": {
      "banner": {
        "size": [300, 250]
      }
    }
  }]
}

Response:
[{
  "filled": true,
  "adm": "<iframe ...></iframe>",
  "impressionUrl": "https://aofei.example/imp?...",
  "clickUrl": "https://aofei.example/clk?...",
  "price": 1.2,
  "currency": "USD",
  "adId": "10000",
  "campaignId": "10",
  "creativeId": "10000",
  "width": 300,
  "height": 250
}]
```

This sample is intentionally contextual and contains no user/device identifier.
A publisher must propagate applicable `regs`, `user.consent`, GPP/US signals,
and device opt-out fields from its approved consent flow; it must not invent a
grant or copy the example as consent evidence.

Malformed JSON returns `400`, oversized bodies return `413`, and invalid direct
tokens, missing slots, unsupported media, unknown publishers, or site/slot
mismatches, including size mismatches, return `400`.

After token and cache validation, browser requests must prove that the page host
matches the cached site host. A request is treated as browser traffic unless
`platform` is `sdk`. Browser requests must include at least one valid `Origin`
or `Referer`, and every present `Origin` or `Referer` must be an absolute URL
whose host equals the validated cached site string exactly. `Origin: null`,
malformed URLs, missing browser headers, mismatched hosts, and subdomain
variants return `403`. SDK requests may omit both headers, but if they include
either header it must still match. Policy rejections do not set
`aofei_pz_uid`, do not bid, and do not publish request, response, or attribute
audits.

The `aofei_pz_uid` cookie is part of the browser contract only, and only an
accepted configured personalization grant permits it to be read or set. It is
not used for missing, denied, opt-out, invalid, COPPA, or currently unmapped GPP
signals. `platform:"sdk"` requests never read, rotate, set, or propagate this
cookie. The default cookie lifetime is 30 days.

## Cache Lookup

Existing `/bid/{domain}` traffic reads the publisher cache by route domain from
the Redis hash `pubmap` or the local spread `pubmap/` snapshot. That behavior is
unchanged.

Direct tags cannot trust a browser route domain for publisher identity, so M27
adds an additive Redis hash:

```text
pubmap:by-id
```

Keys are decimal `pub_id` values. Values contain the active publisher object,
the domain string, reverse maps from `site_id` and `(site_id, slot_id)` back to
the site and slot strings used by ACL matching, and cache-owned site type,
slot-size, and configured-floor metadata.

Local/static mode derives the same by-id lookup in memory from the loaded
`pubmap` snapshot. It does not add a separate spread file family.

Validation checks performed on the `/pz` request path:

- unpack `site` into `pub_id` and `site_id`;
- look up the active publisher by `pub_id`;
- reject mismatched publisher ids;
- require a bounded safe and unique `adUnits[].code` value;
- unpack each `adUnits[].slot` into `slot_id` and `size_id`;
- require exactly one media type, matching declared dimensions, and a finite
  non-negative request floor;
- validate the site/slot/size/type/floor tuple against cached publisher
  metadata;
- accept browser traffic only for `Web` inventory and SDK traffic only for
  `App` inventory;
- reconstruct site and slot strings for future ACL matching.

These checks run before cookies, Redis mutable state, matching, middleman
fanout, or audit publication. P01 workers also fail closed on old publisher
cache entries that lack type/floor metadata. Follow
[publisher-activation.md](publisher-activation.md) for the compatible cache and
binary rollout order.

No MySQL read is required on the `/pz` request path. Redis mode reads
`pubmap:by-id`; local/static mode reads the derived in-memory by-id lookup from
the loaded `pubmap` snapshot.

## Runtime Adapter

`../pzdesign/cmd/unify` registers `POST /pz` before the Genelet catch-all
handler and handles `OPTIONS /pz` with permissive CORS headers for this endpoint
only:

```text
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

This CORS layer stays credentialless. Preflight does not carry the direct `site`
token body, so `OPTIONS /pz` remains permissive and the validated `POST /pz`
policy is the serving authority.

The adapter builds minimal browser metadata from HTTP headers:

- `User-Agent` -> OpenRTB `device.ua`;
- `RemoteAddr` -> OpenRTB `device.ip` by default;
- `X-Forwarded-For`, then `X-Real-IP` -> OpenRTB `device.ip` only when the
  request peer is listed in `trusted_proxy_cidrs`;
- `Referer` -> OpenRTB `site.ref`;
- `Host` -> OpenRTB `site.name`.

`trusted_proxy_cidrs` accepts explicit proxy IPs or CIDR ranges. Leave it empty
when `cmd/unify` is directly internet-facing.

The cache-derived `siteStr` is used as OpenRTB `site.domain`, and the
cache-derived `slotStr` is used as each impression `tagid`. `adUnits[].code`
is used only as the internal impression/debug id and response ordering aid; it
is never used as publisher, site, slot, or size identity.

M29 publisher tags load `www/js/ads.js`. By default, `ads.js` derives the ad
server endpoint from the script URL, so
`<script src="https://aofei.example/js/ads.js"></script>` posts to
`https://aofei.example/pz`. Integrations can override it explicitly:

```js
pzLoadAds(payload, { endpoint: "https://aofei.example/pz" });
```

The browser helper validates the ordered response, clears no-fill containers,
and reports `filled`, `no-fill`, or `error` through the container dataset.
Filled markup is installed in an opaque-origin `srcdoc` iframe without
`allow-same-origin`; scripts/forms/popups needed by reviewed ad delivery remain
sandbox-scoped. This browser boundary complements, but does not replace, D02
creative and URL validation.

Cross-origin browser calls do not send credentials by default. `ads.js` remains
credentialless and does not add `Access-Control-Allow-Credentials`, so the
browser cookie is best-effort only where the browser accepts and returns it.
When a later personalized browser request returns a valid cookie, the adapter
may use it transiently, but converts it to a domain-separated HMAC pseudonym
before cap or tracking state. A missing cookie does not block contextual
serving; IP and UA may describe the HTTP request, but are removed before
contextual matching and are not combined into a fallback identity.

SDK and in-app requests are represented by `platform:"sdk"` in the v1 `/pz`
contract. They are currently credentialless and cookie-free. This pre-P03
compatibility path does not authenticate a publisher or App and must not be
described as doing so. SDK requests may include
OpenRTB-like `app`, `device`, `user`, and `regs` objects. The adapter synthesizes
OpenRTB `app` and leaves `site` empty. The validated cache-derived site string
is authoritative for app identity; any supplied `app.id`, `app.bundle`, or
`app.domain` must be empty or exactly match it. Body `device` IDs such as `ifa`,
`didmd5`, `didsha1`, `dpidmd5`, and `dpidsha1` feed the normal attribute
identity path only after an accepted personalization grant, with HTTP headers
available transiently for IP, user agent, and language derivation. Contextual
and restricted decisions remove body identifiers, IP, raw UA, precise geo, and
demographics before matching. See
[privacy-data-governance.md](privacy-data-governance.md) for the exact matrix.

M33 reuses the existing middleman runtime for validated `/pz` auctions after
local matching. External disclosure additionally requires
`privacy_contextual_middleman_enabled`. Local no-fill impressions may use
`Fallback` routes when both gates are true. Local filled impressions may use
`Always` routes only when the privacy gate, `middleman_enabled`, and
`middleman_always_enabled` are true; the
marked-up middleman bid must beat the comparable local effective CPM to replace
the local winner. Middleman callback proxy setup remains the materialization
gate: setup failure falls back to a local winner when one exists, otherwise the
impression remains no-fill.

Before any local D01 reservation, D02 validates the selected creative's exact
media type and dimensions, accepted MIME, secure-inventory URLs, landing and
tracker URLs, and requested Native assets. Invalid higher-price creatives are
removed and the auction is re-evaluated. Middleman responses pass corresponding
size/media/callback/secure and contained Banner, VAST Video, or strict Native
validation before they can replace a local winner. See
[auction-pricing-creatives.md](auction-pricing-creatives.md).

Middleman bidders receive independently rebuilt contextual OpenRTB requests,
not the original `/pz` JSON body. Each receives only its assigned impressions,
with identity and uncontrolled extensions removed. SSP request and response
audits wrap privacy-scrubbed `/pz` request and final response payloads.

`/pz` does not write MySQL, refresh caches, add credentialed CORS, accept
wildcard publisher subdomains, or add account roles. P01 additively extends the
publisher cache and `ads.js` renderer while preserving Redis key names, request
and response shapes, and the existing `pub` account boundary.

## Measurement And Audits

Filled SSP markup is rendered by the existing DSP `NewBid` path, so banner,
native, and video markup carry the same signed `/imp` and `/clk` tracking URLs as
ADX-rendered local bids. `/imp` publishes `StatusTrackImp` win/loss records and
`/clk` publishes `StatusTrackClk` records. Those records carry the validated
publisher, site, slot, size, campaign, item, creative, advertiser, and price
data already consumed by `cmd/ledger`; M30 does not require a ledger schema
change.

NATS/log audit payloads distinguish entrypoints:

- ADX `/bid` request and response audit subjects retain OpenRTB shape but remove
  identity, precise device/location data, consent strings, and uncontrolled
  extensions.
- SSP `/pz` request and response audit subjects use a JSON envelope:
  `{"source":"ssp","contract":"pz-v1","privacy_mode":"contextual","privacy_reason":"...","payload":...}`.
- Attribute audit rows remove identity and precise derived facts and include
  additive `source`, `contract`, `privacy_mode`, and `privacy_reason`. ADX rows
  use `source:"adx", contract:"openrtb"`; SSP rows use
  `source:"ssp", contract:"pz-v1"`.

Valid all-no-fill SSP requests still publish request and response audits with the
actual selected response format, but no attribute rows because no impression
served.
Malformed or invalid-token requests return HTTP errors and do not publish SSP
audits.

## Product Boundary

Direct SSP is currently identified by the `/pz` entrypoint and SSP audit
metadata: request and response audit envelopes use `source:"ssp"` and
`contract:"pz-v1"`, while attribute audits add the same fields. ADX/OpenRTB
traffic remains on `/bid/{domain}` with `source:"adx"` and `contract:"openrtb"`.

M34 originally accepted ADR 0001 as a future direction. P02 now implements it:

- keep `pub`, `pub_site`, and `pub_slot` as the publisher and inventory
  ownership boundary;
- add defaulted taxonomy and seller fields to those existing publisher tables;
- normalize site/app identity, integration mode, slot/media intent, and
  quality/source metadata instead of overloading legacy packed fields;
- extend both publisher cache views, privacy-safe attribute audits, R02 report
  facts, and Summer/Genelet pub/site/slot forms additively.

`/pz` plus audit `source:"ssp"` and `contract:"pz-v1"` remains the runtime
entrypoint boundary; inventory taxonomy does not replace or rename it.

Seller and supply-chain claims are server-owned. `/pz` ignores a client source
claim and builds `source.schain` only from an operator-authorized cached seller.
An owned publisher chain is complete; an intermediary/resale chain is
incomplete unless a real upstream owner is recorded, and the service never
invents one. Unauthorized metadata emits no chain. Middleman fanout preserves
only a bounded, standard, validated chain; malformed chains reject that bidder
request, and `source.pchain` plus node extensions are removed.

M35 accepts ADR 0002 as the account/schema boundary decision. Aofei will not add
a separate `ssp` account role or separate SSP-owned inventory schema for the
current direct SSP path. The existing `pub`, `pub_site`, and `pub_slot` model
continues to own publisher accounts, sites, apps, and slots. A later milestone
should reopen the account-boundary decision only for concrete legal/settlement
owner separation, reseller/intermediary ownership, materially different
permissions, incompatible compliance/reporting isolation, or third-party
partner credentials that cannot belong to publisher accounts.

## Publisher UI

The existing `pub` role remains the publisher account boundary. Slot topics
pages expose the server-owned USD CPM floor and preserve the parent site type.
Web inventory produces a browser snippet only; App inventory produces an
SDK/API sample only. The pages keep copy/download affordances.
Publisher forms also expose controlled site/slot taxonomy and proposed public
seller metadata. Operator approval is required before seller identity reaches
an external request, and publisher edits revoke that approval.
Browser sample downloads are named `aofei-slot-<slot_id>.html` and are created
in the browser with a Blob from the rendered sample.

## Milestone Boundary

M29 wires publisher-facing tag generation, copy/download UI, stored slot sizes,
external `ads.js` endpoint resolution, and permissive `/pz` CORS. M30 adds
cookie fallback and audit/reporting semantics. M31 hardens browser
origin/referrer policy and documents the direct SSP source boundary. M32 adds
SDK `app`/`device`/`user` parsing and explicit `"html"`, `"json"`, and
`"openrtb"` response formats. M33 adds middleman fallback/`Always` use for
valid `/pz` auctions without changing `ads.js`, schema, cache shape, or account
roles. M34 adds ADR 0001 and historically deferred richer supply taxonomy
without making a runtime, schema, cache payload, audit payload, ledger, or admin
UI change in that milestone. M35
adds ADR 0002 and closes the separate SSP account/schema question by keeping
the existing `pub` boundary. P01 adds commercial inventory validation,
server-owned floors, Web/App sample separation, isolated browser rendering, and
the staged activation/rollback gate without changing the public `/pz` schema.
P02 implements the additive taxonomy, seller authorization, `source.schain`,
privacy-safe audit, and R02 reporting work while retaining that schema and
ownership boundary.
