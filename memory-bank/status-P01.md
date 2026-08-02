# Status P01 - Direct SSP Commercial Readiness And Activation

State: `[+]` Completed

## Goal

Make the existing direct publisher `/pz` path commercially safe and prove a
repeatable publisher onboarding, activation, and rollback workflow.

## Dependencies

- Technical staging can start with the current runtime.
- Revenue launch requires D01, S01, A01, and O01 acceptance.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Server-owned floors | `[+]` | Existing `pub_slot.bidfloor` is validated and carried through `pubmap`/`pubmap:by-id`; `/pz` synthesizes a USD impression floor from the greater of the finite non-negative request and configured values. Publisher forms create/edit/display the normalized six-decimal floor. |
| Inventory readiness | `[+]` | Cache publication validates active commercial publisher/site/slot identity, exact Web/App type, safe Web hostname/App identity, unique mappings, packed size, and finite non-negative floor. Static request shape is rejected before cache access; cache-owned platform/type/size/media policy is rejected before auction, mutable state, middleman, or audits. |
| Cache publication | `[+]` | `cmd/redis-cache -validate-publishers` reads MySQL only and emits a deterministic credential-free packed-token manifest. Full jobs validate commercial metadata before shadow publication; Redis read output includes both publisher views, and local mode derives the same additive fields without a new family or reload mechanism. |
| Browser integration | `[+]` | Web inventory alone generates `ads.js` tags. Source/Node tests prove endpoint selection, ordered fill/no-fill/error states, no host-page `innerHTML`, and opaque-origin sandboxed `srcdoc` materialization; `/pz` tests prove origin/referrer policy, fill/no-fill, signed impression/click, audits, and ledger facts. |
| SDK-style integration | `[+]` | App inventory alone generates contextual SDK/API samples. Contract tests prove JSON/OpenRTB fill and no-fill, exact app identity, platform/type rejection, device/user privacy handling, cookie isolation, tracker behavior, and middleman response compatibility. |
| Rollout operations | `[+]` | The activation runbook defines DB readiness, additive cache-first rolling order, full-family inspection, D01 freshness/reservation signals, O01 canary evidence, A01 reconciliation, disablement, safe binary/cache rollback, privacy approval, troubleshooting, and prohibited evidence. |

## Acceptance Criteria

- A client cannot reduce an operator-configured slot floor.
- Approved browser and SDK-style integrations pass end to end; invalid tokens,
  origins, media, and identities fail before auction side effects.
- Cache publication and inventory disablement have documented bounded effect.
- Advertiser charge and publisher pay facts reconcile before paid launch.
- Response/render/middleman replacement and tracking outcomes preserve D01's
  reserve, release, and finalization lifecycle for browser and SDK fills.

## Verification

- Direct SSP parser, policy, cache, media, response-format, tracking, and
  middleman interaction suites.
- Browser sample/manual checks and full Aofei/pzdesign closeout gates.

## Reconciliation From S01

- Browser and SDK acceptance must exercise the S01 decision matrix: no signal
  is contextual, cookies require an approved personalization grant, SDKs never
  use the browser cookie, exact coordinates are always removed, and no IP/UA
  fallback identity is permitted.
- Production publisher onboarding must verify approved CMP/signal propagation
  and the public contextual samples; it must not copy fixture identifiers or
  invent consent evidence.

## Reconciliation From S04

- Browser acceptance must keep publisher setup, generated-tag, fill/no-fill,
  and troubleshooting values under contextual escaping. Control-plane pages
  may show returned or stored creative only as escaped source; actual `/pz`
  rendering must occur in the integration's explicitly isolated ad container
  and pass D02 creative validation.
- Browser samples may load only reviewed local W8M assets. Hostile creative,
  site, slot, and query values must not create script/event/unsafe-URL behavior
  in the publisher workspace or sample host page.

## Reconciliation From O01

- Browser and SDK canaries must configure the exact `ssp` admission policy and
  verify bounded-body, rate, concurrency, timeout, and dependency behavior.
  O01 limits are per process and are not a global publisher quota.
- Before revenue traffic, staging evidence must exercise browser, SDK, Redis,
  local-fill, no-fill, and middleman profiles in O01's capacity-baseline format
  and validate the fixed rejection and latency shapes.

## Reconciliation From A01

- Staging must prove that OpenRTB and `/pz` response prices remain USD CPM
  while each accepted impression reserves and ledgers `CPM / 1000` USD under
  `usd-cpm-impression-v2`. Paid launch cannot mix pre-v2 writers/readers or
  proceed until the documented populated-database conversion is complete.
- Publisher acceptance must reconcile `daily_pub` to an A01 publisher
  statement, including six-decimal source rounding, immutable adjustments,
  confirmation, correction, and manual settlement evidence. Integration pages
  and support channels must never collect full card or bank credentials.

## Completion Review

- P01 reuses the existing `pub`, `pub_site`, `pub_slot`, `/pz`, Redis key, and
  public response boundaries. The Gob payload adds site type and floor fields;
  older workers decode them, while P01 workers deliberately fail closed on a
  pre-P01 publisher entry. This yields a cache-first rolling order without a
  schema or public API migration.
- Deep review made the parent site type database-authoritative in the publisher
  workspace, preserved it through CRUD links, normalized both string and
  numeric DB floor values, rejected invalid/duplicate commercial identities,
  validated Web hostnames, aligned NULL floor compatibility to zero, and
  propagated manifest writer errors.
- Static platform, code, floor, and media validation runs before publisher
  cache lookup. Cache-dependent approval/type/size/dimension validation still
  happens before cookies, matching, delivery reservation, middleman fanout, or
  audit publication. The configured floor can only increase the request floor.
- Browser isolation is an explicit delivery boundary: filled markup enters an
  opaque-origin sandboxed iframe and no-fill/error clears the host container.
  D02 still owns creative/URL acceptance before materialization; P01 does not
  declare arbitrary stored markup safe.

## Closeout Verification

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. Pinned
  staticcheck v0.5.1 passed for Aofei and for the sibling repositories with
  their documented exclusions. The documented Aofei race suite (including
  `acl`/cache), pzdesign `cmd/unify`/Summer/slot race suite, and full Genelet
  race suite passed.
- A disposable MySQL 8.0.41 database loaded the clean baseline and synthetic
  fixtures. The read-only readiness command identified one App and one Web
  site with exact packed tokens. A second disposable MySQL/Redis stack built
  and read one complete atomic generation containing `pubmap`,
  `pubmap:by-id`, `slot:4194368`, `audience`, `creative`, and both route keys;
  decoded publisher views retained exact type/size/floor metadata. All
  temporary containers were removed.
- All 253 pzdesign templates, contextual publisher-slot rendering, Node browser
  fill/no-fill behavior, public-copy/data guards, Aofei documentation/public-
  data guards, workflow actionlint, and repository diff hygiene passed. The
  benchmark gate recorded `BenchmarkServeSSPLocalTwoAdUnits` at about 190
  microseconds/op on the current Haswell test host; this is regression evidence,
  not a production capacity claim.
- The live Docker stack, database, Redis cache, website deployment, and external
  services were not touched. No commit was created under the goal's no-commit
  policy.

## Downstream Reconciliation

- D02 now preserves the synthesized server floor and treats the browser sandbox
  as containment rather than creative approval. I01 consumes exact P01
  platform/media/floor semantics for partner response validation. I02 must use
  App inventory, remain cookie/fallback-free, and prove the same tracker,
  reservation, accounting, and rendering ownership.
- No evolution entry is required: P01 implements the already planned direct
  SSP commercial-readiness boundary without changing product direction,
  ownership, or the public contract.
