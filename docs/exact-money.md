# Exact Monetary Sources

A03 defines `usd-cpm-impression-v3` as the authoritative monetary contract.
It extends, rather than replaces, A01's six-decimal statement contract.

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
