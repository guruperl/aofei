# Result V22

R02 establishes the marketplace reporting and experimentation boundary.

Implemented direction:

- `report_delivery` is a derived UTC interval fact containing coarse
  demand/route/supply/geo/device dimensions, reconciled USD commercial sides,
  callback state, and accounting version. It is not a serving or settlement
  authority.
- Authenticated advertiser and publisher reports are always constrained by
  Genelet's session role/account; operators receive broader commercial/route
  aggregates. Agents remain outside reporting until S02 defines delegation.
- Metric source, formula, scope, currency, timezone, freshness, and
  zero-denominator behavior are versioned in the `reporting` package and
  documented for UI/internal JSON consumers.
- Controlled experiments use deterministic domain-separated assignment,
  append-only idempotent exposure/outcomes, explicit audited transitions,
  bounded expiry prune, and exact audited subject deletion. They have no path
  to change bids, budgets, delivery, or accounting.
- MySQL remains the analytical store after a reproducible 100,000-row query
  baseline met the current review thresholds. Production latency/row/retention
  triggers, rather than technology preference, govern future OLAP work.

Contract consequences:

- Populated systems require the R02 schema migration before ledger/report
  binaries; the local baseline is not a migration.
- Summer's authenticated JSON chartag remains an internal UI export, not the
  I03 public API.
- P02, S02, I03, S03, A02, and any named I02 integration inherit the R02
  dimension, permission, privacy, freshness, and observational-only rules.
