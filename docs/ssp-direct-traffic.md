# SSP Direct Publisher Traffic

Direct SSP traffic is the browser/API publisher tag entrypoint for publishers
managed by the existing `pub` role. It is separate from ADX/OpenRTB traffic on
`/bid/{domain}`.

M28 wires runtime serving on `POST /pz`. The handler validates packed direct
SSP tokens against the publisher-by-id cache, converts ad units into internal
OpenRTB impressions, serves local Aofei demand, and returns a JSON array of
HTML strings in request order.

## V1 Browser Contract

Browser tags post JSON to `/pz` and receive a JSON array of HTML strings. The
array order matches the input `adUnits` order. A later no-fill response returns
an empty string at the matching array position.

Request fields:

```json
{
  "id": "optional-request-id",
  "platform": "browser",
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

`adUnits[].code` is only the publisher page DOM element id. It is echoed for UI
and debug use, but it is not trusted as supply identity in the v1 contract.
Older Holiday samples used `code` as the slot token; the parser can recognize
that shape for compatibility tests, but supply validation for the v1 contract
requires `slot`.

Supported `mediaTypes` keys for the v1 contract are `banner`, `video`, and
`native`. Runtime conversion uses the trusted `size_id` from
`adUnits[].slot`, not browser-supplied dimensions, when building the internal
OpenRTB banner, video, or native impression.

Valid requests always return `200 application/json` with one string per input
ad unit. No-fill units are represented by `""` at the matching array position:

```json
["<iframe ...></iframe><img ...>", ""]
```

Malformed JSON returns `400`, oversized bodies return `413`, and invalid direct
tokens, missing slots, unsupported media, unknown publishers, or site/slot
mismatches return `400`.

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
the domain string, and reverse maps from `site_id` and `(site_id, slot_id)` back
to the site and slot strings used by ACL matching.

Local/static mode derives the same by-id lookup in memory from the loaded
`pubmap` snapshot. It does not add a separate spread file family.

Validation checks performed on the `/pz` request path:

- unpack `site` into `pub_id` and `site_id`;
- look up the active publisher by `pub_id`;
- reject mismatched publisher ids;
- unpack each `adUnits[].slot` into `slot_id` and `size_id`;
- validate the site/slot pair against cached publisher metadata;
- reconstruct site and slot strings for future ACL matching.

No MySQL read is required on the `/pz` request path. Redis mode reads
`pubmap:by-id`; local/static mode reads the derived in-memory by-id lookup from
the loaded `pubmap` snapshot.

## Runtime Adapter

`../pzdesign/cmd/unify` registers `POST /pz` before the Genelet catch-all
handler. The adapter builds minimal browser metadata from HTTP headers:

- `User-Agent` -> OpenRTB `device.ua`;
- `X-Forwarded-For`, then `X-Real-IP`, then `RemoteAddr` -> OpenRTB
  `device.ip`;
- `Referer` -> OpenRTB `site.ref`;
- `Host` -> OpenRTB `site.name`.

The cache-derived `siteStr` is used as OpenRTB `site.domain`, and the
cache-derived `slotStr` is used as each impression `tagid`. `adUnits[].code`
is used only as the internal impression/debug id and response ordering aid; it
is never used as publisher, site, slot, or size identity.

M28 intentionally serves local Aofei demand only. It reuses the existing
candidate, frequency-cap, audience, creative rendering, tracker, and audit
paths, but it does not invoke middleman fallback. `/pz` does not write MySQL,
refresh caches, set cookies, change CORS/origin policy, or add separate SSP
reporting semantics.

## Milestone Boundary

M28 wires `POST /pz` and returns ads. Cookie handling, origin/referrer controls,
publisher tag UI/download changes, and direct-SSP-specific reporting semantics
belong to M29 through M31.
