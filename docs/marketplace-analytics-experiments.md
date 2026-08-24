# Marketplace Analytics And Experiments

R02 adds authorized marketplace reports and a controlled experiment contract.
Both are analytical: they read or append derived facts and never change auction
bids, delivery schedules, reservations, budgets, accounting statements, or
settlement state.

## Authorities And Storage

`report_delivery` is the interval reporting fact derived transactionally by the
ledger job from immutable win/loss input. It is not the accounting authority;
`ledger_*`, `daily_*`, and the A01 statement/audit records remain authoritative
for reconciliation and settlement. Monetary values in `report_delivery` are
`DECIMAL(20,6)` USD amounts already converted at the billable-impression
boundary. Reports must not divide them by 1000 again.

The interval identity is `(timely, dimension_hash)`. `dimension_hash` is a
deterministic SHA-256 digest over the complete demand, route, supply, geography,
and device dimension tuple; every original dimension remains stored in an
explicit queryable column. This avoids silently dropping dimensions to fit a
wide MySQL unique index. The legacy generated
`device_key=device_os*256+device_type` remains available for lookup
compatibility but is not the report identity. Ledger interval and report
dimension identities reject duplicate insertion; they do not merge a replay
silently.

`measurement_action` is the direct, idempotent action source. Its retention is
bounded by `action_retention_hours`, so action-based totals are partial outside
that window. Report facts never recreate expired actions.

## Metric Registry

All dates and intervals are UTC. Amounts are USD with six decimal places unless
the table says otherwise. The Go registry in `reporting/contracts.go` is the
machine-checked contract.

| Metric | Exact source and formula | Authorized scope | Freshness |
|---|---|---|---|
| Impressions | `SUM(report_delivery.imps)` | advertiser, publisher, operator | interval fact |
| Clicks | `SUM(report_delivery.clis)` | advertiser, publisher, operator | interval fact |
| CTR | clicks / impressions; zero for a zero denominator | advertiser, publisher, operator | interval fact |
| Actions | `COUNT(measurement_action)` | advertiser, operator | receipt and retention window |
| CVR | actions / clicks; zero for a zero denominator | advertiser, operator | partial if either source is unavailable |
| Spend | `SUM(report_delivery.spend_usd)`; already per-impression USD | advertiser, operator | interval fact |
| Revenue | `SUM(report_delivery.revenue_usd)`; already per-impression USD | publisher, operator | interval fact |
| Cost | `SUM(report_delivery.cost_usd)`; already per-impression USD | operator | partial while callback/retry is unresolved |
| Margin | `SUM(report_delivery.margin_usd)`; nonnegative | operator | partial while callback/retry is unresolved |
| ROI | (purchase value - spend) / spend; zero for a zero denominator | advertiser, operator | partial if action or delivery input is unavailable |
| ROAS | purchase value / spend; zero for a zero denominator | advertiser, operator | partial if action or delivery input is unavailable |
| Downstream CPM | sum of raw downstream bid CPM / impressions | operator | auction/callback evidence |
| Returned CPM | sum of returned upstream CPM / impressions | operator | auction/callback evidence |

Derived CTR, CVR, ROI, ROAS, CPC, CPA, and effective CPM values are report-only
analytics. They are not supported auction cost types and are never fed back
into an auction automatically.

## Dimensions And Visibility

Authorized reporting dimensions are demand source (`Local`, `Fallback`,
`Always`, or `MiddlemanUnknown`), advertiser/campaign/item/creative,
bidder/group/route/target, publisher/site/slot, numeric country/state, numeric
device OS/type, and the closed P02 supply categories: inventory environment,
integration mode, media intent, placement, render context, refresh mode,
refresh seconds, ad density, traffic quality, source quality, and management
control. An operator-authorized public seller contributes seller type and id;
missing or unauthorized values become an explicit unknown seller type and an
empty id. Route and bidder fields are operator-only. The table contains no IP
address, user agent, browser ID, advertising ID, consent string, email address,
raw supply-chain input, private partner metadata, or raw experiment subject.

The authenticated path role (`_grole`) supplied by Genelet is the authority:

- advertisers always query `WHERE report_delivery.adv_id=<session account>`;
- publishers always query `WHERE report_delivery.pub_id=<session account>`;
- operators can query all rows and commercial sides;
- agents have no R02 report until S02 defines a trustworthy delegated-account
  relationship. An `adv_id`, `pub_id`, or `admin_id` request parameter cannot
  change the authenticated scope.

The advertiser and publisher pages expose only their own commercial side.
Operator pages include charge/pay-derived spend, revenue, cost, margin,
downstream/returned CPM, route identity, and callback errors. R01 action facts
are shown separately from geo/device delivery facts so joins cannot multiply
actions.

The sibling Summer module exposes authenticated HTML and JSON chartags under
`/goto/<role>/<chartag>/ledger?action=topicsMarketplace`. The JSON chartag is an
authenticated UI export using the same account scope; it is not the public I03
API contract.

## Freshness And Partial Data

Every report shows its UTC interval range, USD/accounting version, and source
freshness. The current UI classification is:

- `current`: the newest interval fact is less than two hours old;
- `partial`: interval facts are older, the latest complete daily input is
  behind yesterday UTC, retained actions do not cover the selected period, or
  callback retries remain unresolved;
- `unavailable`: a required interval/daily source has no high-water mark;
- `unknown`: action receipt has no high-water mark for a scope where actions
  are applicable;
- `not_applicable`: publisher views do not expose advertiser actions.

A shared dependency, callback, or reconciliation failure is never rendered as
a true zero. The interval ledger must run after complete win/loss files are
available; action reconciliation and callback retry lag are separate high-water
marks.

## Controlled Experiment Contract

Experiments are operator- or advertiser-owned definitions with an immutable
version, declared primary and guardrail metrics, two to twenty variants, and
exactly 10,000 allocation basis points. State transitions are explicit:

```text
Draft -> Running -> Stopped -> Completed
                 -> Completed
```

Only the trusted create operation generates an assignment salt; caller-supplied
salts or algorithm versions are rejected. New experiments use assignment
algorithm v2: SHA-256 over a fixed domain label, experiment id, algorithm
version, experiment version, the decoded random salt, and a 32-byte hexadecimal
pseudonym. It is deterministic inside one version and unlinkable across
experiment identities even if input is reused. Only the resulting 32-byte hash
is stored. `report_exposure` accepts one append-only variant per experiment
version and subject hash until its bounded expiry or an exact authorized
subject deletion.

The schema compatibility default is assignment algorithm v1, reproducing the
pre-R03 salt/pseudonym/experiment-version hash byte-for-byte while its retained
exposures expire. Trusted creates explicitly store v2. The algorithm version,
salt, and experiment version plus all variant keys/allocations are immutable;
a running experiment is stopped and replaced by a new experiment/version
instead of changing buckets. List and operator report output disclose the
algorithm version but never the salt.

`report_experiment_outcome` attaches an append-only observed value to an existing
exposure. The caller supplies a 32-byte hexadecimal idempotency digest and an
exact six-decimal `DECIMAL(20,6)` string. Only the experiment's declared
primary or guardrail metric is accepted; conflicting reuse of the idempotency
digest fails. Outcomes cannot precede exposure. The runtime API is:

1. `reporting.LoadExperiment`;
2. `reporting.Assign`;
3. `reporting.RecordExposure`;
4. after observing a declared metric, `reporting.NewOutcome` and
   `reporting.RecordOutcome`.

`Assign` also returns the stored owner scope and a private salt-bound proof.
`RecordExposure` reloads the exact experiment/version/variant contract and
requires its owner, algorithm, exposure window, and calculated expiry to match
before an idempotent insert. A caller-built struct, wrong-account assignment,
undeclared variant, altered retention, or stopped experiment without a prior
matching exposure fails before insertion. `RecordOutcome` repeats that proof
and scope validation against the immutable stored exposure, then permits only
its experiment's declared metrics and exact idempotency tuple.

The caller owns the pseudonym and event-digest derivation. Do not pass raw
account, cookie, email, device, or conversion identifiers. The package has no
automatic campaign or auction mutation path. Operator UI results aggregate
exposure count and primary/guardrail record counts and values per variant;
assignment salts, subject hashes, idempotency digests, and audit reasons are
never exported.

## Operator Workflow

Create a reviewed draft using the exact service environment:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json \
  go run ./cmd/report-experiment \
  -action=create -owner=advertiser -adv-id=17 \
  -name=reviewed-copy-v1 -version=1 \
  -primary-metric=actions -guardrail-metric=spend \
  -starts-at=2026-08-03T00:00:00Z -ends-at=2026-08-10T00:00:00Z \
  -retention-hours=2160 \
  -variants=control=5000,treatment=5000 \
  -reason='approved experiment definition'
```

Review the returned ID, then use explicit transitions:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json go run ./cmd/report-experiment -action=list
GOWORK=off AOFEI=/etc/aofei/aofei.json go run ./cmd/report-experiment -action=start -experiment-id=7 -reason='approved start'
GOWORK=off AOFEI=/etc/aofei/aofei.json go run ./cmd/report-experiment -action=stop -experiment-id=7 -reason='guardrail review'
GOWORK=off AOFEI=/etc/aofei/aofei.json go run ./cmd/report-experiment -action=complete -experiment-id=7 -reason='analysis complete'
```

Mutations use the effective OS UID as the audit actor and require a bounded
reason. Keep this command on an authorized operations host; it is not a public
HTTP service.

Run the bounded retention task from a singleton authorized operations timer:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json \
  go run ./cmd/report-experiment -action=prune -limit=1000
```

For a verified privacy request, derive the already-stored domain-separated
subject hash on the authorized privacy host, then delete only that experiment
version. Do not put the hash in shared shell history or logs, and keep the audit
reason free of identifiers:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json \
  go run ./cmd/report-experiment -action=delete-subject \
  -experiment-id=7 -version=1 -subject-hash=<64-hex-hash> \
  -reason='verified privacy request'
```

The deletion removes the exact exposure and its outcomes transactionally and
records a `SubjectErased` audit without the hash. It never enumerates adjacent
subjects.

## Retention, Query Evidence, And OLAP Trigger

Keep interval report facts only for the approved analytical window. The current
operating target is at most 400 days; deletion/archival is an operator-owned,
audited database maintenance procedure and is not performed by an HTTP node.
Each experiment sets 24–9,600 retention hours (90 days by default). Assignment
stores the expiry; outcomes must occur before it. The bounded prune deletes
outcomes before exposures in one transaction. Exact verified subject deletion
remains available before expiry. Experiment definitions and non-identifying
audit rows remain for control evidence.

Run the clean-room benchmark after schema, index, dimension, or query changes:

```bash
./scripts/aofei-reporting-benchmark.sh
```

The 2026-08-01 P02-expanded baseline used MySQL 8.0.41 in a disposable
container on x86-64 with 8 visible CPUs, 100,000 synthetic interval rows, a
two-day range, a 200-row limit, and five warm measured runs. Median/max results
were advertiser 100/119 ms, publisher 105/118 ms, and operator 1684/1830 ms.
These are local query measurements, not production p95/p99 or an availability
SLO.

MySQL remains the R02 store because the measured baseline meets the review
threshold of 250 ms for account views and 2 seconds for the broader operator
view at this workload. Reconsider partitioning, summary tables, or an OLAP
store only after production evidence shows one of these for three consecutive
review windows: account-report p95 above 2 seconds, operator-report p95 above 5
seconds, more than 50 million retained interval rows, or a required retention
window longer than 400 days. A technology change must preserve account scope,
metric formulas, UTC/USD semantics, immutable experiment identities, freshness,
and reconciliation back to MySQL facts.

## Rollout And Verification

Apply the populated-database migration before deploying a ledger binary that
writes `report_delivery` or a Summer binary that queries it. The checked-in
baseline is not a production migration. Recommended order:

1. take and verify the encrypted backup required by the O02 runbook;
2. add the R02/P02 tables or columns, indexes, foreign keys, and immutable
   triggers, and backfill an exact `dimension_hash` for any retained delivery
   facts before enforcing interval uniqueness;
3. verify the schema and a scoped read on a canary database;
4. deploy the ledger writer, then the Summer report pages;
5. compare interval totals to ledger/daily/accounting facts and inspect all
   freshness states;
6. create experiments only after the observational integration is reviewed.

Rollback disables the report navigation/writer before reversing application
binaries. Preserve R02 rows for reconciliation; do not drop facts as an
application rollback. Verification includes:

```bash
GOWORK=off go test ./reporting ./internal/jobs/ledger ./dsp ./etc ./cmd/report-experiment
./scripts/aofei-reporting-benchmark.sh
./scripts/aofei-recovery-drill.sh
(cd ../pzdesign && GOWORK=off go test ./summer/ledger)
```
