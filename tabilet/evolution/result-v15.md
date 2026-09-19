# Result V15

M27 established the direct SSP contract and cache lookup foundation without
wiring runtime `/pz` serving.

Implemented direction:

- Added `docs/ssp-direct-traffic.md` for the v1 browser contract and milestone
  boundary.
- Added historical base32 direct token pack/unpack helpers.
- Added `dsp.ParseSSPRequest` and supply validation over cached publisher
  metadata.
- Added additive Redis cache hash `pubmap:by-id`, derived from `pubmap`, with
  reverse site/slot metadata.
- Kept local/static mode derived from existing `pubmap` snapshots, with no new
  spread directory.
- Updated cache smoke coverage to require `pubmap:by-id`.

Deferred direction:

- M28 wires `POST /pz` runtime serving.
- M29 updates publisher tag UI/download behavior in `../pzdesign`.
- M30 adds direct SSP measurement, cookie, and reporting semantics.
- M31 adds hardening and final product-boundary decisions.
