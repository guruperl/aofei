# Status A03 - Exact Monetary Source Migration

State: `[~]` In progress - user-authorized review extension through iteration 15

## Goal

Extend exact USD accounting from A01 statement mutations through every
authoritative price, reservation, ledger, daily, management, and reconciliation
source while preserving auditable compatibility and hosted-payment safety.

## Dependencies

- A01 manual accounting and A02 hosted funding/payout remain the financial
  authority and outage fallback.
- D04 must first correct tracking price type and callback/reservation integrity.
- S05's verified-principal and protected quality-billing boundaries remain
  authoritative. R02/R03 reporting contracts and O02 backup/restore govern
  migration evidence.
- O03 owns renewable maintenance leases, durable file replacement, and
  completeness-validated Redis/static generation publication.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Exact-money contract | `[x]` | `usd-cpm-impression-v3` inventories authoritative and compatibility-only sources. CPM is integer micro-USD/1,000 at six-decimal ingress; one impression is the same integer count of nano-USD, aggregates use checked integer arithmetic, and the statement boundary rounds half away from zero once. Historical floats remain labeled evidence and are never promoted as recovered exact input. |
| Schema and history migration | `[x]` | The baseline and offline `etc/a03_exact_money_migration.sql` use DECIMAL(12,6) CPM and DECIMAL(20,9) amount columns for demand, floors, budgets, balance history, interval/daily, and middleman sources. `money_migration_evidence` preserves database-rendered legacy values without claiming recovered precision; unsupported, invalid, or signed-64-overflow sources are quarantined and stop promotion. Inactive `adv.balance`, `his_payment`, and `pay_payment` floats remain explicitly outside authority. |
| Runtime and cache representation | `[x]` | RAdv v3 writes exact CPM and nano-USD balances while retaining bounded v2/v1/headerless read conversion. Publisher/direct-publisher v3 caches scan and carry exact slot-floor CPM plus matching old-reader float projections; their public serializers and Redis/spread helpers validate the complete v3 shape before bytes or mutations, so unmarked gob remains read-only drain data. Direct SSP validates and maximizes request/configured floors in fixed point, and demand filtering compares exact CPM before OpenRTB projection. New `delivery:v3:*` state uses decimal-string comparison plus Redis `HINCRBY` for atomic signed-64-bit nano-USD reservations; old float keys are untouched for drain. Middleman route caches carry exact four-place percentage/six-place minimum terms, callbacks bind exact charge/pay/margin identity, and authoritative interval/daily aggregation uses checked nano-USD addition before DECIMAL writes. |
| Management and report interfaces | `[x]` | Management item CPM and budget limits are canonical exact JSON strings (six and nine places); numeric money receives `money_string_required` and cannot mutate. SQL scans reject binary float, item/limit responses preserve exact strings, report spend remains account-scoped six-decimal output, and rows distinguish historical v2 from new v3 authority. |
| Statement and database invariants | `[x]` | Database triggers freeze request/party/cadence/period/currency/source/supersession/creator identity, reject statement deletion, and allow draft/Held amount changes only when they equal the immutable adjustment sum. Adjustment/audit and A03 migration evidence remain update/delete immutable; service adjustment, Hold/approval/settlement, and correction replacement paths stay valid. Current baseline: 96 tables, 6 routines, 65 triggers. |
| Account and sensitive-data scope | `[x]` | Statement listing requires an explicit authorized party scope; only the offline operator's explicit `-all-parties` export is global, while Summer passes its typed-principal party. Audit/reference guards reject nine-plus digit account/routing/card groups across common separators, IBAN-like forms, and provider secret prefixes without retaining or echoing the candidate value. |
| Hosted webhook resolution | `[x]` | `account.updated` and `payout.failed` now take a binding-only planner path keyed by the signed event envelope's exact `account`; readiness additionally requires the account object ID to match. Mutable object metadata and operation-object mappings are never consulted, so connected payout failures cannot create cross-operation reconciliation. Immutable event replay, stale-readiness ordering, and retryable `binding_not_found` recovery remain intact. |
| Reconciliation and rollout | `[x]` | The active singleton/report/Summer contract is v3 while historical facts retain v2. The offline migration now captures every affected legacy column and its scale-specific discrepancy. `aofei-exact-money-drill.sh` checksum-restores one synthetic frozen v2 backup into untouched rollback and migrated copies, matches all 23 source-column tuples, proves 26 immutable evidence rows/zero quarantine, and exercises reservation, ledger, statement, and provider idempotency. The runbook fixes legacy-only tolerances, freeze/backup/comparison, O03 cache-first canary, pre-write restore, post-write roll-forward correction, and financial retention. |

## Acceptance Criteria

- Every new authoritative monetary mutation is exact under the published scale,
  rounding, overflow, and CPM-to-impression contract from API through statement.
- Existing history is migrated with explicit discrepancy evidence; precision is
  never fabricated by formatting a previously rounded binary value.
- Mixed-version deployment cannot reinterpret price units, double reserve or
  bill, expose a partial cache/schema generation, or let an old client silently
  write an inexact value.
- Direct SQL cannot rewrite protected statement identity/amount fields while
  every documented A01/A02 lifecycle and correction remains usable.
- Connected-account webhook metadata cannot attach reconciliation evidence to
  an unrelated operation.

## Verification

- Boundary/rounding/overflow/property tests, concurrent reservation and ledger
  reconciliation, exact API/CSV compatibility, webhook replay/reordering, and
  statement lifecycle/trigger tests.
- Disposable populated-schema migration with clean reset/load/check/diff,
  backup/restore and cache rebuild, old/new comparison evidence, and rollback.
- Full Aofei/pzdesign/Genelet tests, vet, pinned staticcheck, scoped race,
  documentation/template/public-data guards, benchmarks, and diff hygiene.

## Deep Review Gate

- Iteration 1: `[x]` Eight blocking findings were confirmed and resolved:
  1. **P1 - local billing price identity:** signed impression/click tracker URLs
     serialize the `float32` compatibility projection instead of the exact CPM,
     and callback parsing reconstructs authority from that float text. Larger
     six-place prices can therefore disagree with the reservation and ledger.
  2. **P2 - reservation release floors:** release subtracts spend without
     clamping it to `floor_spend_nano`; a newer reconciled database floor or an
     evicted state key can be reopened by a late release.
  3. **P2 - exact-value fail closed:** an invalid nonzero v3 RAdv CPM can fall
     back to the legacy float, and middleman ledger conversion ignores CPM
     range errors, permitting malformed exact facts to be reinterpreted or
     reduced to zero.
  4. **P2 - migration preflight/evidence:** the offline SQL does not
     executable-gate the v2 contract and expected source types before durable
     mutation, does not quarantine values outside Go's signed nano range, and
     labels already-exact route minimums as legacy float recovery.
  5. **P2 - Summer monetary writes:** item CPM, slot floor, balance limits, and
     route minimum CPM still parse/format through `float64`, so direct portal
     writes can silently round over-scale values before MySQL stores them.
  6. **P2 - report version identity:** marketplace queries can aggregate mixed
     v2/v3 rows while the page contract labels every result v3 and detail rows
     omit their persisted accounting version.
  7. **P2 - middleman price authority:** downstream CPM, minimum margin, markup,
     callback state, and charge/pay reconciliation still use binary floating
     point before exact fields are published, without a declared derived-price
     rounding boundary or overflow rejection.
  8. **P2 - configured supply-floor authority:** publisher cache generations
     scan and serialize the exact database floor only as `float64`, and local
     matching compares exact demand CPM through a `float32` projection.
  Resolution: signed tracking and callbacks now bind the exact CPM; reservation
  release preserves reconciled floors; malformed v3 values fail closed;
  migration preflight, range quarantine, and already-exact evidence are
  executable; Summer writers parse exact strings; reports preserve mixed
  version identity; middleman pricing is fixed-point end to end; and publisher
  floors remain exact from database/cache through direct SSP and demand
  comparison.
- Iteration 2: `[x]` Four blocking findings and one lower-severity evidence
  defect were confirmed after the full
  iteration-1 fix set passed automated verification:
  1. **P1 - announced versus billable local price:** exact candidate ordering
     still returns the selected CPM through `float32`; the OpenRTB bid can
     therefore announce a lower compatibility projection while reservation,
     tracker, and ledger charge the exact six-place CPM.
  2. **P2 - win/loss version and identity:** win/loss facts have no top-level
     accounting marker, so legacy drain records are inserted under the v3
     report default, while a marked-v3 middleman fact can still reconstruct
     missing exact charge/pay from compatibility floats without verifying the
     RAdv charge identity or USD currency.
  3. **P2 - report CPM aggregates:** `downstream_cpm_sum` and
     `returned_cpm_sum` still add float projections (including local
     `float32`) before insertion into exact DECIMAL report columns.
  4. **P2 - middleman reconciliation rounding:** reconciliation rounds charge,
     pay, and margin independently to statement scale before checking
     `charge-pay=margin`, which can report a false discrepancy even when the
     underlying nano-USD aggregates are exact.
  5. **P3 - absent floor evidence label:** a nullable legacy publisher floor is
     preserved as absent but labeled `LegacyRenderedHalfAway`, falsely implying
     that a binary value was converted.
  Resolution: local selection now carries exact CPM through the OpenRTB float64
  response boundary; top-level win/loss markers separate v2 drain and v3 facts
  and v3 middleman identity fails closed; report CPM sums use checked fixed-
  point totals; middleman reconciliation compares nine-place aggregates before
  statement projection; and absent floors receive truthful evidence labels.
- Iteration 3: `[x]` One blocking finding was confirmed after the complete
  iteration-2 verification matrix passed:
  1. **P2 - constructor/drain version provenance:** exported `NewWinLoss`
     labels a float-only compatibility RAdv as v3, allowing a new signed
     tracker to promote an already-rounded value into apparent exact authority;
     conversely, an explicitly labeled v2 log fact can carry `CostCPM` and the
     ledger consumes that exact field instead of rejecting the mixed contract.
  Resolution: `NewWinLoss` derives v3 only from a present authoritative exact
  field and keeps float-only compatibility callers on v2; ledger normalization
  rejects explicit v2 facts that mix in an exact demand CPM.
- Iteration 4: `[x]` One blocking finding was confirmed during the whole-
  milestone review after the iteration-3 fix:
  1. **P2 - v2 cache provenance loss:** the version-2 RAdv decoder materializes
     its float-derived CPM into `CostCPM`, and `billableRAdv` always populates
     that field, so an auction served from a draining v2 cache is relabeled v3
     despite originating from the compatibility float contract.
  Resolution: v2 decode validates but does not populate the v3 exact field;
  billable RAdv construction retains that absence and emitted facts remain v2.
- Iteration 5: `[x]` One blocking finding was confirmed during the next full
  review:
  1. **P2 - compatibility cache promotion on write:** current RAdv packing calls
     the read-compatibility adapter, so a decoded v1/v2 float record can still
     be serialized into a v3 cache payload and falsely acquire exact-source
     authority through public cache write helpers.
  Resolution: v3 RAdv packing now requires a present, valid exact CPM and
  refuses every float-only compatibility record; v2 decoding remains read-only.
- Iteration 6: `[x]` One blocking finding was confirmed during the next full
  review:
  1. **P2 - current middleman cache downgrade:** a version-2 route cache can be
     written and decoded with entries that omit the v3 accounting marker, and
     unknown nonempty entry markers also fall through to legacy float
     conversion. The current cache can therefore silently lose exact margin
     provenance despite its version and completeness contract.
  Resolution: current route-cache write, decode, and activation validation
  require v3 on every entry; legacy entries accept only empty or v3 provenance,
  and all other markers fail closed.
- Iteration 7: `[x]` One blocking finding was confirmed during the next full
  review:
  1. **P2 - publisher floor provenance:** publisher/direct-publisher floor
     readers consult an exact field before resolving the cache accounting
     marker and treat every unknown marker as legacy. A legacy or unknown cache
     can therefore supply `SlotFloorCPMs`, while direct SSP validation then
     hardcodes the resulting unit as v3.
  Resolution: publisher and direct-publisher readers now resolve a recognized
  accounting marker before selecting the corresponding floor field; current
  commercial caches require v3 provenance, and SSP units retain the resolved
  contract instead of relabeling legacy data.
- Iteration 8: `[x]` One blocking finding was confirmed during the next full
  review:
  1. **P2 - callback accounting downgrade:** middleman callback reconciliation
     sends every non-v3 accounting marker, including an unknown nonempty
     version, through the legacy float conversion branch. Corrupted or future
     callback state can therefore be billed under a weaker contract instead of
     failing closed.
  Resolution: callback reconciliation now admits only v3, explicit v2, or an
  unmarked pre-A03 context; every unknown marker is rejected before prices are
  derived or published.
- Iteration 9: `[x]` One blocking finding was confirmed during the next full
  review:
  1. **P2 - unmarked tracker promotion:** billable callbacks from pre-A03
     signed URLs have no accounting marker but their float-rendered price often
     parses as a six-place decimal. Tracking ingress stores that value in the
     exact CPM field, and ledger normalization infers v3 from the field, falsely
     promoting compatibility evidence into exact authority.
  Resolution: tracking ingress populates the exact CPM only for an explicit v3
  callback; unmarked and v2 callbacks retain a validated float projection, and
  ledger normalization rejects an unmarked record that mixes in exact fields.
- Iteration 10: `[x]` One blocking finding reached the original bounded review
  limit on 2026-08-25. The user explicitly authorized this continuation and an
  extension through iterations 11-15; A03 returned to in-progress and
  downstream reconciliation remains paused until a clean extended pass.
  1. **P2 - publisher compatibility generation rewrite:** the exported
     publisher serializers and their direct Redis/spread helpers can write a
     newly encoded active `Pub`/`DirectPub` with an empty accounting marker and
     float-only floors. The supported generation jobs validate v3 before these
     calls, but the public writer contract can bypass that guard and repopulate
     live publisher keys with a newly created compatibility generation instead
     of reserving unmarked gob data for read-only drain.
  Resolution: publisher and direct-publisher packing now shares a write-v3
  validator requiring exact markers, valid per-slot exact floors, complete and
  canonical float projection parity, and direct/embedded agreement. Packing
  fails before bytes; map Redis/spread helpers preflight every active publisher;
  and the single-publisher Redis helper builds both payload shapes before its
  first command. Legacy and unknown-version gob objects remain decodable but
  cannot be reserialized.
- Extended review authorization (2026-08-25): iterations 11-15 are available
  for full-milestone review/fix cycles. Stop at the first clean pass; if
  iteration 15 still has a P1/P2-or-higher finding, return A03 to blocked and do
  not reconcile downstream work.
- Iteration 11: `[x]` Three blocking findings and one lower-severity contract
  defect were confirmed during the full extended review:
  1. **P2 - exact constructor downgrade:** exported `NewDSPForImp` always
     reparses its `float32` bid-price argument. When the supplied RAdv already
     carries a v3 exact CPM, `billableRAdv` replaces that authority with the
     rounded float projection and emits the replacement as v3.
  2. **P2 - RAdv partial serialization:** `RAdvs.Pack` and `PackIO` write the v3
     cache header before validating every candidate's exact CPM. A rejected
     compatibility or malformed record can therefore return nonempty bytes or
     mutate an output writer with a corrupt partial current generation.
  3. **P2 - middleman route write/projection downgrade:** the exported route
     cache writer still accepts a legacy unmarked generation for
     reserialization, while marked entries validate exact and compatibility
     margin terms independently rather than requiring pointer/value parity.
     New and old binaries can therefore use different markup from the same
     newly published cache.
  4. **P3 - management decimal canonicality:** management writes reject JSON
     numbers but accept and normalize noncanonical exact strings even though
     the public OpenAPI request schema requires exactly six-place CPM and
     nine-place spend strings.
  Resolution: the exported compatibility constructor now treats a
  present exact CPM as authority, uses the float only as a projection check,
  and fails bid materialization on disagreement without downgrading its audit
  provenance. RAdv packing now constructs and validates the complete v3 wire
  slice before writing its header, so rejected input returns no bytes and
  leaves `PackIO` output unchanged. Middleman route serialization now writes
  current generations only, validates every exact/compatibility group and route
  term (including pointer and negative-zero parity), and checks the derived
  old-reader generation before its atomic dual-key write; unmarked legacy data
  remains readable only. Management validation now requires the submitted
  string itself to equal the canonical six- or nine-place representation rather
  than normalizing a looser value. Each blocking fix was committed separately;
  iteration 12 is the next whole-milestone review.

## Exclusions

- Internally operated payment-card storage/processing remains deferred.
- The migration does not convert CPC, CPA, ROI, or automatic bidding into
  supported commercial models.

## Reconciliation From S05

- Summer money, statement, and hosted-payment actions must continue consuming
  Genelet's typed exact component/action/permission/resource principal and
  server-derived recent-MFA deadline. Exact-decimal request fields, account
  numbers, compatibility `_g*` values, provider metadata, and API credentials
  cannot become actor or scope evidence.
- The migration starts from the S05 clean baseline of 95 tables, 6 routines,
  and 57 triggers. Any exact-column replacement must update and prove the
  `quality_billing_protected_update` contract in the same versioned migration:
  decision/statement/digest/disposition/recommender evidence stays immutable,
  independent review remains mandatory, and valid Hold application continues
  to compose with A01/A02 state transitions.
- Restricted retention/health commands retain their exact effective-Unix
  principals and cannot move money, reconcile, call a provider, or acquire
  wildcard/recent-MFA authority through retry or compatibility paths.
- New provider/webhook resolution must not introduce a caller-selected URL or
  injected transport that bypasses S05 address, DNS-rebinding, TLS, redirect,
  and credential-forwarding policy.

## Reconciliation From O03

- Exact-money cache migration must publish versioned shadows through O03's
  completeness-marker script. Missing, evicted, partially recreated, or mixed-
  version shadows preserve the prior complete live generation; no compatibility
  writer may reset or repopulate live RAdv/price families in place.
- Migration, comparison, ledger, and retention commands honor the renewable
  lease-owned context while database uniqueness/idempotency remains the durable
  correctness boundary. Filesystem evidence uses the shared durable writer,
  restricted modes, atomic replacement, and identifier-free fixed-cardinality
  failure reporting.

## Reconciliation From R03

- The current A03 starting schema is 95 tables, 6 routines, and 61 triggers.
  Exact-money migrations must preserve the complete experiment-version/state
  guard, Draft-only serialized variant insertion, variant/outcome immutability,
  and metric-value CHECK. Update schema counts and recovery evidence for A03's
  own trigger changes without rewriting legacy v1 or current v2 assignments.
- `report_delivery` and experiment outcomes remain derived analytical facts,
  even when represented as exact decimals. A03 must derive every authoritative
  monetary value from its inventoried demand/reservation/ledger/accounting
  source; it cannot promote an experiment outcome, aggregate export, ratio, or
  formatted historical float into price, balance, statement, or settlement
  authority.
- Exact report/API work preserves R03's registry value domains: counts are
  nonnegative integers, repeated-event CTR/CVR may exceed one, ROI is at least
  -1, and money/CPM/ROAS are nonnegative. NaN/Inf, negative zero, overflow, and
  noncanonical decimal input remain rejected, while account scope and Summer's
  aggregate-only experiment export remain unchanged.
- Populated migration and recovery fixtures create experiments as Draft, add a
  complete 2-20 variant/10,000-basis-point allocation, then transition to
  Running with audit evidence. A03 rollback must not disable or bypass these
  guards to load fixtures or reconcile monetary history.
