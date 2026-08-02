# Status D02 - Auction, Pricing, And Creative Correctness

State: `[+]` Completed

## Goal

Align auction, pricing, and creative behavior with the W8M marketplace product
contract before advertising CPC, CPA, ROI, native, or highest-price behavior as
fully supported.

## Dependencies

- D01 campaign delivery guardrails.
- R01/R02 before any true conversion-based bidding mode is introduced.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Pricing contract | `[+]` | CPM is the only supported v1 local commercial type. Forms create reviewed positive finite USD CPM, active legacy CPC/CPA/ROI rows are rejected before activation/cache publication, and no historical factor silently converts them. |
| Winner policy | `[+]` | Highest qualified CPM selects the demand unit first; positive relative weights rotate only its creatives. D01 reserves `CPM / 1000` after creative compatibility and releases on materialization, response, or middleman-displacement failure paths. |
| Tie and no-bid rules | `[+]` | Empty/USD currency, finite non-negative floors, deterministic campaign/item/advertiser ties, invalid prices/weights, incompatible creatives, exhausted reservations, and no-eligible-demand behavior are explicit and tested. |
| Native authoring | `[+]` | Summer writes version-1 structured source fields and renders only escaped review data. Runtime maps only requested assets, rejects empty/duplicate/unsupported requirements, and enforces requested image MIME without inventing defaults. |
| Creative validation | `[+]` | Local Banner/Video are URL-only and Native is structured JSON. Local and middleman gates enforce media, size, MIME, HTTPS, URL/callback, requested Native version/assets, VAST resource, and container active-content policy before reservation or replacement. |
| Compatibility | `[+]` | OpenRTB and `/pz` shapes, RAdv v2, and creative payload version remain stable. Media type/MIME are additive fail-closed creative fields; campaign external IDs are no longer misread as fallback URLs; cache-first rollout and migration are documented. |

## Acceptance Criteria

- The public UI and manuals describe only pricing behavior enforced by runtime.
- The highest qualified campaign eCPM wins; creative weight cannot let a lower
  campaign price win.
- Banner, video, and native acceptance/rejection behavior has focused tests.
- Hostile creative markup and unsafe landing/callback schemes cannot escape the
  approved ad-delivery container or become executable in management/review UI.
- Existing stored demand is migrated, rejected, or explicitly grandfathered
  without silent billing reinterpretation.
- Winner reselection, creative failure, response failure, and middleman
  replacement release any D01 reservation exactly once.

## Verification

- Deterministic auction distribution, tie, floor, currency, and creative-format
  test suites plus existing bid-path benchmarks.
- Full Aofei/pzdesign verification and schema/cache compatibility gates when
  their contracts change.

## Exclusions

- Automatic bidding/ML remains deferred; controlled reporting experiments are
  owned by R02.

## Completion Review

- Deep review removed the old cross-demand weighted lottery, made tie order
  deterministic, and kept weight as a relative value without the broken legacy
  activation-time normalization query. Administrator and agent activation now
  validates price and every active creative before changing an item state.
- Creative loading validates and packs the entire selected set before the first
  sink write, preventing an invalid later row from partially publishing a
  selection. Creative weight is checked in both creative and RAdv compilation.
- The database compiler preserves `adv_campaign.foreign_id` as an external
  business identifier instead of treating it as a Native fallback URL. The
  additive fallback cache field remains decodable for rolling compatibility.
- Review tightened Native behavior so local delivery never invents unrequested
  assets and honors image MIME. Middleman Native requires the requested version
  and exact asset shapes; embedded Native VAST receives the same active-content
  checks as ordinary video. URL-bearing VAST nodes require absolute HTTP(S) and
  HTTPS on secure inventory.
- Invalid higher-price creatives are removed before D01 mutation and the
  auction is rerun. Reservation-limit rejection removes the whole winning
  demand unit before reselection. Short/failed OpenRTB and `/pz` response writes
  release materialized local reservations, while a validated higher middleman
  winner releases the displaced local only after callback setup succeeds.

## Closeout Verification

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. Pinned
  staticcheck v0.5.1 passed for Aofei and pzdesign with its documented legacy
  style exclusions. The documented Aofei race suite plus pzdesign
  `cmd/unify`, item, campaign, and creative race coverage passed.
- Focused auction, creative-cache, Banner/Video/Native, middleman hostile-markup,
  activation, response-failure, reservation-reselection, and displacement
  suites passed. Both pzdesign template parsers, public-copy/data guards, Aofei
  documentation/public-data guards, actionlint, and repository diff hygiene
  passed.
- Bid-path benchmarks completed on the current Haswell test host:
  `ServeBidLocalTwoImpressions` about 565 microseconds/op,
  `ServeSSPLocalTwoAdUnits` about 218 microseconds/op,
  `BidResponseMarshal` about 1.6 microseconds/op,
  `TrafficGateAccepted` about 4.8 microseconds/op, and parallel creative
  selection about 190 nanoseconds/op. These are regression evidence, not a
  production capacity claim.
- A uniquely named disposable MySQL/Redis/NATS stack loaded the clean baseline
  and synthetic sample, proved schema parity, compiled two active CPM items and
  two Banner creatives, and passed Redis, spread, and combined cache smoke. All
  disposable containers and volumes were removed; the live stack and website
  were not touched.
- No commit was created because the active goal's commit policy is `none`.

## Downstream Reconciliation

- I01 consumes the exact local/middleman media, size, MIME, Native version/asset,
  VAST, floor, currency, and callback contract. R01/R02 retain CPM auction
  prices and may derive conversion analytics without changing winner policy.
- O02/D03 must canary creative rejection and middleman validation signals before
  expanding routes. P02/I03/I02 must preserve explicit media intent, secure
  rendering, source-only management review, and the cache-first compatibility
  order recorded below.
- No evolution entry is required: D02 implements the already planned auction
  and creative correctness boundary without changing product ownership or the
  public pricing/accounting direction.

## Reconciliation From S04

- S04 deliberately removed execution and fetches from creative management and
  review pages. D02 owns the separate auction-delivery trust decision,
  validation/sanitization policy, secure-inventory behavior, and any future
  isolated preview; it must not reopen a general raw-template helper.

## Reconciliation From A01

- Winner comparison and OpenRTB/tracker payloads stay in USD CPM. The selected
  winner's D01 reservation and all billable ledger facts use one-impression USD
  (`CPM / 1000`) under `usd-cpm-impression-v2`; no pricing refactor may divide
  an already converted limit, floor, or statement amount again.
- Any CPC/CPA/ROI or revenue-share work requires a new versioned billable-event
  and rounding contract plus migration; it cannot silently enter A01's current
  impression-only statements.

## Reconciliation From P01

- Winner and partner-response validation consume the P01 synthesized floor,
  which is already the greater of the cache-owned slot floor and a valid
  request floor. No D02 comparison or compatibility branch may replace it with
  a lower client value.
- Creative media, dimensions, secure markup, landing URL, and callback policy
  must pass before server markup is materialized into the P01 browser sandbox
  or returned to an SDK. The opaque-origin iframe limits impact but is not the
  creative acceptance policy. Preserve additive site-type/floor publisher
  cache decoding and the cache-first rollout contract.

## Reconciliation From R01

- R01 now supplies trustworthy signed/idempotent action facts and deterministic
  click/view attribution, but it deliberately leaves actions analytical. This
  does not reopen D02 or authorize CPC/CPA/ROI bidding: D02 remains positive USD
  CPM with highest-price demand selection and within-unit creative rotation.
- Automatic or conversion-based bidding still requires R02 volume/quality and
  offline evidence plus a new versioned D/A accounting and optimization
  milestone. The reconsideration gate remains in `docs/defer.md`.
