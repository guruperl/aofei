# Status R01 - Conversion, Action, And Attribution Measurement

State: `[+]` Completed

## Goal

Measure conversions and post-click/post-view actions through signed,
idempotent contracts that can support trustworthy advertiser attribution.

## Dependencies

- D01 delivery identity and spend correctness.
- S01 privacy, consent, retention, and user-data rules.
- A01 accounting definitions for billable versus analytical events.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Event taxonomy | `[+]` | Version 1 defines conversion, positive-USD purchase, download, video completion, and strict lowercase `namespace.name` custom actions. The W8M HMAC token carries full auction/demand/supply lineage and a random JTI but never the D01 reservation token. |
| Ingestion contract | `[+]` | `POST /action` bounds JSON at 64 KiB, rejects unknown/trailing fields, validates the exact-expiry W8M token and exact-body timestamp MAC before MySQL, and returns 204 duplicate success, 409 altered-id conflict, or retryable 503 storage failure. |
| Attribution | `[+]` | Durable same-lineage touches use inclusive configured windows, latest click before latest view, then unattributed. Action occurrence/future skew, maximum age, late state, retry identity, and a repair pass for still-unattributed rows are deterministic and tested. |
| Persistence pipeline | `[+]` | `measurement_touch` and `measurement_action` are direct MySQL facts with concurrency-safe `(adv_id,event_id)` uniqueness. Touch writes are bounded/detached and tracker fail-open; `cmd/action-measurement` reconciles/prunes without reading or mutating reservations, accounting statements, or bid-time MySQL. |
| Privacy lifecycle | `[+]` | Actions accept no raw identity/consent, remain contextual, use an `w8m-action-pseudonym-v1` HMAC domain over random token JTI, and expire under `action_retention_hours`, which must cover maximum action age. Authorized pseudonym export omits token/auction/publisher identity and scoped deletion removes only matching facts/orphan touches. |
| Advertiser reporting | `[+]` | The authenticated advertiser `topicsAdvActions` report shows daily action/attribution/late/purchase totals beside impressions, clicks, and spend, plus escaped type/name breakdown. It explicitly labels actions analytical and non-billable. |

## Acceptance Criteria

- Duplicate valid action callbacks produce one attributed fact.
- Invalid signatures or identities cannot mutate reporting or billing state.
- Attribution rules are deterministic, documented, and covered at boundary
  times and retry conditions.
- No action event changes billing unless A01 explicitly classifies it as
  billable.
- Conversion retries cannot finalize, release, or otherwise reopen a D01
  delivery reservation.

## Verification

- Signature, replay, concurrency, retry, late-event, attribution-window,
  retention, ledger, report-scope, and full closeout suites.

## Completion Review

- Review preserved the action token as a bearer proof of exact W8M delivery
  lineage, not an advertiser authentication or Redis-reservation identity. One
  rendered lineage reuses one random token, token/request HMACs use separate
  domains, and the action pseudonym derives from random JTI rather than a
  person/device identifier.
- Review expanded the lineage hash to include publisher/site/slot as well as
  auction and demand IDs, kept all validation ahead of storage, and confirmed
  duplicate/conflicting concurrent events converge through the database unique
  key without touching Redis or A01 sources.
- Review found that S01's seven-day operational-log default could shorten the
  30-day attribution contract. R01 now uses a distinct configured
  `action_retention_hours=2160`; validation requires it to cover the full
  maximum accepted action age.
- Tracking publication remains primary: a touch write ignores HTTP
  cancellation, is bounded to two seconds, and fails open. Durable action
  insert failure returns 503 so the advertiser can retry the same event id;
  the maintenance job repairs only unattributed facts and never rewrites an
  existing click/view decision.
- Reporting review kept advertiser identity in the Genelet validation boundary
  and every supplied dimension in ordinary escaped table text, with no dynamic
  metric keys, script/chart literals, raw HTML, or executable links.

## Closeout Verification

- Go 1.23.5 full tests and vet passed in Aofei and pzdesign. Pinned staticcheck
  v0.5.1 passed in fresh Go caches for Aofei and pzdesign with its documented
  legacy style exclusions. The documented Aofei race suite plus pzdesign
  `cmd/unify` and ledger report race suites passed.
- Focused token expiry/tamper/reservation isolation, exact-body signature,
  hostile taxonomy, pre-storage rejection, click precedence/window arguments,
  duplicate/conflict/retry, touch fail-open, retention/reconcile/prune/export,
  route registration, advertiser scope, and template escaping tests passed.
- A uniquely named disposable MySQL 8.0.41 container loaded the complete
  63-table baseline. Two concurrent identical inserts produced one action row,
  both measurement tables existed, their checks accepted the reviewed fact,
  and `acct_statement` stayed empty. The container was removed; the live Docker
  stack was not touched.
- Aofei documentation/data guards, pzdesign template/public-copy guards (255
  templates), and both repository diff-hygiene checks passed. Bid/match
  benchmarks passed; the current Haswell host measured the local two-impression
  bid at about 418 microseconds/op and the local two-unit SSP path at about 311
  microseconds/op. These are regression evidence, not production p99 claims.
- No schema/cache was applied to a live environment, no service was restarted,
  and no external system was mutated. No commit was created because commit
  policy is `none`.

## Downstream Reconciliation

- R02 now names `measurement_action` as its authoritative retained analytical
  source, preserves R01 attribution/freshness/expiry semantics, and keeps
  CVR/ROI/ROAS derived and non-adaptive. S03 may consume bounded sequence and
  replay signals but cannot expose tokens/pseudonyms or rewrite facts/billing.
- Completed D02 and `docs/defer.md` record that trustworthy R01 measurement is
  only one prerequisite: CPC/CPA/ROI and automatic/conversion bidding remain
  closed until R02 volume/quality evidence and a new versioned optimization and
  accounting contract exist.
- No evolution entry is required: R01 implements the already planned
  measurement boundary without changing advertiser/publisher ownership,
  public auction response formats, or the CPM accounting direction.

## Reconciliation From S01

- Action callbacks may correlate through signed, time-bounded transaction
  lineage, but published action/attribution facts must not restore the user
  suffix removed from measurement logs or expose cap/reservation keys.
- GPP jurisdiction mappings remain unimplemented; R01 must stay contextual or
  reject unsupported personalized action uses until a reviewed mapping exists.

## Reconciliation From S04

- Action names, attribution dimensions, advertiser-supplied labels, and
  report/chart values remain ordinary escaped data in Summer/Genelet. Signed event
  payloads must not introduce raw HTML, executable callback links, or unsafe
  URL schemes into reports, mail, or diagnostics.
- Add hostile action-name and attribution-label fixtures to the reporting
  tests, including JavaScript chart-string and URL/query contexts.

## Reconciliation From A01

- A01 bills accepted impressions only and keeps clicks/actions analytical.
  R01 must preserve auction/bid/impression/action lineage for reconciliation,
  but action retries cannot create an A01 adjustment, statement mutation, or
  second `CPM / 1000` charge.
- Any future conversion-billable mode needs a new accounting contract version,
  currency/rounding rules, source tables, correction behavior, and populated-
  data migration before it can affect invoices or settlements.

## Reconciliation From D02

- Conversions and actions remain analytical under the D02 CPM-only auction.
  Adding an action taxonomy, attribution window, CPC/CPA label, ROI, or ROAS
  report does not authorize automatic price conversion or conversion-based
  bidding.
- Any future conversion-billable or action-optimized bid mode requires a new
  versioned auction/accounting contract after R01 attribution evidence is
  trustworthy; existing legacy cost rows cannot be repurposed.
