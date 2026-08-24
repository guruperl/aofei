# Exact Monetary Sources

A03 defines `usd-cpm-impression-v3` as the active authoritative monetary
contract. It supersedes the active v2 marker while preserving A01's
six-decimal statement boundary and keeping historical v2 facts labeled v2.

## Units and arithmetic

- Public demand prices and supply floors are USD CPM decimal strings with at
  most six fractional digits. The supported range is `0.000000` through
  `999999.999999`; a billable demand price must be at least `0.000001`.
- Go represents CPM as signed integer micro-dollars per thousand impressions.
  MySQL represents it as `DECIMAL(12,6)`.
- One billable impression is represented as integer nano-USD. Conversion is
  exact: one CPM micro-unit equals one impression nano-unit. Redis reservations
  and ledger aggregation use integer nano-USD and checked addition.
- Values are aggregated before conversion to A01 `DECIMAL(20,6)` statements.
  That single boundary rounds half away from zero. UI and CSV display six
  decimals and never feed formatted values back into authority.
- Overflow, negative demand price, excess scale, NaN/Inf, negative zero, and
  an unsupported commercial model fail closed before mutation. CPC, CPA, and
  ROI remain unsupported.

## Authority inventory

| Source | A03 authority | Compatibility only |
|---|---|---|
| Demand item price | Exact CPM column and versioned RAdv field | OpenRTB JSON number and retired `adv_item.cost` float history |
| Supply floor | Exact CPM column | OpenRTB JSON number and retired `pub_slot.bidfloor` float history |
| Campaign/item budgets | Exact nano-USD limit/current columns | Retired float columns and rendered UI numbers |
| Auction reservation | Redis integer nano-USD counters | Version-2 float cache/state during bounded drain only |
| Interval/daily/local/middleman ledger | Exact nano-USD columns | Preserved legacy float values and comparison evidence |
| Reports/experiments | Derived exact decimals, never money authority | Ratios, aggregates, exports, and formatted values |
| Statements/settlements/hosted payments | A01/A02 micro-USD/cents contracts | `adv.balance`, `his_payment`, and inactive provider metadata |

Historical IEEE-754 values are not recoverable as their original human-entered
decimals. Migration preserves those values, records the conversion method and
discrepancy, and quarantines invalid or ambiguous rows. Formatting a legacy
float to six or nine places does not make it exact authority.

## Mixed-version rule

Exact schema columns, cache payloads, Redis keys, and evidence carry the v3
contract. A v3 writer never updates an unversioned monetary key. Publication
uses O03's complete-generation protocol; readers retain the previous complete
generation until every v3 family is present. Old numeric management clients
are read-only after activation and receive a deprecation error on money writes.

RAdv payload v3 omits binary monetary fields and carries exact CPM plus
nano-USD delivery balances. Readers may convert a v2 float payload once for a
bounded drain, but republishing emits v3. Mutable delivery state is isolated
under `delivery:v3:*`; its spend fields use Redis integer operations and do not
touch the retired unversioned float family. Win/loss records carry exact local
or middleman CPM, and ledger aggregation uses checked nano-USD addition.

## Frozen-backup comparison

Migration is an offline maintenance operation. Before the freeze, assign a
change owner, accounting reviewer, database/Redis backup owner, canary owner,
and rollback decision deadline. Confirm that all binaries are still on the
expected v2 contract, inventory every affected column, and resolve every
unsupported model, negative authoritative value, over-scale value, and prior
migration artifact. Do not infer a value for an ambiguous row.

At the freeze:

1. Remove auction nodes from admission and stop cache publication, callback,
   interval/daily ledger, accounting, hosted-payment execution, and all
   management writers. Preserve read access only.
2. Take a consistent encrypted physical MySQL backup plus a Redis snapshot in
   access-controlled storage outside Git. The chosen MySQL backup must preserve
   the legacy IEEE-754 payloads exactly; a default logical `mysqldump` is not
   sufficient because its rendered `FLOAT` text can restore to different
   target-scale values. Record checksums, sizes, backup-tool/version and restore
   procedure, source commit/config, `acct_contract`, schema inventory, cache
   generation, last admitted request time, and all singleton high-water marks.
3. Restore the same frozen backup twice in isolation. One untouched restore is
   the rollback proof. Run `etc/a03_exact_money_migration.sql` only on the
   comparison restore. Its executable preflight requires the exact v2
   singleton, report default, 21 historical float columns, two already-exact
   four-place route minimums, and no prior evidence table; a partial or v3
   source fails before durable mutation.
4. Compare every primary-key/column tuple in the authority inventory, not only
   table sums. Compare the old database-rendered value at the target scale to
   the migrated decimal and retain a digest plus row counts. Review every
   `money_migration_evidence` row and require zero `Quarantined` rows before
   activation. Values outside signed 64-bit nano-USD or supported CPM range are
   quarantined and stop the script before column or contract promotion. Restore
   the frozen source and resolve them; do not retry the partial database.

Tolerance is source-specific and applies only to an
`LegacyRenderedHalfAway` row recovered from pre-v3 IEEE-754 storage:

- CPM and floor conversion may differ by at most `0.000000500` USD CPM, half
  one target micro-USD CPM unit.
- Aggregated amount conversion may differ by at most `0.000000001` USD, one
  target nano-USD unit because evidence itself is stored at nine places.
- `AlreadyExact`, every new v3 mutation, cross-table totals, statements, and
  provider cents have zero tolerance. A tolerance is never permission to
  rewrite an exact source or make totals balance.
- The pre-v3 `mid_route_group.min_margin_cpm` and
  `mid_route_bidder.min_margin_cpm` columns were already exact
  `DECIMAL(10,4)` values. Their evidence is `AlreadyExact` with zero
  discrepancy even though v3 widens their supported scale to six places.

`scripts/aofei-exact-money-drill.sh` rehearses this contract using only
synthetic data and three uniquely named disposable MySQL containers. It creates
a deterministic pre-v3 fixture, takes and checksum-verifies one stopped
physical data-directory backup, proves an untouched restore, migrates a second
restore, compares all affected
source tuples, validates evidence/tolerances/schema/contract, and exercises the
duplicate reservation, ledger, statement, and provider-event guards. Its
owner-only temporary backup is deleted on exit and is not production evidence:

```bash
./scripts/aofei-exact-money-drill.sh
```

Archive the production comparison manifest in the approved financial change
record. Store no customer rows, backup fragments, provider tokens, or Redis keys
in Git, chat, logs, or tickets.

## Canary and activation

After the isolated comparison passes, keep writers frozen and use this order:

1. Apply the reviewed migration to the frozen primary. Require 23 affected
   columns to be decimal, the `report_delivery` default and singleton
   `acct_contract` to be `usd-cpm-impression-v3`, the evidence table immutable,
   and the complete schema/trigger inventory to match the release manifest.
2. Install the v3 cache compiler and publish complete versioned RAdv/static
   shadows through O03's completeness marker. Inspect the manifest before any
   live pointer changes. Never repopulate an old live key in place.
3. Start one v3 HTTP node outside admission against the v3 schema and complete
   cache generation. Prove canonical management strings, rejected numeric
   writes, one local and one middleman synthetic CPM conversion, exact
   reservation/finalization/release, duplicate callback suppression, and
   account-scoped report output. No real provider movement is part of a canary.
4. Admit the canary at a bounded share. Compare accepted impressions to Redis
   nano-USD deltas, win/loss facts, interval/daily ledgers, publisher pay,
   middleman charge/pay/margin, and draft statement source. Require exact
   equality for v3 facts and no new v2 writer/key/report rows.
5. Expand HTTP nodes, then restart singleton cache, callback retry,
   interval-ledger, daily-ledger, and accounting jobs one at a time. Release
   management writes only after all writers are v3. Retain old Redis families
   read-only through the approved drain window; never copy their float value
   into a v3 key.

Stop expansion on any discrepancy, overflow, quarantine, partial cache
generation, new v2 write, duplicate identity, readiness failure, hosted
reconciliation anomaly, or unexplained change in spend/revenue/margin. Preserve
the failed evidence before taking corrective action.

## Rollback and correction

Before the first v3 authoritative write, rollback means removing all v3 nodes,
restoring the checksum-verified frozen MySQL and Redis generations, reinstalling
the complete prior binary/config set, verifying the untouched-restore digest,
then reopening admission. Do not reverse selected decimal columns, multiply
amounts, delete migration evidence, or point an old writer at v3 mutable keys.

After any v3 authoritative write or accepted impression, restoring the frozen
database can lose or duplicate money and is no longer a rollback. Freeze
admission and all writers, preserve MySQL/Redis/provider evidence, identify the
last exact high-water marks, and roll forward. Rebuild derived cache/report
state from authoritative MySQL facts. Financial differences use A01 immutable
adjustment/correction statements with independent review; hosted-payment
differences use A02 reconciliation and provider-side evidence. Never replay a
provider create/refund/payout merely to make local totals agree.

`money_migration_evidence`, statements, adjustments, hosted operations/object
mappings/reconciliations, and their audits follow the approved financial and
statutory retention schedule and remain immutable. Encrypted backup generations
follow the O02 default of 35 daily and 13 monthly copies unless a documented
legal/accounting/privacy owner sets a stricter schedule. Expiry destroys the
whole approved generation and reapplies deletion cases; it does not selectively
erase evidence from a retained accounting chain.
