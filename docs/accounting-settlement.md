# Accounting And Manual Settlement

This is the A01 monetary, statement, approval, correction, reconciliation, and
sensitive-data contract. It remains the manual authority and provider-outage
fallback. A02 now adds an independently disabled hosted Stripe Checkout/Connect
adapter without replacing these records; see
[hosted-funding-payout.md](hosted-funding-payout.md).

## Monetary Unit And Billable Facts

The active contract is `usd-cpm-impression-v3`, recorded by the singleton
`acct_contract` row. It preserves the A01 statement semantics while making the
upstream CPM, impression aggregation, reservation, and ledger path exact:

- OpenRTB bid prices, auction macros, and tracking payload prices remain USD
  CPM. The local auction carries the authoritative six-place value to the wire
  boundary; a price of `2.500000` means USD 2.50 per thousand impressions.
- One billable impression contributes `CPM / 1000`; therefore CPM `2.500000`
  contributes USD `0.002500`.
- Live reservations use integer nano-USD per-impression amounts. Loss or
  publication failure releases the reservation; a successfully published,
  signed impression finalizes it.
- The ledger counts only idempotently accepted impression trackers as spend.
- New tracker and win/loss facts carry `usd-cpm-impression-v3` at top level.
  The ledger labels unmarked float-only drain records as v2, recognizes the
  bounded pre-marker exact transition by its exact fields, and rejects unknown
  or mixed versions. V3 middleman facts must bind USD exact charge, pay, and
  margin to the billable demand CPM; compatibility floats cannot reconstruct a
  missing v3 value.
  The exported compatibility constructor preserves v2 when its caller supplies
  only a float; only a present exact CPM can originate a v3 tracker.
  Likewise, decoding a draining v1/v2 RAdv cache validates and uses its bounded
  float adapter without populating the v3 exact field, and the resulting
  billable tracker remains labeled v2.
- Statement request/party/cadence/period/currency/source/supersession identity
  is database-immutable. Draft or Held totals may change only when they equal
  the sum of immutable adjustment rows; corrections create a replacement and
  original statements cannot be deleted.
  Clicks remain measurement facts and do not create a second charge.
- Ledger accumulation retains source precision. A statement snapshot rounds
  the daily aggregate to six decimal places. Statement, adjustment, and total
  mutations use `DECIMAL(20,6)` in MySQL and integer micro-dollars in Go.
- Delivery-report downstream and returned CPM sums likewise add integer
  six-place CPM values with overflow checks and write canonical decimal
  strings; their floating fields are read-only compatibility projections.
- USD is the only supported currency. Unsupported bid-floor or downstream
  currencies cannot silently enter a USD statement.

Current commercial facts are deliberately explicit:

| Fact | Source | Formula |
|---|---|---|
| Advertiser charge | `daily_adv.spend` | sum of accepted impression CPM / 1000 for the advertiser |
| Publisher pay | `daily_pub.spend` | sum of accepted impression CPM / 1000 for the publisher under the current direct-supply contract |
| Middleman upstream charge | `daily_mid.charge_spend` | sum of upstream charge CPM / 1000 |
| Middleman downstream pay | `daily_mid.pay_spend` | sum of bounded downstream pay CPM / 1000 |
| Middleman margin | `daily_mid.margin_spend` | non-negative charge minus pay, per accepted impression |
| Statement total | `acct_statement` | immutable source snapshot plus immutable adjustment rows |

Middleman reconciliation sums the nine-place daily charge, pay, and margin
facts and checks `charge - pay = margin` before any statement rounding. Its
operator output includes both nine-place exact totals and six-place statement
projections, so a sub-micro discrepancy remains visible and blocking while an
exact identity cannot be rejected merely because the three display values
round independently.

Local advertiser charge and publisher pay currently use the same gross
impression fact; no undocumented revenue-share percentage is deducted. A new
commercial share requires a versioned accounting contract and migration, not
a UI-only formula change.

The legacy `adv.balance`, `pay_payment`, `his_payment`, and related `pay_*`
compatibility rows are not statement, funding, delivery, or settlement
authority. No active Summer route changes them, and the old payment view and
balance-crediting trigger are absent. Existing deployments archive or remove
them under the migration policy rather than reconciling A01 from those rows.
An absent legacy publisher floor remains `NULL` and is labeled `AlreadyExact`
in migration evidence; only present binary-float floors use the legacy-rendered
conversion label.

## Statement And Approval Lifecycle

Statements are daily UTC, Monday-through-Sunday UTC, or UTC calendar-month
snapshots. `request_key` makes creation idempotent. Party access is advertiser
or publisher plus the numeric account ID; `currency` is fixed to USD.

```text
Draft -> Held -> Draft
Draft/Held -> Confirmed -> Held
Confirmed -> Settled
Confirmed/Settled -> Corrected + replacement Draft
```

- Creation and every mutation require a bounded operator identity and reason.
- Adjustments are additive immutable rows and are allowed only in Draft or
  Held. They cannot make a statement total negative and never rewrite ledger,
  D01 budget limits, or Redis hard-budget floors.
- Confirmation re-reads the daily source inside the transaction. If it differs
  from the snapshot, confirmation fails and the operator must reconcile and
  create a correction.
- The creator cannot confirm the same statement. The confirmer cannot record
  its settlement. This maker/checker separation is enforced by the service.
- Settlement records only an opaque `invoice:`, `payout:`, or `manual:`
  document/ticket reference. It rejects account-number-like references and
  never stores a card, bank account, or routing credential.
- Correction preserves the old statement as `Corrected` and creates a linked
  replacement Draft from current source facts. Its request key makes retries
  return that same replacement; confirmed or settled values are never silently
  edited.
- `acct_adjustment` and `acct_audit` are protected by database triggers against
  update and deletion.

The command is an operator surface, not a public API. Restrict its executable,
config, and database permissions to named accounting Unix accounts. The command
derives `created_by`, `confirmed_by`, and `settled_by` from the effective Unix
UID; there is no actor flag or environment override. Maker, checker, and
settlement recorder must therefore invoke it through distinct non-shared OS
principals. S02 now adds granular portal RBAC and recent-MFA gates, but cannot
weaken this service state machine or replace its OS-principal separation.

## Operator Commands

Use a generated or deployed `AOFEI` config and never put reasons, statements,
or references into Git:

```bash
export AOFEI=/etc/aofei/aofei.json

# Run as the named maker's Unix account.
GOWORK=off go run ./cmd/accounting \
  -action=create -party=advertiser -party-id=7 -cadence=monthly \
  -from=2026-07-01 -to=2026-07-31 \
  -request-key=invoice-2026-07-adv-7 -reason='July period close'

GOWORK=off go run ./cmd/accounting \
  -action=adjust -statement-id=44 -amount=-0.250000 \
  -reason='Approved service credit ticket AC-42'

# Run as a different authorized Unix account.
GOWORK=off go run ./cmd/accounting \
  -action=transition -statement-id=44 -status=Confirmed \
  -reason='Source and adjustment review complete'

# Run as a third authorized Unix account after external manual settlement.
GOWORK=off go run ./cmd/accounting \
  -action=transition -statement-id=44 -status=Settled \
  -reference=invoice:ticket-AC-44 -reason='Settlement evidence archived'

GOWORK=off go run ./cmd/accounting \
  -action=reconcile -statement-id=44

GOWORK=off go run ./cmd/accounting \
  -action=reconcile-middleman -from=2026-07-01 -to=2026-07-31

GOWORK=off go run ./cmd/accounting \
  -action=export -party=advertiser -party-id=7 >statement-export.csv
```

CSV export contains statement metadata and amounts only. It does not export
addresses, emails, payment credentials, raw events, or bank details. Store
exports in access-controlled, encrypted operator storage outside Git.
Statement listing never defaults to all parties: browser callers pass the
authorized advertiser/publisher scope from their typed principal. An offline
operator must add `-all-parties` for a cross-party CSV and cannot combine that
flag with `-party` or `-party-id`.

## Reconciliation And Finality

Reconcile in this order:

1. Accepted signed/replay-suppressed impression facts and D01 reservation
   finalization.
2. Interval `ledger_*` counts and spend against complete `winloss.<stamp>`
   input; missing input remains retryable and does not become a zero interval.
3. `daily_*` totals against all interval rows for the UTC day.
4. Middleman charge, pay, margin, callback forwarding, and retry state.
5. Draft/held statement `source_amount` against current daily facts.
6. Immutable adjustments and `total_amount`.
7. Confirmation, external settlement evidence, and audit actors.

A source difference is explicit `ErrSourceDiscrepancy`; it never rewrites a
statement automatically. Callback retry republishes only the documented
downstream notification and must not create another impression or adjustment.
S03 traffic-quality review uses these Held/correction semantics and cannot
delete facts. Only complete evidence resolved as `InvalidTraffic`, including a
case whose appeal was denied, can create a billing recommendation. `Hold`
requires a second administrator and may move
only a Draft or Confirmed statement to Held with a distinct `quality_hold`
accounting audit; `Exclude` and `Reverse` stop at approved recommendations for
an A01 correction workflow. The recommender cannot approve the same item, an
open or upheld appeal blocks approval, and only a denied appeal remains a
confirmed-invalid state. No quality action rewrites ledger, measurement,
statement source, adjustment,
or settlement facts. See
[traffic-quality-anti-fraud.md](traffic-quality-anti-fraud.md).
The read-only middleman reconciliation independently checks that non-negative
daily charge minus pay equals margin, at six-decimal USD precision, and reports
the exact difference without mutating source rows.

## Existing-Deployment Migration

`etc/step4_init.sql` is a clean-environment baseline, not a production
migration script. A populated deployment needs a reviewed, backed-up migration
that performs these changes in one maintenance window:

1. Stop auction admission, cache refresh, ledger, and callback/accounting jobs;
   preserve encrypted database and log backups outside Git.
2. Verify every relevant price is USD CPM. Quarantine unsupported currency or
   unexplained historical rows instead of guessing a conversion.
3. Divide existing `spend` fields in `ledger_log`, `ledger_adv`, `ledger_pub`,
   `ledger_pub_adv`, `daily_log`, `daily_adv`, `daily_pub`, and
   `daily_pub_adv` by 1000. Divide `charge_spend`, `pay_spend`, and
   `margin_spend` in `ledger_mid` and `daily_mid` by 1000. Use one transaction,
   six-decimal reviewed results, and an idempotent migration marker.
4. Reconcile `adv_balance.current_spend` from converted ledger facts. Do not
   divide `limit_spend`: configured limits are already currency amounts.
5. Create the `acct_*` tables, immutable triggers, and historical
   `usd-cpm-impression-v2` marker. An A03 deployment then follows the separate
   frozen-backup migration in [exact-money.md](exact-money.md) and advances the
   singleton only after exact-column conversion succeeds.
6. Remove full `cardnumber`, `routing_number`, and `account_number` values,
   sender identity, and payment IP fields from the live schema after the legal
   retention owner either approves deletion or moves a required record into
   approved encrypted storage. Remove `view_payment` and the balance-crediting
   `trig_payment`; retire the old Summer funding routes and account-balance
   action. Never copy retired values into statements, logs, tickets, or Git.
7. Delete only the explicit D01 delivery reservation/budget Redis key families
   under an approved maintenance runbook, republish static caches, restart all
   binaries at the same accounting version, and canary synthetic traffic.
8. Prove CPM `2.5` reserves and ledgers USD `0.0025`, regenerate daily rows,
   reconcile sample advertiser/publisher/middleman totals, and only then reopen
   traffic.

Never mix v1 reservation writers with v2 ledger readers. Rollback restores the
database and Redis backup plus the complete prior binary set; it is not a
partial multiplication of selected rows.
