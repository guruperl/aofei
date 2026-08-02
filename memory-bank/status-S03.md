# Status S03 - Traffic Quality And Anti-Fraud

State: `[+]` Completed

## Goal

Add explainable, reviewable traffic-quality controls that reduce invalid
delivery and billing risk without opaque automated model decisions.

## Dependencies

- R01 stable impression/click/action identities.
- S01 privacy limits on traffic signals and retention.
- O01 traffic, rate, latency, and operational telemetry.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Signal taxonomy | `[+]` | Strict aggregate windows derive the eight closed signals; counters/keys/future time are bounded and infrastructure failures cannot enter the taxonomy. |
| Rule engine | `[+]` | Immutable rule versions implement Draft/Observe/Canary/Active/Disabled rollout and Observe/Flag/Throttle/Reject/Quarantine; incomplete evidence always reduces action and billing to Observe. |
| Review workflow | `[+]` | S02-scoped Summer pages and the domain service provide advertiser/publisher/partner evidence, review, optimistic case history, own-account appeal, independent resolution, enforcement, and rollback. |
| Billing boundary | `[+]` | Complete confirmed-invalid evidence may recommend Exclude/Hold/Reverse; separate approval applies only a Draft/Confirmed A01 Hold, re-checks live appeals, and never rewrites source facts. |
| Privacy and retention | `[+]` | HMAC digests replace raw event/partner identity, summaries are bounded, evidence expires in 1–720 hours, aggregates retain 365–2555 days, and fresh-context cleanup discards uncertain connections. |
| Operations | `[+]` | Fixed metrics, per-rule health, exact canary-review/false-positive gates, detached serving snapshots, CLI health/prune, incident guidance, and fail-open stale-snapshot behavior are implemented. |

## Acceptance Criteria

- Every enforcement decision names a rule version and safe reason.
- A rule can be observed and canaried before blocking or changing billing.
- Cross-account evidence and personal identifiers are access-controlled and
  retention-bounded.
- False-positive and rollback tests cover each blocking rule.

## Verification

- Sequence/replay/rate fixtures, rule-version and canary tests, authz/privacy,
  ledger reconciliation, race/load, and full closeout gates.

## Exclusions

- ML-based fraud scoring is part of the automatic-ML deferral in
[docs/defer.md](../docs/defer.md).

## Result

- Added `trafficquality` with strict aggregate detection, HMAC identities,
  versioned rules, immutable decisions, case/appeal history, per-rule counters,
  reviewed enforcement snapshots, billing recommendations, and bounded
  retention. `cmd/traffic-quality` provides strict 64 KiB aggregate ingest,
  protected health output, and dedicated-connection evidence pruning.
- Added nine `quality_*` tables and ten immutability/retention triggers. Database
  checks keep incomplete evidence observe-only, action/billing mappings exact,
  and rule/enforcement canary state unambiguous. The clean baseline is 88
  tables, 6 routines, and 43 triggers.
- `cmd/unify` requires S02 identity when quality review is enabled. The
  `trafficquality` Summer component supplies administrator rule/review/
  enforcement controls, exact delegated reads, and advertiser/publisher
  own-scope appeals in Chinese and English without exposing digests.
- DSP serving checks publisher scope before demand lookup, filters local
  advertiser candidates and external partner assignments, refreshes an
  immutable snapshot independently of request cancellation, and fails open
  after maximum age. The default-disabled path performs no quality hashing or
  candidate-filter allocation.
- Updated the database, privacy, identity, accounting, observability,
  operations, advertiser, publisher, Pzdesign UI, product, architecture, and
  toolchain contracts. Evolution V25 records the new optional boundary; A02 and
  conditional I02 carry the reconciled constraints.

## Deep Review Closeout

- Replaced aggregate-counter Canary→Active approval with an exact query over
  selected complete Canary decisions and their current human review, so older
  Observe reviews cannot satisfy rollout.
- Required complete evidence for billing recommendations and re-checked the
  current case during approval. An open/upheld appeal blocks billing; a denied
  appeal remains confirmed invalid. Maker/checker and A01 statement-state
  checks remain transactional.
- Bounded every input counter, rejected derived-key overflow and future-dated
  observations before database work, restricted rule-definition listing to
  administrators, and rejected ambiguous serving rollout rows.
- Removed all quality digest/filter work from the disabled bid path after the
  capacity gate exposed the allocation regression. Blocking-action fixtures,
  false-positive rollback, appeal/billing, scope denial, and stale-snapshot
  behavior cover the corrected paths.

## Verification Result

- Go 1.23.5 full tests and vet passed in Aofei, Pzdesign, and Genelet. Scoped
  race suites passed for Aofei traffic quality/DSP/commands and Pzdesign
  `cmd/unify`/quality registry; Genelet's full race suite also passed.
- Pinned staticcheck v0.5.1 passed for Aofei and Pzdesign with the documented
  Pzdesign legacy style exclusions. Both template parsers checked 289
  templates; public-copy, public-data, documentation, baseline-SQL, actionlint,
  and all three repository `git diff --check` gates passed.
- Three clean disposable MySQL lifecycle runs passed after successive review
  fixes. The final run proved 88/6/43 restore, exact rollout, idempotent
  decisions, scoped review/appeal, enforcement/rollback, appeal-blocked A01
  Hold approval, immutability, bounded evidence prune, and clean connection
  state.
- The isolated Docker cache smoke passed with unique containers, ports,
  volumes, configs, and state; all artifacts were removed. The clean-room
  recovery drill passed at inventory
  `88:6:43` and the 100,000-row reporting benchmark passed at advertiser
  103/113 ms, publisher 104/115 ms, and operator 1673/1812 ms median/max.
- The post-review capacity baseline passed with quality disabled: ADX
  408.9–423.4 us/op, SSP 426.3–452.6 us/op, admission 4.82–5.09 us/op, and
  selection 187.0–201.0 ns/op. These are local regression measurements, not a
  production capacity or SLO claim.

## Reconciliation From O01

- IVT signals and enforcement reasons must remain distinct from O01 overload,
  dependency, timeout, and invalid-response outcomes. Reuse the fixed safe
  taxonomy rather than treating infrastructure failures as fraud.
- Throttle and quarantine controls must preserve O01 partner isolation and
  bounded metric dimensions; account- or partner-specific evidence belongs in
  access-controlled audit storage.

## Reconciliation From A01

- Fraud review may move Draft/Confirmed statements to Held where the A01 state
  machine permits, or create reasoned adjustments/corrections through distinct
  trusted actors. It cannot delete impression facts, immutable audit rows, or
  silently lower conservative D01 Redis floors.
- Reversal/exclusion rules must specify the original billable identity,
  `usd-cpm-impression-v2` amount, currency, rule version, evidence retention,
  and replacement-statement reconciliation before changing settlement totals.

## Reconciliation From D02

- Creative-policy violations and invalid partner markup are D02 validation
  outcomes, not fraud findings. S03 may consume their bounded counts as signals
  but must not duplicate permissive parsing or label infrastructure/format
  failures as IVT.
- A quality hold cannot reinterpret disabled CPC/CPA/ROI rows or alter the
  highest-CPM/within-unit rotation contract. Billing changes still require the
  A01 rule-version and correction path.

## Reconciliation From R01

- S03 may treat invalid action signatures, conflicting advertiser event ids,
  replay rates, impossible view/click/action sequences, and abnormal
  attribution ratios as bounded signals. It must not expose bearer lineage
  tokens, token hashes, auction ids, or action pseudonyms in public metrics or
  cross-account review evidence.
- R01 actions are analytical and immutable by advertiser/event identity. A
  fraud rule may flag or hold derived reporting, but cannot delete/rewrite the
  source fact or change CPM billing without the explicit A01 rule-version,
  adjustment/correction, evidence-retention, and appeal contract required by
  this milestone.

## Reconciliation From O02

- MySQL, Redis, NATS, cache, disk, DNS/proxy, and node failures are O02
  infrastructure outcomes, never fraud signals by themselves. Rules must
  distinguish missing/partial evidence from a valid zero and fail according to
  the documented serving/accounting boundary.
- Rule canaries and rollback preserve lifecycle readiness, N-1 capacity, and
  recovery compatibility. A rule-state restore retains version, decision,
  appeal, and billing linkage; expired raw evidence is not resurrected merely
  to make restored review data appear complete.

## Reconciliation From D03

- Partner timeout, malformed/gzip/oversize response, currency/floor/seat/id
  mismatch, unsafe creative/callback, and route or credential failure remain
  D03 interoperability, policy, or infrastructure outcomes—not IVT findings by
  themselves. A quality rule needs separate evidence and a named version.
- Middleman callback and billing signals must retain their D03 route generation,
  trigger, billable identity, accounting version, and retry/partial state. Any
  fraud hold or adjustment is canaried and reconciled through A01; it cannot
  rewrite the original partner callback or auction fact.

## Reconciliation From R02

- IVT and review outputs extend authorized R02 storage as separate rule-version,
  evidence-state, and decision dimensions; they never rewrite delivery,
  exposure, outcome, action, or accounting facts. Missing/partial evidence is
  not a valid zero and infrastructure/callback errors remain distinct.
- Experiment outcomes cannot be used to train or activate automatic fraud or
  bid changes. Any quality experiment declares a registry metric and guardrail,
  records only immutable pseudonymous exposure/outcome facts, and reports rule
  version plus false-positive/appeal context without exposing cross-account or
  raw identity evidence.

## Reconciliation From P02

- P02 `traffic_quality`, `source_quality`, placement, refresh, density, seller,
  and supply-chain fields are reviewed inventory declarations, not proof of
  valid or invalid traffic. S03 may compare them with observed behavior but
  must record a separate rule version and evidence state instead of rewriting
  the declared taxonomy or seller authorization.
- A missing, incomplete intermediary, or rejected `schain` is a transparency or
  partner-policy outcome unless a named rule has independent evidence. Seller
  ids and chains stay out of public metric labels and cross-account evidence;
  enforcement cannot change the A01 settlement owner implicitly.

## Reconciliation From S02

- Traffic-quality review and rule controls require named permissions for
  evidence read, scoped account disclosure, rule draft, canary, activation,
  quarantine, appeal resolution, and billing-hold recommendation. Analysts
  remain exact-grant read-only; publishers, advertisers, and partners can see
  only explicitly disclosed evidence for their own account.
- Blocking/quarantine activation, cross-account disclosure, and any action
  that can hold or adjust billing require a reauthenticated S02 human actor,
  safe reason, immutable security audit, and the separate A01 maker/checker
  transition. Rule-engine service identity is not a human session and cannot
  grant itself permissions or approve its own recommendation.
- Evidence stores use pseudonymous bounded identities and S01 retention. They
  never contain passwords, sessions, TOTP/recovery material, raw bearer
  tokens, or caller-controlled audit metadata. Tests must prove denial happens
  before evidence disclosure or enforcement mutation.

## Reconciliation From I03

- Invalid/expired management credentials, API quota rejection, dependency
  timeout, idempotency conflict, optimistic-version conflict, delayed cache
  publication, and commercial `New`/`Prepare` review are authentication,
  infrastructure, concurrency, or workflow outcomes—not IVT signals by
  themselves. S03 may consume only bounded aggregate counts under a named rule
  with independent traffic evidence.
- Generic advertiser API scopes cannot read raw fraud evidence, activate rules,
  quarantine traffic, resolve appeals, or recommend/approve billing holds. Any
  future quality API is a separate major contract with exact S02 permissions,
  account-safe evidence, recent MFA for enforcement, immutable audit, quotas,
  idempotency, and explicit publication state; it never returns bearer,
  idempotency-claim, or cache-generation tokens.
