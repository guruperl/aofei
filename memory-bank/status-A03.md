# Status A03 - Exact Monetary Source Migration

State: `[ ]` Planned

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
| Exact-money contract | `[ ]` | Inventory authoritative versus compatibility-only monetary columns and define one versioned USD CPM, per-impression, rounding, overflow, minimum-unit, aggregation, and display contract before schema or API changes. Do not claim that existing floating-point history can be recovered exactly. |
| Schema and history migration | `[ ]` | Migrate authoritative demand price, balance, interval, daily, middleman, ledger, and other live sources to reviewed DECIMAL/integer representations. Preserve original values and reconciliation evidence, quarantine ambiguous rows, and keep inactive legacy payment metadata outside accounting authority. |
| Runtime and cache representation | `[ ]` | Remove binary floating-point from monetary mutations and reservations, version affected RAdv/cache/Redis state, and provide cache-first dual-read/dual-write or drain/rebuild sequencing with overflow-safe atomic budget operations. |
| Management and report interfaces | `[ ]` | Introduce exact decimal-string or integer-minor-unit request/response handling with explicit compatibility and deprecation for existing numeric clients. Validate scale/range before writes and preserve account-scoped reports and CSV formatting. |
| Statement and database invariants | `[ ]` | Prevent amount, party, cadence, period, currency, source, adjustment, total, and supersession identity from direct mutation after their allowed draft construction. Keep status transitions, corrections, Holds, approvals, settlements, and immutable replacement/audit workflows functional. |
| Account and sensitive-data scope | `[ ]` | Require an explicit authorized scope for statement listing, strengthen payment-material detection for plausible account-number forms without collecting the value, and prove no compatibility or diagnostic surface broadens party access. |
| Hosted webhook resolution | `[ ]` | Resolve connected-account readiness and payout-failure events from trusted event account/object bindings rather than mutable object metadata. Prevent spurious cross-operation reconciliation while preserving replay, reordering, and dependency-pending recovery. |
| Reconciliation and rollout | `[ ]` | Compare old/new calculations over a frozen backup, define tolerances only for unrecoverable historical float input, prove no double reservation/ledger/statement/provider movement, and document backup, canary, rollback, correction, and audit retention. |

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
