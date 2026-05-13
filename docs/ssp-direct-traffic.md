# SSP Direct Publisher Traffic

Direct SSP traffic is the planned browser/API publisher tag entrypoint for
publishers managed by the existing `pub` role. It is separate from ADX/OpenRTB
traffic on `/bid/{domain}`.

M27 defines the request contract and cache lookup foundation only. Runtime
serving on `/pz` is intentionally not active until the later SSP runtime
milestone.

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
`native`. The current foundation parses these fields; runtime conversion into
OpenRTB impressions is a later milestone.

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

Validation checks performed by the foundation:

- unpack `site` into `pub_id` and `site_id`;
- look up the active publisher by `pub_id`;
- reject mismatched publisher ids;
- unpack each `adUnits[].slot` into `slot_id` and `size_id`;
- validate the site/slot pair against cached publisher metadata;
- reconstruct site and slot strings for future ACL matching.

No MySQL read is required on the future `/pz` request path.

## Milestone Boundary

M27 does not wire `POST /pz`, return ads, set cookies, publish SSP audits, or
change publisher UI snippets. Those behaviors belong to M28 through M31.

The stable v1 browser response shape for those later milestones remains:

```json
["<div>renderable ad html</div>", ""]
```
