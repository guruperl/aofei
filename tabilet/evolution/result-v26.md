# Result V26

A02 establishes W8M's optional hosted funding and payout boundary.

Implemented direction:

- Stripe Checkout collects advertiser payment data and Connect Express collects
  publisher identity/bank data on provider-hosted pages. Aofei stores opaque
  customer, checkout, account, payment, transfer, refund, settlement, payout,
  and dispute identifiers plus the immutable two-letter publisher onboarding
  country only; hosted URLs are not retained.
- Confirmed A01 statements remain the authority. Active operations cannot in
  aggregate exceed a statement or succeeded parent funding amount, Held state
  is re-read immediately before a provider call, and maker/checker/executor
  state transitions require separate S02 permissions, exact scope, and recent
  MFA.
- Independently approved provider-ready bindings are mandatory for both
  funding and payout. The first submission freezes its binding ID, so a later
  replacement cannot change retry parameters. A durable `Submitting` claim and
  stable provider idempotency key make request
  cancellation/crash recoverable. A fast webhook may win the response race;
  stale submissions can be taken over after two minutes by a separately
  authorized administrator without changing the provider key. Every attempted
  replay stops at a conservative 23-hour boundary, and uncertain calls remain
  capacity-reserved instead of becoming locally cancelable.
- Exact raw-body Stripe signatures are accepted under a bounded current/old
  secret overlap. Durable event IDs/content hashes prevent replay, provider
  timestamps plus event-specific transitions prevent regression. A
  dependency-pending event is retained and retried, then may make one guarded
  unresolved-to-applied/ignored transition without changing signed evidence.
  Linked
  provider Balance Transactions are retrieved exactly and recorded once per
  object/category, so retries do not double-count amount/fee/net while
  cumulative refund/dispute event evidence remains distinct.
- Refund, dispute, chargeback, transfer reversal, connected payout failure, and
  mismatched/missing settlement facts create scoped explicit reconciliation
  evidence. A different actor resolves exceptions; A01 settlement remains a
  separate audited transition and manual outage fallback.

Contract consequences:

- The clean schema gains six `hosted_*` tables and twelve protective triggers,
  reaching 94 tables, 6 routines, and 55 triggers. Financial identities,
  provider mappings, reconciliation evidence, and audit cannot be rewritten or
  deleted; only eligible unreferenced event envelopes have bounded retention.
- `hosted_payments` is a default-off config block with separate environment
  references for the API key and current/previous webhook secrets. Exact
  `POST /webhooks/stripe` and the three-role Summer module exist only behind the
  constructed service; navigation remains hidden while disabled.
- Mode-matched Stripe restricted keys are supported and preferred over broad
  secret keys; API version `2024-06-20` and the remote API origin are pinned.
- `cmd/hosted-payment` exposes aggregate health and bounded event retention but
  cannot move money. Provider calls/errors and webhook dispositions use fixed-
  cardinality metrics.
- Code completion does not enable a Stripe account or live mode. A populated
  migration, S02 permissions, sandbox/restore/incident evidence, and named
  legal, finance, tax, risk, privacy, support, and operational approval remain
  mandatory external gates.
