# Status D03 - External DSP / AdX Middleman Activation

State: `[+]` Complete

## Goal

Safely activate the existing external DSP / AdX middleman runtime through a
staged, observable, and reversible production workflow.

## Dependencies

- Revenue traffic requires D01, I01, S01, A01, and O01 acceptance.
- Credential material remains environment-managed and outside Git/MySQL/Redis.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Bidder onboarding | `[+]` | Existing advertiser metadata/admin approval now revalidates the exact OpenRTB profile, requires a portable credential name, creates or verifies a same-advertiser synthetic chain, and keeps every synthetic demand row inactive. Admin health and command preflight reject broken chains. |
| Credential readiness | `[+]` | Environment JSON headers are resolved without returning values; invalid names/values, CRLF, and hop-by-hop headers are rejected. UI/database store only `[A-Za-z_][A-Za-z0-9_]*` names of at most 128 bytes, and redaction/output tests cover values. |
| Route setup | `[+]` | Database and admin health checks reject missing active targets/bidders, inactive/unapproved profiles, invalid target entity/size pairs, and synthetic rows enabled as local demand. Existing route validation covers trigger, timeout, margin, priority, and inventory scope. |
| Route publication | `[+]` | `-validate-middleman` compares a newly compiled MySQL generation with the published Redis v2 version, source, entry count, checksum, and route high-water mark. Routes remain Redis-only to preserve one activation/revocation timeline. |
| Fallback canary | `[+]` | The `fallback` preflight enforces both disclosure/runtime gates, keeps `Always` off, and requires a Fallback entry. Existing in-process suites cover contextual isolation, COPPA/gate blocks, fill/no-bid/timeout/invalid/floor outcomes, callbacks/retry, and settlement; the disposable restore drill proves published-route and credential wiring without partner traffic. |
| Always gate | `[+]` | The separate `always` preflight requires all three gates and an active Always entry. Existing auction tests prove valid higher marked-up CPM competition, local-winner preservation on invalid/failing responses, and reservation release. Checked-in/runtime defaults remain off. |
| Operations | `[+]` | `docs/middleman-activation.md` defines onboarding, publish/preflight, fallback and optional Always canaries, fixed evidence, accounting reconciliation, rotation, disablement, rollback, and explicit no-traffic semantics. |

## Acceptance Criteria

- Controlled staging proves the current feature without publishing secrets or
  changing public response contracts.
- Fallback failures preserve valid local behavior and produce actionable
  metrics/health evidence.
- Charge, pay, margin, win/loss, and callback-retry facts reconcile before
  revenue traffic.
- Disabling either gate and republishing routes provides a tested rollback.

## Verification

- Existing middleman, callback, retry, route-cache, SSP, and ledger suites.
- Credential-redaction and endpoint-safety tests.
- Route health/read commands, production-style canary checklist, and full
  closeout gates.

## Exclusions

- Arbitrary downstream markup rewriting remains closed unless R01 proves that
  cooperative notification cannot satisfy a measurement requirement.

## Result

- Added a read-only activation command that verifies current MySQL topology,
  exact Redis v2 publication, partner profiles, credential availability, and
  stage-specific gates while emitting counts/checksum only. It performs no
  partner request and mutates no database or cache state.
- Kept the existing runtime/config response contracts and Redis key names.
  Production traffic remains disabled because no named partner, approval
  record, or live-mutation authority was supplied; D03 completion means the
  staged activation mechanism and evidence contract are ready.
- The deep review found two safety gaps: active synthetic reporting rows could
  enter local demand, and invalid active route targets could be silently
  omitted. Database/admin preflight now rejects both. It also tightened
  environment-reference portability and outbound header syntax/CRLF safety.
- D03 does not add spread/local route snapshots. Redis routes are small shared
  operational state with one bounded-refresh activation/revocation timeline;
  copying them into demand snapshots would create conflicting disablement
  state.

## Closeout Verification

- Passed Go 1.23.5 full tests and vet in Aofei and pzdesign.
- Passed pinned staticcheck in both repositories and the documented race suites
  for DSP/matching/cache/callback/ledger/lease/action/commands plus pzdesign
  unify, Summer, and tools.
- Passed documentation, public-data, template, public-copy, SQL-baseline,
  actionlint, and both repository `git diff --check` gates.
- Passed the documented DSP/match benchmark suite. Latest local results include
  `BenchmarkServeBidLocalTwoImpressions` 255857 ns/op,
  `BenchmarkServeSSPLocalTwoAdUnits` 280973 ns/op, and
  `BenchmarkSelectOneParallel` 190.1 ns/op; these are local evidence, not a
  production latency claim.
- Passed `scripts/aofei-recovery-drill.sh` in uniquely named disposable MySQL
  and Redis containers: restored inventory, rebuilt seven Redis keys, and
  completed the fallback middleman preflight in 31 seconds. The live cache,
  deployed database, external partner, and production service were untouched.
- No evolution entry is required: D03 implements the already planned ownership,
  privacy, accounting, and Redis-route boundaries without changing a public or
  repository boundary.

## Reconciliation From S01

- Every external bidder remains contextual even when local personalization is
  granted. D03 onboarding must contractually approve the S01 allowlist,
  regulatory-signal handling, retention, deletion, and incident disablement;
  credentials and raw request capture remain prohibited.
- The canary must observe `aofei_privacy_middleman_blocked_total` and prove that
  disabling either the privacy disclosure gate or the middleman runtime stops
  fanout without disturbing eligible local delivery.

## Reconciliation From O01

- Fallback canaries must configure the exact upstream admission profile and
  keep middleman work within its own bounded timeout. Evidence must use O01's
  fixed local, middleman, dependency, timeout, invalid-response, and overload
  classifications.
- Metrics and alerts must never place a partner, bidder, route, endpoint, or
  credential in a dynamic metric key or label; use bounded audit records for
  authorized partner-specific diagnosis.

## Reconciliation From A01

- The revenue canary must prove charge, downstream pay, and non-negative margin
  are each converted from CPM to one-impression USD exactly once and reconcile
  `daily_mid`, `daily_adv`, `daily_pub`, and sample A01 statements before
  enabling traffic.
- Rollout must use one accounting version across reservation, callback, ledger,
  daily, and statement jobs. Callback retries cannot create another billable
  impression, and external evidence remains an opaque A01 reference rather
  than bidder-supplied payment data.

## Reconciliation From D02

- D03 activates only the D02-validated response path. Bidder onboarding and the
  fallback canary must cover exact dimensions/media, secure callbacks/markup,
  contained Banner, well-formed VAST, strict requested Native assets, and
  hostile/encoded response rejection before any `Always` competition.
- `Always` canaries must prove a higher valid marked-up CPM releases the local
  D01 reservation once, while an invalid response or callback-setup failure
  preserves the valid local winner. Observe bounded invalid-response and
  creative-rejection metrics without logging raw `adm`.

## Reconciliation From O02

- Canary and rollback run on the O02 multi-node topology: publish compatible
  route/cache state first, add one ready node, preserve N-1 headroom, and
  withdraw readiness before drain. A successful origin-node exercise is not a
  public SLO result when DNS, TLS, proxy, or bidder dependencies were bypassed.
- Callback retry and cache publication remain renewable singleton jobs whose
  loss of lease fails the run; route/callback idempotency is still the durable
  partition backstop. Recovery evidence must cover route republish, callback
  backlog, credential rotation/disablement, and charge/pay/margin
  reconciliation before revenue traffic resumes.
