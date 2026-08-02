# Status A02 - Hosted Funding And Publisher Payout Integration

State: `[+]` Complete

## Goal

Integrate external hosted/tokenized funding and payout providers without making
Aofei a storage or processing boundary for full payment credentials.

## Dependencies

- A01 accounting and manual settlement safety.
- S02 financial permissions, 2FA, and audit controls.
- O02 recovery, secret rotation, webhook, and dependency-failure operations.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Provider boundary | `[+]` | Stripe Checkout and Connect Express sit behind a provider-neutral Go interface. Configuration contains secret references only; API responses are bounded/sanitized, movements require exact USD cents, and only opaque `cus_`, `cs_`, `acct_`, `tr_`, `re_`, and `txn_` identifiers cross the boundary. |
| Checkout and payout | `[+]` | Advertiser, publisher, and administrator pages expose only authorized binding, proposal, independent approval, execution, cancellation, and status actions. Approved provider-ready bindings are mandatory; the first provider attempt freezes the exact binding, and retries cannot follow a replacement destination. Funding/payout creation, approval, and execution remain distinct maker/checker transitions with a final Held-state check. |
| Webhook ingestion | `[+]` | Exact raw-body HMAC verification supports a bounded current/previous-secret overlap and rejects wrong mode, API version, account context, replay content, or stale transitions. Durable event identity/hash, connected-account isolation, and one guarded dependency-pending reprocess tolerate duplicate and reordered delivery without duplicate movement. |
| Reconciliation | `[+]` | Exact linked Balance Transactions record amount, fee, and net once. Refund, dispute/chargeback, transfer reversal, payout failure, and missing/mismatched settlement evidence create scoped exceptions that a different authorized actor resolves before the separate A01 settlement transition. |
| Secrets and access | `[+]` | Provider keys remain environment references; mode-matched restricted keys are preferred, secret keys are supported, and raw provider credentials/bodies/signatures never enter state or audit. Separate S02 grants, exact account scope, recent MFA, maker/checker separation, a DB-only maintenance capability, public-data guards, and Gitleaks protect the boundary. |
| Failure operations | `[+]` | Bounded same-key retry, durable Submitting ownership, two-minute authorized takeover, conservative 23-hour replay cutoff, aggregate health/metrics, bounded retention, recorded-provider fixtures, manual A01 fallback, deployment guidance, and incident procedures cover local failure operation. Live Stripe sandbox and production-governance evidence remain explicit external go-live gates. |

## Acceptance Criteria

- Aofei never receives or stores full card or bank credentials.
- Duplicate or reordered provider events do not double-fund, refund, or pay.
- Provider and internal balances reconcile with explicit unresolved exceptions.
- Provider outage leaves accounting recoverable through A01 manual workflows.

## Verification And Deep Review

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. Pinned
  staticcheck v0.5.1 passed in Aofei and with pzdesign's documented legacy style
  exclusions. The expanded Aofei race suite includes `hostedpayment` and
  `cmd/hosted-payment`; pzdesign hosted-payment/UI races and the full Genelet
  race suite also passed.
- The recorded-provider unit and disposable MySQL lifecycle passed signature,
  API-version, replay, reordering/reprocess, connected-account isolation,
  mandatory/frozen binding, exact scope/MFA, Held-state, refund/dispute/payout,
  reconciliation, retention, and failure-recovery cases. The restored baseline
  contains 94 tables, 6 routines, and 55 triggers and matches
  `etc/step4_init.sql` exactly.
- Documentation, public-data, template, public-copy, actionlint, and
  `git diff --check` gates passed. Gitleaks v8.27.2, the latest tested pin whose
  module supports the repository's Go 1.23.5 toolchain, found no history or
  current-tree leak in Aofei, pzdesign, or Genelet. Narrow allowlists contain
  only named synthetic idempotency/provider fixtures and the published RFC 6238
  test vector.
- The cache smoke ran against isolated disposable MySQL, Redis, NATS, volumes,
  ports, and state. The recovery drill restored the complete 94/6/55 inventory,
  immutable evidence, cache rebuild, and middleman preflight without reading or
  resetting the long-lived local database.
- The 100,000-row reporting benchmark passed at advertiser 101/127 ms,
  publisher 104/122 ms, and operator 1707/1732 ms median/max. DSP/match and the
  three-run capacity baseline passed with no unexpected result or status.
- Deep review closed fail-open/retry uncertainty, cross-account event,
  replacement-binding, unordered event, API-version, maintenance-capability,
  UI authorization, retention cleanup, and CI toolchain findings. Live Stripe
  sandbox traffic was not sent; it remains a named pre-live prerequisite with
  legal, finance, tax, risk, privacy, support, and operations approval.

## Reconciliation From A01

- Provider objects and webhook facts must link through opaque tokens and
  idempotency keys to an A01 statement or correction; they never replace the
  immutable daily source, adjustment, approval, or audit records.
- Provider settlement/refund/dispute state must reconcile explicitly to A01's
  USD six-decimal amounts and opaque `invoice:`, `payout:`, or `manual:`
  evidence. Outages fall back to A01's distinct-Unix-principal manual workflow
  without accepting full credentials into Aofei.

## Reconciliation From O02

- Provider webhook facts need durable idempotency before regional load
  balancing or retry. Readiness/drain and load-balancer retries cannot be the
  correctness boundary for funding, refund, dispute, or payout mutations.
- Encrypted off-Git backups retain opaque provider identities, sanitized
  evidence, reconciliation state, and secret-generation metadata without full
  credentials. Restore/provider-outage exercises must meet named RPO/RTO
  evidence and preserve the A01 manual fallback; the disposable local drill is
  not provider recovery proof.

## Reconciliation From D03

- D03 advertiser charge, downstream pay, and margin facts link to A01
  statements first; a hosted funding or payout provider settles those internal
  obligations through opaque provider identities. A bidder callback, route, or
  credential reference is never a provider payment instruction.
- D03 auction/callback replay suppression and provider webhook idempotency are
  different boundaries. A02 must preserve both identities and explicitly
  reconcile callback/retry partial state before funding, refunding, or paying;
  neither mechanism may infer success from the other.

## Reconciliation From R02

- Provider status, fee, refund, dispute, and payout state may appear only in
  authorized operator/accounting reports under opaque provider identifiers.
  They do not become auction runtime metrics, general advertiser/publisher
  dimensions, or experiment subjects, and R02 report values never initiate a
  funding or payout action.
- Provider reports preserve R02 UTC/USD six-decimal, source, freshness, partial
  and scope semantics while reconciling to A01 immutable statements. The
  existing internal JSON export is not a provider API or webhook surface; A02
  adds separate S02 permissions, idempotency, audit, and outage behavior.

## Reconciliation From P02

- Seller id/type/ASI and `source.schain` are public transparency metadata, not
  payout instructions or provider identities. Hosted payout tokens bind to the
  existing A01 publisher party through a separately verified, permissioned
  workflow; changing or authorizing seller metadata cannot redirect funds.
- Reseller/intermediary commercial fees require an explicit versioned A01
  contract, immutable source facts, maker/checker review, and reconciliation.
  A02 must not infer a revenue share or beneficiary from `complete=0`, source
  quality, management control, or a seller id.

## Reconciliation From S02

- Name separate permissions for hosted-checkout initiation, opaque funding
  token binding, payout-account binding, payout proposal, independent approval,
  cancellation, refund/dispute handling, reconciliation, and provider-secret
  readiness. Money movement and token rebinding require recent MFA, exact
  account scope, a safe reason, and immutable audit; the maker cannot be the
  checker or settlement recorder.
- Provider webhook authentication is a separate machine boundary, never an
  admin/advertiser/publisher S02 session. Human permissions cannot bypass
  signature, replay, order, and idempotency checks; a valid webhook cannot
  impersonate a human approver. Audit stores opaque provider object ids and
  hashes only, never raw signatures, API keys, session/TOTP/recovery data, full
  request bodies, card data, or bank credentials.
- Credential binding and financial actions derive actor and party scope from
  verified S02 identity plus A01 facts before mutation. Cross-account and
  insufficient-MFA denials occur before provider calls, idempotency claims, or
  local accounting changes. Operators use the separated S02 maintenance and
  application database principals for security evidence retention.

## Reconciliation From I03

- An I03 advertiser management credential is not a hosted-payment customer
  token, payout-account token, provider API key, webhook identity, or human
  financial approval. A02 must never embed/reuse it with a provider or infer
  money-movement authority from `api.campaign.*`, `api.creative.*`,
  `api.targeting.*`, or `api.report.read`.
- Any external funding/payout API uses separate versioned paths, credentials,
  S02 financial scopes, per-party quotas, idempotency namespace, immutable
  audit, and maker/checker operation states. I03's generic write and cache
  publication acknowledgement cannot mutate or claim provider, statement,
  adjustment, settlement, refund, dispute, or payout success.

## Reconciliation From S03

- A valid provider webhook, dispute, chargeback, refund, or payout failure is a
  provider/payment fact, not an S03 invalid-traffic signal. It cannot activate,
  resolve, or override a quality case or rule; quality dependency failures
  likewise cannot fabricate provider state.
- An S03 Hold is an internal A01 Draft/Confirmed statement state produced only
  from complete reviewed evidence and separate recommender/checker actors. A02
  must refuse new capture/payout movement while that statement is Held, retain
  idempotent provider facts for reconciliation, and require the explicit A01
  release/correction workflow before resuming. Approved Exclude/Reverse
  recommendations do not themselves call a provider.
- Provider token binding and quality evidence remain separately scoped and
  retained. Webhooks and hosted pages never receive rule evidence or digests;
  quality reviewers never receive provider secrets, raw signatures, payment
  methods, or bank/card data. Any combined operator view uses exact S02
  financial/quality permissions, recent MFA, safe reasons, and immutable audits.
