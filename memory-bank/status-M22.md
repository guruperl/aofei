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

- `[ ]` Spread/local snapshots for bidder routes remain deferred; middleman
  routes are still Redis-only runtime cache data.
- `[ ]` Arbitrary downstream markup impression/click rewrite remains out of
  scope unless a future reporting requirement justifies it.
- `[ ]` Durable callback retry queues are not part of M22.
- `[ ]` Real invoicing/payment execution remains out of scope; M22 produces
  reportable settlement facts only.

## Verification

- `[+]` `GOWORK=off go test ./internal/jobs/ledger ./summer/ledger ./summer/registry ./genelet ./cmd/ledger`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./internal/jobs/ledger ./summer/ledger ./cmd/ledger`
- `[+]` `./scripts/aofei-local.sh reset && ./scripts/aofei-local.sh load`
- `[+]` `./scripts/aofei-local.sh check-sql`
- `[+]` `./scripts/aofei-local.sh diff-schema`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check && git -C ../pzdesign diff --check`
