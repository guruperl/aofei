# Status M46 - Review3 Cross-Lane Remediation

State: `[~]` In progress

## Goal

Resolve the confirmed correctness, operational-safety, concurrency, hot-path,
and static-analysis findings from `review3.md` without weakening the established
exact-money, immutable-generation, creative-containment, or sensitive-data
contracts.

## Review Disposition

| Finding | State | Resolution |
|---|---:|---|
| 1. Over-scale OpenRTB floor | Rejected | Public floors support at most six decimals and excess scale deliberately fails closed. |
| 2. Missing exact CPM stops ledger interval | Rejected | Zero/missing authoritative CPM is malformed accounting input; skipping it would silently underbill. |
| 3. Numeric date-stamped settlement refs | Rejected | Settlement accepts opaque document/ticket references and deliberately rejects account-number-like digit groups. |
| 4. Route margin schema gap | P2 confirmed | Keep `0..1` cache validation and add database/preflight enforcement. |
| 5. Unset demand cost type | P1 confirmed | Require explicit CPM for current and legacy auction/accounting eligibility. |
| 6. Existing-directory chmod | P2 confirmed | Never mutate an existing directory that `EnsureDir` did not create. |
| 7. Script substring detector | P3 confirmed, qualified | It is limited defense in depth, not a sanitizer or the browser containment boundary. |
| 8. Legacy spread reset receiver | Rejected | It is receiver-first rollout compatibility and is ignored after generation activation. |
| 9. Exposure lock upgrade | P2 confirmed, qualified | Existing-row idempotent retries can upgrade duplicate shared locks to `FOR UPDATE`; fresh inserts are not assumed to deadlock. |
| 10. Host-specific MaxMind default | Rejected | Operators must pass `-city` or `AOFEI_GEOLITE_CITY_FILE`; no host path is portable authority. |
| 11. Numeric hosted-payment reasons | Rejected | The broad separator-aware digit guard is the approved payment-data boundary. |
| 12. Audience alias Redis RTT | P3 confirmed, qualified | Use one cross-key Lua read; `SMISMEMBER` cannot read one member across multiple keys. |
| 13. Repeated callback-client wrapping | P3 confirmed | Normalize protected clients once per controller/job run. |
| 14. Dead direct-publisher predicate | P3 confirmed | Remove the unreachable version-mismatch branch. |
| 15. Disabled unused-code check | P3 confirmed | Pinned U1000 reports 17 unused symbols; remove them and enable the gate. |

## Tasks

| Item | State | Notes |
|---|---:|---|
| Review intake and dependency reconciliation | `[x]` | Validated all 15 findings against current code, tests, and operator contracts. Completed source milestones remain closed; M46 precedes conditional I02. |
| Explicit CPM demand eligibility | `[x]` | `exactCPM` now requires explicit CPM for exact and legacy adapters; headerless/v1/v2 unset records remain readable but cannot bid or bill, and database compilation reports the item and unsupported type before delivery hydration or cache publication. |
| Middleman margin schema contract | `[x]` | The baseline constrains group and nullable route overrides to exact `0..1` fractions; activation checks active rows, the populated-system migration validates source shape and every row before ALTER without inference, and a disposable MySQL 8.0.41 drill proves failure, installation, enforcement, and rerun behavior. |
| Existing-directory ownership | `[x]` | `EnsureDir` chmods only a directory successfully created by that call; existing and raced paths are validated without mutation, broader permission bits fail, restrictive modes plus setgid/sticky bits are preserved, and concurrent creation tests cover the ownership boundary. |
| Concurrent experiment exposure | `[x]` | The immutable post-insert exposure check now uses `FOR SHARE` instead of upgrading a duplicate `INSERT IGNORE` record to `FOR UPDATE`; disposable-MySQL coverage runs simultaneous first writes and eight-way idempotent retries and requires one stored row with no caller error. |
| Audience and callback hot paths | `[x]` | Canonical/legacy audience membership now runs through one bounded cross-key Lua action with one-call canonical-hit, legacy-hit, and miss tests. Controllers reuse their construction-time protected client; retry batches normalize once before the loop, while the full URL/redirect/rebinding suite remains green. |
| Creative-boundary and validation clarity | `[x]` | Markup validation now names its literal script checks as defense in depth and tests that obfuscated scripts remain supported; docs assign executable containment to the opaque-origin no-top-navigation renderer or an external consumer. Direct-publisher validation now checks its required v3 marker once after the embedded publisher independently passes v3 validation. |
| Static-analysis hygiene | `[ ]` | Remove all pinned-U1000 findings and add U1000 to the repository gate. |
| Verification and deep review | `[ ]` | Run schema/cache/runtime/full-repository gates, review the complete milestone, and resolve every P1/P2 before closeout. |

## Acceptance Criteria

- Unset cost types cannot become commercial authority through any current or
  legacy demand representation.
- Invalid route margins fail at schema/activation boundaries without publishing
  a partial cache or silently changing operator data.
- Existing shared directories retain their ownership and complete mode while
  unsafe permissions produce an actionable error.
- Concurrent exposure retries remain immutable and idempotent without a lock
  upgrade; alias reads and callback forwards retain their safety contracts with
  bounded hot-path work.
- Creative documentation identifies the renderer sandbox as containment and
  does not present substring checks as JavaScript sanitization.
- Pinned staticcheck passes with both `SA*` and `U1000` enabled.

## Boundaries

- `review3.md` remains untracked intake; this status file is the durable record.
- No production database, deployment, credentials, traffic, Cloudflare, or
  other external state is mutated by M46 implementation or verification.
- I02 remains planned and demand-gated because no named mobile integration
  supplies supported platform and lifecycle requirements.
