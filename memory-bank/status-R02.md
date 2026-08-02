# Status R02 - Marketplace Analytics And Experimentation

State: `[+] Complete`

## Goal

Provide decision-useful advertiser, publisher, and operator analytics with
documented freshness and controlled experimentation.

## Dependencies

- R01 conversion/action attribution.
- A01 charge/pay and settlement definitions.
- O01 metrics, traffic classification, and capacity evidence.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Reporting contract | `[+]` | `reporting/contracts.go` and the R02 guide fix interval/daily/action/callback freshness plus advertiser, publisher, operator, and no-agent visibility. |
| Dimensions | `[+]` | Ledger writes coarse inventory, geo, device, demand, campaign/item/creative, bidder, route, and supply dimensions to `report_delivery`; no raw identity/consent enters the fact. |
| Metrics | `[+]` | Summer renders impressions, clicks, CTR, actions, CVR, spend, revenue, cost, margin, ROI, and ROAS from separately aggregated delivery/action facts with safe zero denominators. |
| Experiment contract | `[+]` | Deterministic versioned assignment, append-only idempotent exposure/outcomes, declared primary/guardrail results, explicit transitions, bounded prune, and exact audited erasure are implemented without serving mutation. |
| Query/storage review | `[+]` | Reproducible MySQL 8.0.41 100k-row evidence met current account/operator thresholds; documented production latency/row/retention triggers gate future summaries/OLAP. |
| Export and UI | `[+]` | Advertiser, publisher, and admin Summer pages plus authenticated JSON chartags state UTC, USD/accounting version, freshness, partial-data semantics, and session-derived scope. |

## Acceptance Criteria

- Every metric has a source, formula, scope, currency, timezone, and freshness
  definition.
- Advertiser and publisher reports cannot access another account's data.
- A/B results identify assignment and exposure and do not silently modify bids
  or budgets.
- Reporting storage changes follow measured evidence rather than a technology
  preference.

## Verification

- Ledger aggregation/reconciliation, authorization, timezone/currency,
  dimension, experiment assignment, export, query benchmark, and full closeout
  suites.

## Reconciliation From O01

- Operator runtime metrics remain on O01's protected, fixed-dimension expvar
  surface. Advertiser, publisher, route, bidder, and experiment dimensions
  belong in authorized reporting storage and must not become metric keys.
- Reporting and experiment capacity tests must record hardware, configuration,
  workload mix, dependencies, and SLOs in O01's baseline format; microbenchmark
  throughput alone is not production p99 evidence.

## Reconciliation From A01

- Spend, revenue, cost, margin, ROI, and ROAS formulas must name the exact A01
  source: USD CPM is converted once at billable-impression ingestion, while
  `daily_*` and statement values are already USD amounts. Reports must not
  multiply or divide those stored amounts a second time.
- Statement/correction/hold state and source discrepancies are authorized
  accounting dimensions, not runtime expvar labels. Exports preserve currency,
  UTC period, six-decimal precision, immutable facts, and account scope.

## Reconciliation From D02

- Auction reports distinguish bid CPM, effective winning CPM, and derived
  post-event CPC/CPA/ROI/ROAS metrics. Derived analytics cannot be presented as
  supported auction cost types or fed back into bids automatically.
- Creative rotation analysis is scoped to the selected advertiser/campaign/item
  demand unit. Experiments may vary reviewed weights inside that unit but cannot
  make lower-CPM demand win or bypass creative validation.

## Reconciliation From R01

- The authoritative action source is the direct, idempotent
  `measurement_action` fact, with `measurement_touch` used only to derive exact
  same-lineage click/view attribution. R02 must preserve advertiser scope,
  event taxonomy, USD purchase-value precision, click precedence, configured
  windows, late/unattributed states, and the distinction between action receipt
  freshness and daily delivery-ledger freshness.
- Action/touch rows expire under `action_retention_hours`. Reports and exports
  must label partial/expired periods rather than presenting retained actions as
  lifetime totals. CVR/ROI/ROAS are derived analytics only: no R02 experiment,
  chart, or export may mutate R01 facts, A01 billing, D01 reservations, or D02
  bids automatically.

## Reconciliation From O02

- Every report and export states source freshness and whether MySQL, NATS/log,
  action reconciliation, callback, interval, or daily input was partial or
  unavailable. Shared-dependency failure must not be rendered as a true zero.
- Query SLA evidence uses the O02 availability/latency measurement format and
  named window. Restore drills preserve report source identities and expiry;
  local benchmarks or one clean-room restore cannot justify a production SLO
  or OLAP migration.

## Reconciliation From D03

- Middleman analytics must distinguish the trigger mode, approved route and
  bidder, raw downstream CPM, returned upstream CPM, advertiser charge,
  downstream pay, and nonnegative margin. Authorized reporting storage—not
  dynamic metric keys or labels—owns the partner dimensions.
- Callback and retry lag can make charge/pay/margin inputs partial even when
  the auction itself completed. Reports must expose that freshness state and
  reconcile against the same D03 route generation and A01 accounting version;
  a dependency or partner failure is not a true zero.

## Reconciliation From S01

- Experiment subjects are per-version SHA-256 hashes over caller-provided
  pseudonyms and a per-experiment salt; raw account, cookie, email, device, and
  event identity are rejected by contract. Retention is 24–9600 hours (default
  2160), outcomes must precede expiry, bounded prune removes expired facts, and
  exact verified subject deletion is transactional and audited without storing
  the hash in the audit.
- Delivery reporting contains only approved aggregate/coarse dimensions.
  Exposure/outcome rows reject updates but can be removed by expiry or exact
  erasure; experiment configuration and non-identifying audit rows remain.

## Verification Results

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. The
  documented Aofei race suite plus R02 reporting/command coverage, pzdesign
  `cmd/unify`/ledger race, and full Genelet race passed.
- Pinned staticcheck v0.5.1 passed for Aofei; pzdesign passed with its documented
  `ST1000`/`ST1003`/`ST1006` exclusions; Genelet passed with its documented
  pre-existing naming/simplification exclusions.
- The clean-room MySQL restore passed with 69 base tables, 6 routines, 25
  triggers, R02 interval/experiment identities, immutable-update/audit guards,
  exact expiry prune, Redis rebuild, and middleman preflight. The real Summer
  experiment and ratio SQL passed against MySQL 8.0.41.
- The reproducible 100,000-row/five-run query baseline on x86-64 with eight
  visible CPUs measured advertiser 103/116 ms, publisher 95/98 ms, and operator
  1051/1125 ms median/max. These remain local query results, not production
  p95/p99 or an availability SLO.
- Both template surfaces (263 templates), public-copy/data guards, Aofei docs,
  SQL guard, actionlint, microbenchmarks, and all repository diff-hygiene checks
  passed. Live schema/cache/deployment mutation was intentionally not performed.

## Review Closeout

- Deep review fixed MySQL's 16-index-column limit without losing device
  dimensions, corrected experiment allocation fan-out, made report scope depend
  on authenticated `_grole`, added missing CVR/ROI/ROAS rendering, executed the
  actual Summer SQL against MySQL, and reconciled pseudonymous retention and
  exact erasure with S01.
- P02, S02, I03, S03, A02, and I02 now carry explicit R02 reconciliation.
  Evolution V22 records the new schema/operator/UI/privacy contract. No commit
  was created because the active goal's commit policy is `none`.
