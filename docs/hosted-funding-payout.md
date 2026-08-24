# Hosted Advertiser Funding And Publisher Payouts

This is the A02 product, security, deployment, and incident contract for
Stripe-hosted advertiser funding and publisher payout onboarding. The feature
is disabled by default. Implementing or deploying the code does not create a
Stripe account, accept Stripe terms, satisfy legal/tax/risk obligations, move
money, or authorize live mode.

## Boundary And Provider Choice

W8M uses Stripe Checkout for advertiser payment collection and Stripe Connect
Express for publisher identity/bank onboarding. W8M creates Transfers to an
approved connected account; Stripe, not W8M, collects the publisher's bank
details and manages the later external-account payout. These hosted products
keep full card numbers, bank account numbers, and routing credentials outside
Aofei and its Summer/Genelet pages.

The Go package `hostedpayment` places those calls behind a provider-neutral
interface. Stripe is the only implemented adapter. The interface does not make
providers interchangeable without a separate legal, operational, event-model,
and reconciliation review.

Official provider references:

- [Checkout Sessions](https://docs.stripe.com/api/checkout/sessions)
- [Connect hosted onboarding and Account Links](https://docs.stripe.com/connect/hosted-onboarding)
- [Create a Transfer](https://docs.stripe.com/api/transfers/create)
- [Refunds](https://docs.stripe.com/api/refunds/create)
- [Balance Transactions](https://docs.stripe.com/api/balance_transactions/retrieve)
- [Account transfer capability](https://docs.stripe.com/connect/account-capabilities)
- [Supported event types](https://docs.stripe.com/api/events/types?api-version=2024-06-20)
- [Idempotent requests](https://docs.stripe.com/api/idempotent_requests)
- [Webhook delivery behavior](https://docs.stripe.com/webhooks)
- [Webhook signature verification](https://docs.stripe.com/webhooks/signature)

Aofei stores only opaque provider identifiers such as `cus_`, `cs_`, `acct_`,
`pi_`, `ch_`, `tr_`, `re_`, `txn_`, `po_`, and `du_`; exact USD amounts;
the publisher's two-letter onboarding country; state; timestamps;
provider-event payload hashes; bounded error codes; reasons; and redacted
audits. Hosted page URLs are returned once and never persisted.
API keys, webhook secrets, raw signatures, raw webhook bodies, provider error
messages, full payment credentials, and identity documents are prohibited from
the database, UI, logs, exports, tickets, and Git.
Audit reasons and idempotency keys reject plausible routing/account/card digit
groups (including spaces, hyphens, dots, slashes, and parentheses), IBAN-like
forms, and provider secret prefixes without retaining or echoing the value.

## Financial State And Authority

Every movement references one immutable A01 `acct_statement`. New funding and
payout require `Confirmed`; `Held`, `Draft`, `Settled`, or `Corrected` fail
before a provider call. A refund instead requires an exact succeeded/partially
refunded parent funding operation on the same advertiser and original
statement, so legitimate refunds remain possible after invoice settlement and
feed the separate A01 correction workflow. Active funding or payout proposals
for one statement cannot in aggregate exceed its total. Active refunds cannot
in aggregate exceed the parent funding operation. Amounts must be positive USD
values exactly representable in cents; A01 continues to retain six-decimal
source precision.

Bindings follow:

```text
Proposed -> Ready -> Approved -> Revoked
```

Provider readiness alone never approves a customer or payout destination. A
different human must approve it. Approving a replacement revokes the prior
approved binding. Loss of Connect payout readiness automatically revokes the
binding; restored readiness returns it to `Ready` and requires human approval
again. Readiness requires submitted account details, enabled payouts, and an
`active` Connect `transfers` capability; test-mode permissiveness is not proof
of live readiness. Both funding Checkout and publisher Transfer execution
require the matching independently approved, provider-ready binding. The
two-letter payout country is part of the immutable binding identity, so a
request-key replay with another country fails instead of silently reusing the
first account.

Money operations follow:

```text
Proposed -> Approved -> Submitting -> Submitted -> Succeeded
                                      |            |-> Disputed
                                      |            |-> PartiallyRefunded -> Refunded
                                      |-> Failed / Canceled
```

The maker cannot approve the same operation, and its checker cannot execute
it. Proposal, approval, execution, cancellation, token binding, reconciliation,
and exception resolution all require named S02 permissions, exact account
scope, recent MFA, and a bounded reason where applicable. Every provider call
uses the operation's stable idempotency key. A stale `Submitting` operation can
be retried after two minutes by a different recent-MFA administrator who has
both execute and reconciliation authority; the takeover is audited and reuses
the exact provider key.
the first funding or payout submission also freezes the exact approved binding
ID on the operation. Retries therefore keep the same customer or payout-account
token even if a later independently approved binding becomes current. Stripe
can prune idempotency results after 24 hours, so
W8M permits replay only within a conservative 23-hour window measured from the
first durable submission claim. After that boundary, execution and Checkout
reopening fail closed: operators must reconcile provider evidence and must not
resubmit the movement under either the old or a new key. This limit also applies
when an uncertain transport response was returned locally to `Approved`.
Attempted-but-uncertain operations cannot be locally canceled or excluded from
the statement-capacity total; keep the original statement/evidence intact and
use an A01 correction/manual obligation workflow when provider evidence cannot
conclusively recover the operation.

A successful provider fact does not silently mark the A01 statement Settled.
Operators first reconcile the provider amount, fee, and net, resolve explicit
exceptions with an independent actor, then use the existing A01 transition and
opaque external evidence. This preserves manual settlement as the recoverable
fallback.

## Configuration And Secrets

The checked-in example remains inert:

```json
"hosted_payments": {
  "enabled": false,
  "provider": "stripe",
  "live_mode": false,
  "api_base_url": "https://api.stripe.com",
  "api_key_env": "STRIPE_API_KEY",
  "webhook_secret_env": "STRIPE_WEBHOOK_SECRET",
  "webhook_previous_secret_env": "",
  "public_base_url": "https://www.w8m.com",
  "request_timeout_ms": 5000,
  "max_body_bytes": 131072,
  "webhook_tolerance_seconds": 300,
  "max_attempts": 3,
  "retry_base_ms": 100,
  "event_retention_days": 400,
  "reconciliation_max_age_days": 90
}
```

JSON contains environment-variable names only. Store the referenced values in
the deployment secret manager or a root-owned service environment. Prefer a
mode-matched Stripe restricted key (`rk_test_` or `rk_live_`) granting only the
Customers, Checkout Sessions, Accounts/Account Links, Transfers, Refunds, and
Balance Transactions access required by the endpoints above. A mode-matched
secret key (`sk_test_` or `sk_live_`) is accepted when restricted-key support is
not viable; publishable and wrong-mode keys are rejected. Live mode also
requires the exact `https://api.stripe.com` endpoint. Apply provider-side IP
restrictions when the deployment has stable egress. Current and previous
webhook secrets must use distinct references and each secret must be at least
16 characters.
Outgoing API requests pin Stripe version `2024-06-20`; configure the webhook
endpoint to the same version and treat any version change as a reviewed
provider-contract migration.

For rotation, add the old signing secret under
`webhook_previous_secret_env`, deploy the new current secret to every HTTP
node, verify both generations, update the Stripe endpoint, wait through the
maximum delivery/retry window, then remove the old reference and roll again.
The readiness page reports presence and mode only; it never displays values.

## Webhook Contract

When enabled, `cmd/unify` exposes only exact `POST /webhooks/stripe` for this
machine boundary. The reverse proxy must preserve the byte-exact body and
`Stripe-Signature`, allow `application/json` POSTs, disable caching, and use
HTTPS. Human sessions and permissions cannot substitute for a signature.
Cloudflare or another proxy must not transform, challenge, redirect, or cache
this path.

Subscribe only to the event families the code handles:

- `account.updated`, `payout.failed`;
- `checkout.session.completed`, `checkout.session.expired`,
  `checkout.session.async_payment_succeeded`,
  `checkout.session.async_payment_failed`;
- `payment_intent.succeeded`, `payment_intent.payment_failed`;
- `charge.succeeded`, `charge.failed`, `charge.refunded`,
  `charge.refund.updated`;
- `charge.dispute.created`, `charge.dispute.updated`,
  `charge.dispute.closed`;
- `transfer.created`, `transfer.reversed`;
- `refund.created`, `refund.updated`, `refund.failed`.

Signature verification covers the unmodified body and timestamp before any
database call. A unique provider event ID and SHA-256 content hash suppress
replay; reuse with different content fails. Supported updates lock the exact
binding/operation and apply provider timestamps plus event-specific transition
rules. Production event snapshots must declare the pinned Stripe
`2024-06-20` API version, so old or weaker events cannot regress terminal state
or silently enter through a different object schema. Later events can
enrich the same operation with immutable Charge, Transfer, Refund, and Balance
Transaction links without regressing state. Stripe does not publish a
per-transaction availability webhook: an authorized reconciliation action
retrieves each linked `txn_` object through the pinned API, requires
`available` status, exact source ownership, USD direction and amount, and valid
fee/net arithmetic, then records one immutable matched fact or an explicit
unresolved exception. Unsupported valid events are retained as `Ignored`;
unmatched or inconsistent facts become explicit `Unresolved` reconciliation
rows. Invalid signatures return 400. Transient processing failures return 503
so the provider can retry; accepted, duplicate, ignored, and durably unresolved
events without a pending owner dependency return 204.

When a supported event arrives before its referenced operation or payout
binding can be resolved, W8M stores the `Unresolved` event and exception, then
returns 503 so Stripe retries. A same-body retry must match the immutable event
ID and SHA-256 hash. Once the missing owner exists, one guarded
`Unresolved -> Applied|Ignored` processing transition can attach that owner and
apply the event exactly once; every signed envelope field and the receipt
evidence remain immutable. This handles Stripe's explicitly unordered delivery
without retaining raw bodies or allowing a changed replay to mutate state.

## Schema And Populated-System Migration

The clean baseline adds six tables: `hosted_binding`, `hosted_operation`,
`hosted_provider_object`, `hosted_event`, `hosted_reconciliation`, and
`hosted_audit`. `hosted_binding.country` retains only the immutable two-letter
publisher onboarding country and is null for advertiser customer bindings.
`hosted_operation.binding_id` is null before first submission and then freezes
the independently approved customer or payout destination used by every retry.
Provider-object ownership is immutable. Financial operation
identity, binding identity, reconciliation evidence, and audits cannot be
rewritten or deleted. Provider events can be deleted only by the bounded
retention command on its dedicated connection; events referenced by retained
reconciliation evidence remain protected.
The event trigger permits only the bounded processing transition described
above; provider identity, type, object, timestamp, payload hash, and receipt
time cannot change.

`etc/step4_init.sql` is not a production migration. For a populated system:

1. Keep `hosted_payments.enabled=false`; back up and restore-test the database.
2. Apply a reviewed additive migration for the six tables and twelve hosted
   triggers. Grant the HTTP principal only the operations required by the
   service; keep broader maintenance privileges on a separate host/principal.
3. Deploy code/templates disabled and verify existing advertiser, publisher,
   A01, identity, auction, and reporting paths.
4. Configure Stripe test mode, the exact public origin, webhook endpoint,
   permissions, recent MFA, secrets, proxy behavior, and clocks on a canary.
5. Complete the sandbox matrix below and a backup/restore exercise before any
   live-mode decision.

Rollback disables `hosted_payments` on every HTTP node and removes its menu
permissions. Do not drop tables/triggers, delete operations/events/audits, alter
opaque mappings, or infer settlement during rollback. Continue A01 manual
settlement while the provider boundary is unavailable.

## Sandbox Verification

Use a dedicated Stripe test account and test data only. A local Stripe CLI may
forward signed events to the exact endpoint; never paste its signing secret
into Git or command history retained by support tooling.

Verify at least:

1. Advertiser binding, independent approval, Confirmed statement funding,
   mandatory-binding denial, hosted redirect, duplicate/concurrent event,
   completed payment, balance fee/net, and A01 settlement.
2. Publisher Connect Express onboarding, `account.updated` readiness,
   independent binding approval, Confirmed statement transfer, and managed
   external payout behavior.
3. Cancellation before completion, partial/full refund, dispute won/lost,
   transfer reversal, payout failure, wrong currency/amount, missing mapping,
   invalid/stale signature, duplicate/reordered events, and provider 429/5xx.
4. A statement moved to Held after approval produces no new provider call.
5. A canceled client request or crash after provider acceptance is recovered
   with the same idempotency key and does not duplicate the movement. Replacing
   the party's current binding during recovery does not change the operation's
   already selected provider input.
6. Cross-account, missing-permission, missing-recent-MFA, maker/checker, and
   secret-readiness denials expose no credential or raw payload.

The repository integration fixture uses a disposable MySQL 8 instance and a
recorded provider adapter; it is not proof of a live Stripe account or bank
payout.

Configure Stripe event destinations for both the platform account events used
by Checkout/refunds/transfers and the connected-account events used by Connect.
Stripe identifies every connected-account event with the top-level `account`
field. W8M accepts `account.updated` and `payout.failed` from that namespace
only when this field identifies the exact stored payout binding. Other
connected-account activity, including direct charges, is retained as
out-of-scope and cannot claim a platform funding/refund/payout operation through
copied metadata.

## Operations, Metrics, And Incidents

Read aggregate health without exposing provider tokens:

```bash
AOFEI=/etc/aofei/aofei.json /opt/aofei/bin/hosted-payment -action=health
```

This maintenance binary constructs a DB-only capability: it does not read API
or webhook secret values and its type exposes no provider, webhook, or money
movement methods.

Alert on any approved operation whose A01 statement is Held, any
`stuck_submitting`, `stuck_canceling`, any `stale_submitted`, any
`unresolved_past_policy`, sustained webhook 4xx/5xx, a drop to zero expected
webhook volume, or rising provider errors. Fixed-cardinality counters are:

- `aofei_hosted_payment_provider_requests_total`
- `aofei_hosted_payment_provider_errors_total`
- `aofei_hosted_payment_webhook_requests_total`
- `aofei_hosted_payment_webhook_invalid_total`
- `aofei_hosted_payment_webhook_errors_total`
- `aofei_hosted_payment_webhook_duplicates_total`
- `aofei_hosted_payment_webhook_reprocessed_total`
- `aofei_hosted_payment_webhook_applied_total`
- `aofei_hosted_payment_webhook_unresolved_total`
- `aofei_hosted_payment_webhook_ignored_total`
- `aofei_hosted_payment_reconciliation_unresolved_total`

Prune only unreferenced expired event envelopes from the restricted maintenance
host:

```bash
AOFEI=/etc/aofei/aofei-maintenance.json \
  /opt/aofei/bin/hosted-payment -action=prune-events \
  -limit=1000 \
  -reason='approved provider-event retention schedule'
```

During a provider/webhook incident, stop new proposals/execution by removing
the relevant S02 permissions or disabling the feature on all nodes; do not
delete local state. Preserve aggregate metrics, event IDs/hashes, operation
IDs, A01 statement IDs, provider dashboard object IDs, timestamps, and external
incident tickets without copying raw bodies or credentials. Reconcile the
provider dashboard against immutable local mappings, retry only with the
existing idempotency key and only inside its 23-hour safety window, resolve
exceptions with a different authorized actor, and use A01 correction/manual
workflows for obligations that were never submitted or remain conclusively
unrecoverable. Never cancel an uncertain attempted movement merely to release
capacity.
Rotate secrets if compromise is suspected and treat unconfirmed provider state
as unresolved—not failed, succeeded, funded, refunded, paid, or settled.

## Live Go-Live Decision

Live mode is a separate business change. Before it, a named owner must approve
the Stripe platform/Connect account, supported countries and currencies,
merchant-of-record and tax treatment, refund and chargeback policy, publisher
KYC/support ownership, sanctions/fraud review, reserves/negative balances,
fees, payout timing, privacy disclosures, data processing terms, customer
support, incident escalation, and finance reconciliation. Retain sandbox,
security, restore, canary, and rollback evidence. Only then provision live
secrets and explicitly enable a controlled canary. No repository test can make
that decision automatically.
