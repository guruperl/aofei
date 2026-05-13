# Result V9

Resulting direction after M22:

- `cmd/ledger` remains the singleton reporting aggregation job on the log
  aggregation node.
- Legacy ledger tables keep existing charge-side delivery semantics.
- Middleman callback metadata is aggregated into additive `ledger_mid` and
  `daily_mid` tables.
- Advertiser middleman reports use pay-side spend scoped to the logged-in
  advertiser.
- Admin middleman reports expose charge spend, pay spend, margin, route and
  bidder dimensions, publisher breakdowns, and callback forwarding health.
- Bidder runtime, callback proxying, arbitrary markup rewriting, durable retry
  queues, and real invoicing remain outside M22.
