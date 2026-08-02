# Status I01 - OpenRTB Partner Interoperability

State: `[+]` Completed

## Goal

Make external OpenRTB integrations bounded, diagnosable, and partner-compatible
without importing Prebid Server or weakening existing safety boundaries.

## Dependencies

- S01 disclosure and outbound sanitation policy.
- O01 partner traffic controls and response-time observability.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Compression | `[+]` | O01 still bounds peer bytes; one gzip layer is decoded under an independent 1 MiB-or-smaller policy and timed in the fixed `compressed` shape. Successful JSON gzip requires accepted negotiation, while `204` remains empty and uncompressed. Partner gzip responses are bounded on encoded and decoded bytes. |
| Partner profiles | `[+]` | Runtime, route compilation, Summer filters, and admin approval require exact OpenRTB 2.5, absolute endpoint syntax without credentials/fragments, 1-5000 ms timeouts, optional bounded seat, environment credential reference, active synthetic reporting IDs, USD CPM/floors, and controlled callback/extensions. |
| Request sanitation | `[+]` | S01 typed sanitation remains authoritative. Each bidder gets only assigned unique imps, contextual minimized data, USD, controlled extensions, and the matching-selected Banner/Video/Native intent; standard inbound multi-format requests remain compatible and are narrowed only for outbound fanout. |
| Response validation | `[+]` | Request ID/version/content type, configured seat, bid identity, imp/media/size, raw price versus forwarded floor, USD, secure callbacks/markup, active synthetic IDs, response size/encoding, and deadline are checked before competition. Invalid partial bids cannot suppress accepted siblings or displace local demand. |
| Diagnostics | `[+]` | Fixed rejection, candidate-stage, outcome, and bidder-latency metrics contain no partner labels. Optional default-off sampled diagnostics expose only a hashed request ID, internal bidder ID, fixed outcome/counts, and elapsed time—never raw traffic, endpoint/callback URLs, credentials, consent, or markup. |
| Compatibility fixtures | `[+]` | `dsp/testdata/openrtb` contains credential-free Web multi-imp, App Native, malformed, timeout, currency, Native, Video, and encoded unsafe-callback fixtures; gzip is generated and bounded in tests. |

## Acceptance Criteria

- Compressed bodies are bounded against decompression abuse.
- One partner cannot observe another partner's metadata or credentials.
- Invalid partner responses cannot win and have an actionable internal reason.
- Existing public response contracts remain compatible unless a separately
  versioned change is approved.

## Verification

- OpenRTB parser/response, gzip limit, middleman isolation, SSRF/callback,
  timeout, rejection taxonomy, benchmark, and full closeout suites.

## Exclusions

- Prebid Server adapters, Prebid Cache, and public debug bodies remain outside
  this product boundary.

## Completion Review

- Deep review preserved OpenRTB multi-format compatibility on the public ADX
  request while narrowing each independently sanitized partner impression to
  the media intent already selected by matching. Audio-only requests remain
  unsupported; no external adapter or uncontrolled extension was introduced.
- Partner profile syntax is checked before DNS/SSRF work, and runtime safehttp
  remains the second endpoint gate. Persisted profiles are revalidated during
  admin approval and active route-cache compilation, preventing an invalid
  legacy row from becoming callable.
- Response review changed floor enforcement to the raw downstream USD CPM
  before margin, rejects late data after both transport and parse work, treats
  malformed encodings/size as invalid protocol rather than a winner, and keeps
  a call outcome `fill` when accepted bids coexist with separately counted
  rejected siblings.
- Request compression owns separate encoded/decoded limits and closes both gzip
  and underlying streams. Response compression honors explicit gzip quality,
  emits `Vary: Accept-Encoding`, flushes write errors back to the auction
  handler, and never compresses `204` or non-JSON errors.
- Safe diagnostics replaced raw partner-call errors/request IDs in operational
  warnings with fixed reasons and hashed identity. Compatibility fixtures use
  IANA example domains/documentation data and are guarded against credential
  text.

## Closeout Verification

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. Pinned
  staticcheck v0.5.1 passed in fresh Go caches for Aofei and pzdesign with its
  documented legacy style exclusions. The documented Aofei race suite and
  pzdesign `cmd/unify`/bidder race suites passed.
- Focused OpenRTB 2.5 HTTP/request, encoded/decompressed gzip bounds,
  negotiation, partner isolation, profile approval, SSRF, timeout, raw-floor,
  seat/currency/request-ID, callback/markup, fixed taxonomy, safe diagnostic,
  and fixture tests passed.
- Both pzdesign template parsers (148 active `.g`, 105 `.e`), public-copy/data
  guards, Aofei documentation/public-data guards, both actionlint checks, and
  all repository diff-hygiene checks passed.
- On the current Haswell test host, accepted traffic-gate requests measured
  about 4.5-5.0 microseconds/op and the 2.2 KiB decoded gzip fixture measured
  about 45-60 microseconds/op over three samples. These are regression evidence,
  not a production capacity claim.
- No schema or cache payload changed, so the disposable schema/cache stack was
  not rebuilt. The live Docker stack, deployed website, and external systems
  were not touched. No commit was created because commit policy is `none`.

## Downstream Reconciliation

- R01/R02 may consume accepted conversion/callback identity and USD CPM facts,
  but cannot treat a win/click as a second charge, log raw partner payloads, or
  bypass the fixed response gate. O02/D03 canary the new rejection, candidate,
  latency, and compressed shapes before expanding partner traffic.
- P02/I03 operator and reporting views may aggregate fixed outcomes but must not
  expose endpoint credentials, raw requests, markup, or callback URLs. I02
  remains demand-gated on a named mobile integration and must preserve the same
  `/pz` body bounds, P01 validation, S01 sanitation, and exact outbound profile.
- No evolution entry is required: I01 completes the already planned external
  interoperability boundary without adopting Prebid, changing account
  ownership, or versioning a public response body.

## Reconciliation From S01

- S01 supplies the mandatory outbound sanitation and privacy gate; I01 owns
  protocol compression, profile-specific compatibility, response validation,
  diagnostics, and fixtures rather than a second privacy parser.
- Partner diagnostics and captures must use S01 audit redaction and bounded
  reason labels. Raw OpenRTB bodies, consent strings, and bidder credentials
  remain prohibited from logs and public responses.

## Reconciliation From S04

- Partner rejection reasons, endpoint labels, response diagnostics, and any
  operator-facing debug view remain ordinary escaped data. Do not expose raw
  bid bodies, creative markup, callback URLs, or partner-supplied HTML through
  a trusted template type or executable preview.
- Compatibility fixtures must include hostile markup and unsafe URL schemes;
  accepted delivery markup is validated as a protocol payload, not blessed as
  control-plane HTML.

## Reconciliation From O01

- Gzip handling must sit beneath O01 compressed-body admission and enforce a
  separate decompressed-size limit. It must populate the fixed `compressed`
  latency shape without adding partner-controlled metric labels.
- Partner profiles must select exact O01 admission policies. Response
  validation and diagnostics must extend the fixed rejection taxonomy so
  invalid partner responses, dependency failures, and overload remain
  distinguishable without exposing a partner identifier.

## Reconciliation From A01

- Partner prices, floors, and auction macros remain USD CPM. Validation must
  reject unsupported currency/invalid price before selection, and accepted
  billable impressions alone enter A01 spend as `CPM / 1000`; diagnostics and
  callbacks must not treat a win or click as a second charge.
- Compatibility fixtures must reconcile local and middleman response CPM to
  advertiser charge, publisher pay, and charge/pay/margin daily facts without
  exposing statements or opaque settlement evidence to the partner.

## Reconciliation From P01

- Partner response price/media validation for direct SSP fanout must use the
  P01 synthesized impression: exact Web/App inventory and dimensions, exactly
  one media intent, and the greater of cache-owned and request USD CPM floors.
  A partner response below that floor or for a mismatched impression cannot
  win.
- Compatibility fixtures should exercise both Web/browser and App/SDK direct
  requests without bypassing P01's pre-auction validation, D01 reservation
  lifecycle, isolated browser materialization, or A01 accounting semantics.

## Reconciliation From D02

- D02 is the mandatory response-validation floor: positive finite USD price,
  exact impression/media/size, secure callbacks and markup, contained Banner,
  well-formed VAST, strict requested Native assets, and active synthetic IDs.
  I01 may add partner-profile restrictions and reason taxonomy but cannot
  weaken or duplicate this gate.
- Partner fixtures must prove encoded unsafe URL schemes, nested markup,
  malformed VAST, unknown active Native fields, wrong/duplicate asset IDs, and
  below-floor or unsupported-currency bids cannot win. Diagnostics expose safe
  reasons, never raw creative bodies.
