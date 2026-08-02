# Result V21

The active roadmap now targets a commercially safe W8M advertiser-publisher
marketplace.

Implemented direction:

- The original single-digit M-lane history is normalized to `M00` through
  `M09`, and later completed M-lane status files remain historical delivery
  records.
- New work uses zero-padded D/P/R/I/S/A/O domain lanes with one status file per
  review and commit unit.
- The first delivery wave is campaign guardrails, direct SSP commercial
  readiness, privacy, manual accounting safety, and production traffic
  controls.
- Existing direct SSP and middleman capabilities have explicit staged
  activation milestones; configuration availability is not treated as
  commercial readiness.
- Conversion/reporting, interoperability, native SDKs, management APIs,
  security, settlement, and availability work now have named owner lanes.
- S04 owns the page-by-page template escaping and XSS audit carried from M18;
  historical M16-M25 carry-forward rows now point at their completed work or
  current D03/A01/A02/S04 owner instead of remaining ambiguously pending.
- Automatic bidding/ML, internally operated payment-card processing,
  multi-region deployment, and million-RPM engineering remain deferred behind
  documented evidence triggers.

Unchanged direction:

- This roadmap change does not alter Go APIs, schema, cache payloads, HTTP
  contracts, active configuration, or deployed services.
- `pub`, `pub_site`, and `pub_slot` remain the direct SSP ownership boundary.
- Middleman fanout remains explicitly gated and Redis-route-cache backed.
