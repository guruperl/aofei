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
the future richer supply taxonomy direction in
[ADR 0001](adr/0001-richer-supply-taxonomy.md). M35 records the account/schema
boundary decision in [ADR 0002](adr/0002-ssp-account-schema-boundary.md): keep
the existing `pub` ownership model. The handler validates packed direct SSP
tokens against the publisher-by-id cache, enforces
the browser page host against the cached site host, converts ad units into
internal OpenRTB impressions, serves local Aofei demand, optionally fans out to
middleman bidders, renders the requested response shape, and identifies SSP
traffic separately in audits.

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
generated slot tags; the browser-supplied dimensions remain advisory runtime
media metadata. Runtime validation rejects a slot token whose `size_id` does not
match the cached configured size for that slot.

`adUnits[].code` is only the publisher page DOM element id. It is echoed for UI
and debug use, but it is not trusted as supply identity in the v1 contract.
Non-empty `code` values must be unique within a request because they become the
OpenRTB impression ids used by downstream bid responses. Omitted or empty
values are allowed and receive generated internal `ssp-<index>` impression ids.
Older Holiday samples used `code` as the slot token; the parser can recognize
that shape for compatibility tests, but supply validation for the v1 contract
requires `slot`.

Supported `mediaTypes` keys for the v1 contract are `banner`, `video`, and
`native`. Runtime conversion uses the trusted `size_id` from
`adUnits[].slot`, not browser-supplied dimensions, when building the internal
OpenRTB banner, video, or native impression.

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
    "ifa": "00000000-0000-0000-0000-000000000000",
    "ua": "ExampleSDK/1.0"
  },
  "user": {
    "id": "example-user-id"
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

The `aofei_pz_uid` cookie is part of the browser contract only. Empty or omitted
`platform` and `platform:"browser"` requests may read an existing valid cookie
or set a best-effort cookie for a later request. `platform:"sdk"` requests do
not read, rotate, set, or propagate this cookie.

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
the site and slot strings used by ACL matching, and cached slot-size metadata.

Local/static mode derives the same by-id lookup in memory from the loaded
`pubmap` snapshot. It does not add a separate spread file family.

Validation checks performed on the `/pz` request path:

- unpack `site` into `pub_id` and `site_id`;
- look up the active publisher by `pub_id`;
- reject mismatched publisher ids;
- reject duplicate non-empty `adUnits[].code` values;
- unpack each `adUnits[].slot` into `slot_id` and `size_id`;
- validate the site/slot/size tuple against cached publisher metadata;
- reconstruct site and slot strings for future ACL matching.

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
- `X-Forwarded-For`, then `X-Real-IP`, then `RemoteAddr` -> OpenRTB
  `device.ip`;
- `Referer` -> OpenRTB `site.ref`;
- `Host` -> OpenRTB `site.name`.

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

Cross-origin browser calls do not send credentials by default. M30 keeps
`ads.js` credentialless and does not add `Access-Control-Allow-Credentials`, so
the browser cookie is best-effort only where the browser accepts and returns it.
When a later browser request returns a valid cookie value, the adapter sets
synthesized OpenRTB `user.id` and `buyeruid` to that value. If the browser
cookie is missing or invalid, the current request leaves `user` empty and
continues through the existing IP+UA fallback in attribute extraction, so cookie
absence does not block serving.

SDK and in-app requests are represented by `platform:"sdk"` in the v1 `/pz`
contract. They are credentialless and cookie-free. SDK requests may include
OpenRTB-like `app`, `device`, and `user` objects. The adapter synthesizes
OpenRTB `app` and leaves `site` empty. The validated cache-derived site string
is authoritative for app identity; any supplied `app.id`, `app.bundle`, or
`app.domain` must be empty or exactly match it. Body `device` IDs such as `ifa`,
`didmd5`, `didsha1`, `dpidmd5`, and `dpidsha1` feed the normal attribute
identity path, with HTTP headers used as fallback for IP, user agent, and
language. Body `user.id` and `buyeruid` are honored for SDK requests only.

M33 reuses the existing middleman runtime for validated `/pz` auctions after
local matching. Local no-fill impressions may use `Fallback` routes when
`middleman_enabled` is true. Local filled impressions may use `Always` routes
only when both `middleman_enabled` and `middleman_always_enabled` are true; the
marked-up middleman bid must beat the comparable local effective CPM to replace
the local winner. Middleman callback proxy setup remains the materialization
gate: setup failure falls back to a local winner when one exists, otherwise the
impression remains no-fill.

Middleman bidders receive the synthesized internal OpenRTB request, not the
original `/pz` JSON body. SSP request and response audits continue to wrap the
original `/pz` request and final SSP response payloads.

`/pz` does not write MySQL, refresh caches, add credentialed CORS, wildcard
publisher subdomains, or change `ads.js`, schema, cache shape, or account
roles.

## Measurement And Audits

Filled SSP markup is rendered by the existing DSP `NewBid` path, so banner,
native, and video markup carry the same signed `/imp` and `/clk` tracking URLs as
ADX-rendered local bids. `/imp` publishes `StatusTrackImp` win/loss records and
`/clk` publishes `StatusTrackClk` records. Those records carry the validated
publisher, site, slot, size, campaign, item, creative, advertiser, and price
data already consumed by `cmd/ledger`; M30 does not require a ledger schema
change.

NATS/log audit payloads distinguish entrypoints:

- ADX `/bid` request and response audit subjects remain raw OpenRTB request and
  response JSON.
- SSP `/pz` request and response audit subjects use a JSON envelope:
  `{"source":"ssp","contract":"pz-v1","payload":...}`.
- Attribute audit rows include additive `source` and `contract` fields. ADX rows
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

M34 does not add a database or cache field for publisher supply source. It
accepts ADR 0001 as the future taxonomy direction:

- keep `pub`, `pub_site`, and `pub_slot` as the publisher and inventory
  ownership boundary;
- add nullable or defaulted taxonomy fields to existing publisher tables in a
  later schema milestone;
- normalize site/app identity, integration mode, slot/media intent, and
  quality/source metadata instead of overloading legacy packed fields;
- extend `pubmap:by-id`, request/response/attribute audits, and
  Summer/Genelet pub/site/slot forms additively only after schema/cache support
  exists.

Until that later implementation, `/pz` plus audit `source:"ssp"` and
`contract:"pz-v1"` remains the current runtime audit boundary.

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
pages keep modal browser and API snippets and add copy/download affordances.
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
roles. M34 adds ADR 0001 for future richer supply taxonomy while making no
runtime, schema, cache payload, audit payload, ledger, or admin UI change. M35
adds ADR 0002 and closes the separate SSP account/schema question by keeping
the existing `pub` boundary.
