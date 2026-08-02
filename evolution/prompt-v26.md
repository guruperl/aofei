# Prompt V26

Add externally hosted advertiser funding and publisher payout integration while
keeping full payment credentials and live-provider authority outside W8M.

- place Stripe Checkout and Connect Express behind a provider-neutral boundary
  that stores only opaque object identifiers and exact A01 statement linkage;
- require S02 exact account scope, recent MFA, maker/checker/executor separation,
  stable idempotency, aggregate amount limits, and a final Held-state recheck;
- authenticate raw webhooks before database work, durably deduplicate replay and
  repeated provider facts, tolerate event/HTTP races and reordering, and retain
  immutable sanitized reconciliation/audit evidence;
- cover refunds, disputes, chargebacks, transfer/payout failures, fees/net
  settlement, provider outages, secret overlap, retention, alerting, and A01
  manual fallback; and
- keep the feature and its navigation disabled by default, with sandbox,
  migration, rollback, and separate business approval required before live
  mode.
