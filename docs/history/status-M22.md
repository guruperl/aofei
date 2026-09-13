# Retired milestone M22 - Middleman Reporting And Settlement Views

**Milestone.** M22
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M22.md
**Source specification.** memory-bank/milestone.md#m22---middleman-reporting-and-settlement-views
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M22 - Middleman Reporting And Settlement Views `[+]`

Turn M21 callback facts into advertiser and operator reporting.

Scope:

- Add middleman-specific interval and daily ledger tables.
- Extend `cmd/ledger` to aggregate `WinLoss.Middleman` charge, pay, margin,
  route, bidder, synthetic demand, publisher, win/loss, billable impression,
  click, and callback health facts.
- Add advertiser Summer/Genelet reports that show pay-side middleman spend by
  hour, bidder, and slot.
- Add admin Summer/Genelet reports that show charge, pay, margin, route,
  bidder, publisher, and callback health views.

Acceptance:

- Existing local campaign ledger semantics remain unchanged.
- Advertiser middleman reports are scoped to the logged-in advertiser and do not
  expose charge or margin fields.
- Admin reports expose charge/pay/margin and route dimensions.
- Bid fanout, callback proxying, durable callback retries, and arbitrary markup
  rewriting remain unchanged.
````

## Status record

````markdown
# Status M22 - Middleman Reporting And Settlement Views

## Goal

Turn M21 middleman callback metadata into advertiser pay-side reports and admin
charge/pay/margin settlement views without changing bidder runtime behavior.

## Tasks

- `[+]` Middleman reporting schema.
  - Added `ledger_mid` and `daily_mid` to the active SQL baseline.
  - Preserved historical bidder, route, synthetic demand, and publisher
    dimensions without adding mutable route/bidder foreign keys.

- `[+]` Ledger aggregation.
  - `cmd/ledger` now aggregates `WinLoss.Middleman` metadata into interval and
    daily middleman tables.
  - `StatusTrackImp` drives billable impressions and charge/pay/margin spend.
  - `StatusTrackClk` drives clicks.
  - `StatusWin` and `StatusLoss` drive admin audit counts only.

- `[+]` Advertiser reporting.
  - Added advertiser ledger actions and templates for middleman hourly, bidder,
    and slot reports.
  - Advertiser-facing spend uses pay-side spend.

- `[+]` Admin reporting.
  - Added admin ledger actions and templates for middleman hourly, bidder,
    route, and publisher settlement views.
  - Admin views expose charge, pay, margin, win/loss, billable impression,
    click, and callback health facts.

- `[+]` Documentation and memory.
  - Updated middleman, measurement, operational, production, schema, and memory
    docs for the M22 reporting contract.

## Carry Forward

- `[+]` Optional spread/local route-snapshot ownership is consolidated in D03;
  middleman routes remain Redis-only until D03 records its decision.
- `[+]` Durable callback retry queues moved to M24 and are implemented for
  retryable downstream post-auction callback forwarding failures.
- `[+]` Real invoicing/payment ownership is consolidated in A01/A02; M22
  continues to produce reportable settlement facts only.
- `[X]` Arbitrary downstream markup impression/click rewrite remains a
  non-goal unless a future reporting requirement justifies reopening it.

## Verification

- `[+]` `GOWORK=off go test ./internal/jobs/ledger ./summer/ledger ./summer/registry ./genelet ./cmd/ledger`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./internal/jobs/ledger ./summer/ledger ./cmd/ledger`
- `[+]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load`
- `[+]` `./scripts/aofei-local.sh check-sql`
- `[+]` `./scripts/aofei-local.sh diff-schema`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Middleman reporting schema. | `[+]` | |
| Ledger aggregation. | `[+]` | |
| Advertiser reporting. | `[+]` | |
| Admin reporting. | `[+]` | |
| Documentation and memory. | `[+]` | |
| Optional spread/local route-snapshot ownership is consolidated in D03; | `[+]` | |
| Durable callback retry queues moved to M24 and are implemented for | `[+]` | |
| Real invoicing/payment ownership is consolidated in A01/A02; M22 | `[+]` | |
| Arbitrary downstream markup impression/click rewrite remains a | `[X]` | |
| `GOWORK=off go test ./internal/jobs/ledger ./summer/ledger ./summer/registry ./genelet ./cmd/ledger` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./internal/jobs/ledger ./summer/ledger ./cmd/ledger` | `[+]` | |
| `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load` | `[+]` | |
| `./scripts/aofei-local.sh check-sql` | `[+]` | |
| `./scripts/aofei-local.sh diff-schema` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check && git -C ../pzdesign diff --check` | `[+]` | |

````
