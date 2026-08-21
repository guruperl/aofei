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
- R02/R03 reporting contracts and O02 backup/restore govern migration evidence.

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
