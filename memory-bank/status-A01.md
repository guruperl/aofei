# Status A01 - Billing And Manual Settlement Safety

State: `[+]` Completed

## Goal

Define trustworthy marketplace accounting and support auditable manual
invoicing and settlement without collecting unsafe payment credentials.

## Dependencies

- D01 delivery and spend guardrails.
- Coordinate billable event identity with R01 when conversions are introduced.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Accounting contract | `[+]` | `usd-cpm-impression-v2` keeps OpenRTB and tracker values in USD CPM and converts each accepted impression to USD `CPM/1000` at reservation and ledger boundaries. Advertiser, publisher, and middleman charge/pay/margin sources, six-decimal statement rounding, USD-only scope, and migration rules are explicit. |
| Manual invoicing | `[+]` | `accounting.Service` and `cmd/accounting` create idempotent source snapshots, immutable adjustments, maker/checker confirmation, evidence-linked settlement, idempotent linked corrections, reconciliation, and stable CSV exports. |
| Publisher settlement | `[+]` | Publisher statements enforce UTC daily, Monday-Sunday weekly, or calendar-month cadence; Draft/Held release, independent confirmation, manual payout evidence, correction, and publisher-scoped export use the same audited lifecycle. |
| Reconciliation | `[+]` | Unit/integration coverage proves accepted-impression `CPM/1000` ledger and D01 balance facts, interval-to-daily behavior, middleman charge/pay/margin identity dimensions, callback retry non-duplication contracts, statement source discrepancies, adjustments, and payout evidence. Held is the safe input for later S03 fraud decisions; no accounting mutation lowers delivery limits or Redis floors. |
| Sensitive-field retirement | `[+]` | Retired Summer funding modules, routes, forms, balance-credit trigger/view, and advertiser funding-balance action are gone. Inactive baseline compatibility tables retain no card, routing, bank-account, sender, or IP fields and are not accounting authority. |
| Access and audit | `[+]` | The command derives actors from the effective Unix UID, requires distinct maker/checker and checker/settler principals, bounded reasons, and opaque evidence references. Adjustment/audit rows have database-enforced update/delete rejection. |

## Acceptance Criteria

- Charge, pay, and margin facts reconcile by billable identity and currency.
- Manual statements cannot silently mutate delivery facts.
- No active public or authenticated form stores a full payment-card or bank
  credential.
- Corrections and settlement status transitions are authorized and auditable.
- Any price-unit migration updates D01 reservation cost, ledger facts, existing
  stored demand, and statements as one versioned contract.

## Verification

- Ledger/middleman reconciliation, rounding, duplicate/missing events,
  statement/payout authorization, sensitive-field scans, schema reset/load/
  check/diff, pzdesign templates, and full closeout gates.

## Exclusions

- Internally operated payment-card processing remains deferred in
  [docs/defer.md](../docs/defer.md).

## Completion Review

- Deep review corrected the legacy thousand-fold spend-unit mismatch: demand
  and OpenRTB prices remain CPM, while reservations, local advertiser/publisher
  ledgers, and middleman charge/pay/margin now store one-impression USD.
- Review hardened integer-money overflow and minimum-value formatting,
  confirmation source re-reading, non-negative adjustments, request-key
  ownership, zero-date rejection, idempotent creation and correction retries,
  and trusted OS-derived actors. The CLI cannot accept an actor override.
- Review retired the remaining active legacy funding side effects and UI
  routes, removed historical identity/credential columns, and kept
  `adv.balance` and `pay_*` compatibility metadata explicitly outside
  delivery, statement, funding, and settlement authority.
- Statement storage is exact `DECIMAL(20,6)`/integer micro-USD. Historical
  interval/daily source tables remain floating-point compatibility facts and
  are rounded at the statement boundary; changing their public report scan
  contract requires a separate coordinated schema migration.

## Closeout Verification

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. Pinned
  staticcheck v0.5.1 passed for Aofei, for pzdesign with its documented style
  exclusions, and for Genelet with its documented legacy exclusions.
- The Aofei scoped race suite (including accounting), pzdesign `cmd/unify` and
  affected Summer packages, and the full Genelet race suite passed. The final
  focused tests and vet passed after the review fixes.
- A disposable MySQL 8.0.41 database loaded the clean baseline and fixtures;
  ledger and full accounting lifecycle integrations passed, including
  idempotent correction, immutable-row triggers, and sensitive-field checks.
  The resulting inventory was 61 base tables, 0 views, 6 routines, 21
  triggers, with `usd-cpm-impression-v2:USD`; the container was removed.
- All 253 pzdesign templates, public-copy and public-data guards, the Aofei
  documentation/public-data guards, and all three repository diff checks
  passed. No live stack, cache, deployment, or external service was mutated,
  and no commit was created under the goal's no-commit policy.

## Downstream Reconciliation

- P01, D02, I01, R01, O02, D03, R02, P02, S02, I03, S03, and A02 now carry
  the versioned CPM/1000 unit, statement/audit authority, security boundary,
  reconciliation, recovery, and future commercial-contract constraints.
- No evolution entry is required: A01 implements the already planned manual
  accounting boundary and does not add a new product or ownership direction.
