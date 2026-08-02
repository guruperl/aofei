# Campaign Delivery Guardrails

This document defines the D01 runtime contract for advertiser campaign and ad
group delivery. MySQL remains the configuration and reconciled-ledger source of
truth. Compiled candidate snapshots carry immutable eligibility facts, while
Redis serializes request-time reservations.

## Eligibility contract

A local candidate can enter selection only when all of these checks pass:

- advertiser, campaign, ad group, creative, publisher, site, and slot are
  active under the existing cache compiler and ACL rules;
- the current instant is inside both campaign and ad-group inclusive UTC
  start/end windows;
- both optional Monday-first 168-hour calendars allow the instant in the
  campaign's IANA delivery timezone;
- the compiled delivery snapshot is no older than
  `delivery_cache_max_age_seconds`;
- none of the campaign-total, campaign-daily, ad-group-total, or ad-group-daily
  spend/impression/click limits is already exhausted; and
- one atomic Redis reservation succeeds across every configured budget scope.

Zero or absent limit values mean unlimited for that dimension. Daily limits
and Redis daily keys reset by UTC date. The campaign delivery timezone affects
only the weekly calendar. An empty weekly calendar means all hours are allowed.

`Fast` pacing applies only the hard limits. `Even` pacing opens the daily limit
in proportion to the elapsed UTC day and opens a total limit in proportion to
the elapsed effective campaign/ad-group interval. Weekly calendars remain
separate eligibility gates. This is deterministic throttling, not traffic
forecasting, automatic bid adjustment, or ML optimization. Narrow calendars,
insufficient eligible traffic, and late starts can therefore underdeliver.

The selected bid and tracking payload retain their OpenRTB USD CPM value. A01's
`usd-cpm-impression-v2` contract converts that value to one-impression USD
spend (`CPM / 1000`) before this reservation. Configured spend limits and
reconciled ledger floors are USD amounts; they are never divided again.

## Reservation lifecycle

The reservation Lua script checks and increments spend plus one impression for
all limited scopes atomically. It seeds each Redis value with the maximum of its
existing value and a monotonic reconciled-MySQL floor. Releases clamp to that
floor, so overlapping reservations cannot decrement state below a newer ledger
baseline and a late or lower snapshot cannot reopen already reserved delivery.

Redis key families are:

```text
delivery:reservation:<random-token>
delivery:budget:total:<balance_id>
delivery:budget:daily:<YYYY-MM-DD UTC>:<balance_id>
```

- A response-generation failure, a replaced local winner, or a valid loss
  callback releases an active reservation.
- A successfully published impression callback finalizes it. A later release
  cannot reopen the budget.
- A successfully published click callback increments click state once through
  the signed reservation token.
- A measurement publication failure leaves the reservation active and releases
  only the tracking processing claim; a retry publishes and finalizes without
  a second budget increment.
- Reservation metadata covers the complete signed callback lifetime, including
  the accepted five-minute future clock skew.
- Total-budget state is persistent and conservative. Daily state expires only
  after its UTC day plus `delivery_state_ttl_seconds`. An unreported reservation
  can cause conservative underdelivery until an operator reconciles the balance;
  it never decrements merely because callback metadata expired.

Ledger interval jobs reconcile total balances from all `ledger_adv` facts and
daily balances from the current UTC date. The daily job corrects the same daily
baseline from `daily_adv`; `adv_balance.current_day` identifies the date. Cache
compilation treats a daily baseline from another date as zero.

## Failure semantics

| Failure | Runtime behavior |
|---|---|
| Expired or malformed delivery snapshot | Candidate fails closed before selection. |
| Redis unavailable for a limited candidate | Candidate fails closed; another local candidate or configured middleman fallback may still fill. |
| Reservation reaches a hard limit | Candidate is removed and selection retries another eligible candidate. |
| Release/finalize/click Redis failure | Measurement response remains unchanged; counters stay conservative and error metrics increase. |
| Tracking replay/cap Redis failure | Existing measurement fail-open behavior is unchanged. |
| NATS measurement publication failure | Callback remains retryable and the delivery reservation is not finalized or released. |

This intentionally gives budget safety precedence over availability for limited
local demand, while measurement publication retains its separate fail-open cap
semantics.

## Cache and deployment contract

The checked-in default delivery snapshot age is 900 seconds. Run the singleton
full cache publisher at least every 300 seconds. Route-only refresh does not
update delivery policy. A failed build preserves the prior static generation,
but that generation becomes ineligible at its embedded deadline instead of
authorizing stale pause or budget state indefinitely.

In local/spread mode, `cmd/spread` persists published snapshots and each local
controller reloads files every one-third of the tightest configured cache-age
bound. The reload loop does not query MySQL; the singleton cache publisher and
spread receiver must still be healthy.

RAdv cache payload version 2 adds delivery policy. New readers accept version 1
and legacy unversioned records, but old readers cannot decode version 2. Use
this rolling order:

1. back up and migrate MySQL with `adv_balance.current_day`, campaign timezone,
   campaign/ad-group weekly schedules, and pacing fields;
2. deploy all HTTP workers with the version-2-capable reader while the live
   cache is still version 1;
3. deploy the new singleton cache compiler and publish one full generation;
4. deploy the matching Summer components/templates and verify create/edit/read;
5. enable the five-minute cache timer and ledger interval/daily jobs.

Rolling back to an old HTTP binary after step 3 also requires the old cache
compiler to republish a version-1 generation (and equivalent spread snapshots)
before the old workers receive traffic.

For an existing database, use deployment-managed, reviewed `ALTER TABLE`
migrations rather than replaying `etc/step4_init.sql`. The target schema shape
is the checked-in baseline. Test the migration and inverse/restore procedure on
representative staging data before production.

## Observability

`/debug/vars` exposes aggregate and reason-specific cache/window/cached-budget
rejections, reservation attempt/success/limit rejection/Redis error, release,
finalization, and click counters under the `aofei_delivery_*` prefix. Alert on reservation errors immediately;
limited demand is fail-closed. Compare rejection rates with cache generation,
ledger completion, campaign edits, Redis health, and expected traffic rather
than treating every hard-limit rejection as an infrastructure error.
