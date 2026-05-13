# Result V16

M32 expanded direct SSP `/pz` from browser-only HTML arrays to explicit
mobile/API response contracts while keeping the existing publisher boundary.

Implemented direction:

- Kept omitted or `responseFormat:"html"` responses as the existing ordered
  `[]string` browser contract.
- Added `responseFormat:"json"` ordered fill/no-fill objects with markup,
  tracker URLs, price/currency, demand IDs, dimensions, and parsed native JSON
  when applicable.
- Added `responseFormat:"openrtb"` standard `BidResponse` output, including
  `200` with empty `seatbid` for all-no-fill.
- Let `platform:"sdk"` requests include OpenRTB-like `app`, `device`, and
  `user` objects; SDK traffic synthesizes `BidRequest.App`, leaves `Site` nil,
  validates app identity against the cached site string, and remains
  cookie-free.
- Preserved `site` and `adUnits[].slot` tokens as authoritative supply
  identity and kept `ads.js`, schema, cache shape, and account roles unchanged.

Deferred direction:

- M33 will decide SSP use of existing middleman fallback and gated `Always`
  middleman behavior.
- M34 will document richer supply taxonomy before schema work.
- M35 will decide whether a separate SSP account/schema boundary is still
  needed.
