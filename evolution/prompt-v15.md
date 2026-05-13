# Prompt V15

Add direct publisher SSP traffic alongside existing ADX/OpenRTB
`/bid/{domain}` traffic.

The existing `pub` role remains the publisher account and inventory owner. The
v1 browser contract is `POST /pz` with packed direct supply tokens:

- `site` packs `(pub_id, site_id)`;
- `adUnits[].slot` packs `(slot_id, size_id)`;
- `adUnits[].code` is only the DOM element id.

The v1 response is a JSON array of HTML strings in the same order as
`adUnits`. The historical `../winter/src/holiday` SSP flow is reference
material only; current Aofei matching, cache, tracking, and Summer/Genelet UI
remain the implementation base.
