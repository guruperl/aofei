# Auction, Pricing, And Creative Contract

This document is the D02 runtime, migration, and rollout contract for local
demand and middleman responses. It supplements the accounting and delivery
guardrail contracts; it does not introduce a new billing unit.

## Supported Commercial Model

The supported v1 local-demand model is a positive finite USD CPM value on
`adv_item`:

- `cost_type` must be `CPM`;
- `cost` is USD per one thousand impressions;
- empty or `USD` request currency is accepted; another currency no-fills;
- a finite non-negative floor is enforced before a candidate can win;
- the selected CPM remains the OpenRTB and tracking price, while D01/A01 convert
  one billable impression to USD with `CPM / 1000`.

The schema retains the legacy `ROI`, `CPC`, and `CPA` enum values so populated
databases remain readable. They are disabled commercial records, not alternate
auction modes. The UI cannot create or silently save them as CPM, the cache
compiler rejects an active legacy record with its item ID, and the runtime
does not apply historical fixed conversion factors. Converting one requires a
business-reviewed CPM value; there is no generally correct mechanical formula.

## Winner Policy

For each impression, filtering first removes ineligible schedules, stale or
exhausted delivery policy, caps, audiences, incompatible creatives, floors,
currencies, and invalid values. The highest remaining campaign/ad-group USD CPM
wins. Creative weight is used only to rotate creatives that share the same
advertiser, campaign, ad group, and winning CPM.

Equal-price demand is ordered by ascending campaign ID, then ad-group/item ID,
then advertiser ID. This makes cross-demand ties deterministic. Within the
selected demand unit, a positive finite creative weight controls rotation; an
invalid or non-positive weight is rejected during cache compilation. If the
chosen creative fails validation, that creative is removed and the auction is
re-evaluated. If live D01 reservation rejects an exhausted demand unit, all of
that unit's sibling creatives are removed before reselection.

Frequency-cap configuration is also a cache-compilation boundary. A positive
impression or click count requires a positive corresponding period; negative
or out-of-wire-range values are rejected rather than truncated. A standalone
positive impression throttle remains valid without a numbered cap. Runtime
matching fails closed if an invalid cap enters through an older cache.

Creative compatibility is checked before a delivery reservation is acquired.
A successfully materialized local winner keeps its reservation until a signed
impression, loss, displacement by a higher middleman bid, or response failure
finalizes or releases it under the D01 contract. Redis release is idempotent.

## Local Creative Source Contract

All active creatives require a non-empty name, a configured packed size, a
positive finite rotation weight, a supported media type, an absolute HTTP(S)
landing URL without embedded credentials, and safe optional impression/click
tracker URLs.

| Media type | Stored source | MIME and response |
|---|---|---|
| Banner | One absolute HTTP(S) image or HTML-document URL. Raw markup is not accepted. | `image/*` or `text/html`; returned in the existing delivery iframe contract. |
| Video | One absolute HTTP(S) media URL. Raw `<video>` or VAST markup is not accepted as local source. | `video/*` or supported HLS MIME; materialized as VAST 3.0. |
| Native | Version-1 structured JSON containing title and optional description, CTA, icon URL, and main-image URL. | Requested required native assets must be satisfiable; image URLs must resolve to an image MIME from their path. |

The management UI writes this source contract directly. Upload is available
only for Banner images and Video media; it persists the resulting URL and
detected MIME, not executable markup. Native fields are authored separately and
serialized into the versioned JSON source. Management and review pages display
escaped source only and never fetch or execute it.

At bid time the creative must exactly match the impression's media type and
packed dimensions. A request MIME allow-list must contain the creative MIME.
When `imp.secure=1`, the landing, creative, optional compatibility fallback,
quality, and tracker URLs must all use HTTPS. URL schemes other than HTTP(S), credential-bearing URLs,
missing required native assets, and incompatible format/size/MIME values remove
the creative before reservation.

## Middleman Response Boundary

A middleman bid is accepted only when it has the selected impression ID,
approved synthetic identifiers, a positive finite price in USD, non-empty
markup, exact dimensions, a compatible media type, and safe callback/image
URLs. It must also satisfy the synthesized server/request floor.

- Banner markup is contained but may include scripts needed by the approved ad
  container. Top/parent navigation, refresh/base elements, JavaScript-like URL
  schemes, document-domain changes, and sandbox escape tokens are rejected.
- Video markup must be well-formed VAST with a protocol compatible with the
  request. URL-bearing VAST resources must be absolute HTTP(S), and embedded
  HTML/active content remains subject to the container policy.
- Native markup must decode as an OpenRTB native response, reference only
  requested asset IDs, match the requested Native version, supply every
  required asset, use compatible image MIME, and use safe click, fallback,
  tracker, image, and embedded-video URLs. Local Native materialization never
  invents assets for an empty request.
- Secure inventory rejects HTTP URLs or insecure markup in every format.

Only a validated marked-up middleman price can compete with the local winner.
A higher middleman winner releases the local D01 reservation after callback
proxy setup succeeds. Setup failure preserves the local winner.

## Populated-System Audit And Migration

Run the following read-only audit before installing the D02 cache compiler or
HTTP runtime:

```sql
SELECT item_id, campaign_id, cost_type, cost, active
FROM adv_item
WHERE cost_type <> 'CPM'
   OR cost IS NULL
   OR cost <= 0
   OR cost <> cost;

SELECT v.creative_id, v.item_id, v.media_type, v.size_id, v.weight,
       v.active, v.content, m.mime, i.item_click, i.imp_url, i.click_url,
       c.iurl
FROM adv_creative AS v
JOIN adv_item AS i ON i.item_id = v.item_id
JOIN adv_campaign AS c ON c.campaign_id = i.campaign_id
LEFT JOIN adv_media AS m
  ON m.media_id = (
    SELECT m1.media_id
    FROM adv_media AS m1
    WHERE m1.creative_id = v.creative_id
    ORDER BY m1.series, m1.media_id
    LIMIT 1
  )
WHERE v.active = 'Yes'
ORDER BY v.creative_id;
```

Quarantine or disable each invalid active row. For every legacy price, obtain an
approved replacement CPM from the commercial owner and record the decision in
the change ticket. Apply only reviewed item-specific values, for example:

```sql
START TRANSACTION;
UPDATE adv_item
SET cost_type = 'CPM', cost = :reviewed_usd_cpm
WHERE item_id = :item_id
  AND cost_type = :reviewed_legacy_type;
COMMIT;

ALTER TABLE adv_item
  MODIFY cost_type ENUM('ROI','CPM','CPC','CPA') DEFAULT 'CPM';
```

Do not replace `:reviewed_usd_cpm` with a CPC/CPA/ROI multiplier. Normalize each
active creative through the management source fields or a reviewed SQL
migration, then run a dry cache build. A build error is a migration blocker,
not a reason to weaken runtime validation.

## Cache-First Rolling Release And Rollback

The RAdv payload remains version 2. Creative `MediaType` and `MIME` are additive
gob fields under the existing creative payload version: old readers can decode
and ignore them, while D02 readers reject an old entry where they are absent.
Use this order:

1. Freeze advertiser edits and native activation. Back up the database and run
   the audits above.
2. Migrate or disable invalid pricing and creative rows. Change the database
   default to CPM.
3. Install the new `redis-cache` compiler on the singleton cache node. Build a
   complete Redis generation and, for local/spread serving, a complete spread
   generation. Do not publish a partial family.
4. Verify representative Banner, Video, and Native entries plus RAdv winners,
   then roll D02 HTTP nodes gradually and watch creative rejection, no-bid,
   response, reservation, and cache-freshness metrics.
5. After every HTTP node is new, deploy/use the D02 management UI, lift the edit
   freeze, and activate new native creatives.

For runtime rollback, stop the canary and return HTTP nodes to the previous
binary while retaining the new additive cache generation. Do not publish an old
creative generation while any D02 reader remains. Keep newly authored native
creatives inactive until the rollback is resolved because the earlier runtime
does not understand the structured native source contract.

## Observability

`aofei_creative_rejections_total` counts local candidates rejected for creative
load or compatibility failure. Correlate it with no-bid, eCPM error, delivery
reservation/release, middleman invalid-response, and cache freshness metrics.
An increase immediately after rollout normally indicates a stored-data audit
gap; identify and disable the affected record rather than restoring a permissive
renderer.
