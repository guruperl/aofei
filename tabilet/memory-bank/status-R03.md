# Status R03 - Experiment And Reporting Integrity

State: `[+]` Completed

## Goal

Prevent controlled-experiment pseudonym correlation and reject malformed or
mis-scoped analytical facts without allowing reports or experiments to mutate
bids, delivery, accounting, or settlement.

## Dependencies

- R01 attribution identity, R02 metrics/experiments, S01 privacy, and S02 scoped
  authorization.
- S05 owns verified request-principal provenance and creative/runtime trust
  boundaries. O03 owns maintenance/retention job reliability; A03 owns exact
  money sources.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Assignment namespace | `[+]` | The trusted create transaction rejects caller-supplied salts/algorithm versions, generates a fresh 16-byte salt, and stores assignment algorithm v2. V2 hashes a fixed domain plus experiment id, algorithm version, experiment version, decoded salt, and the 32-byte input pseudonym. Deterministic tests prove reused input and salt cannot link different experiment identities; only the result hash crosses storage. |
| Algorithm compatibility | `[+]` | `assignment_algorithm_version` defaults legacy rows to v1 while trusted creates store v2. Assignment dispatch preserves a literal v1 golden hash and uses the new domain only for v2; list/report output exposes the non-secret version but never the salt. Database triggers make the complete version contract immutable, enforce forward-only state transitions/reasons, reject variant update/delete, and reject additions after Draft, so a started experiment cannot change semantics or buckets in place. Mixed-version schema and runtime tests cover both paths. |
| Exposure and outcome validation | `[+]` | `Assign` binds stored owner scope and a private salt-derived proof to the experiment/version/algorithm/hash/variant/times. Exposure recording reloads and locks the exact database contract, validates owner, algorithm, variant, window, and calculated retention, preserves concurrent idempotency, and permits stopped-state retries only for an identical prior exposure. Outcome recording repeats the proof/scope/time checks against the immutable exposure and accepts only declared metrics and an exact idempotency tuple. Caller-built or altered assignments fail before storage; raw subjects/events remain transient. |
| Numeric and allocation safety | `[+]` | The metric registry now declares a value domain. Checked ratio derivation rejects negative/non-finite monetary sources and overflow while preserving valid repeated-click/action ratios above one; experiment outcomes reject NaN/Inf, negative zero, fractional/negative counts, negative CTR/CVR, ROI below -1, negative money/CPM/ROAS, and noncanonical DECIMAL input, with a matching baseline CHECK. The exact 64-bit modulo bucket spread is `10000/2^64` (about 5.42e-16), below the 1e-12 acceptance bound, so v1/v2 hashes and basis-point allocation remain stable. |
| Privacy, deletion, and operations | `[+]` | Unit and disposable-MySQL tests prove v2 unlinkability, idempotent facts, exact one-subject erasure without adjacent deletion or hash-bearing audit, and expiry removal of outcomes before exposures. Every experiment mutation runs through the fixed renewable Redis lease and its ownership context; a closed operation/outcome metric admits no resource data. Summer exports only per-variant aggregates and now omits stop reasons as well as salts, hashes, idempotency keys, audit reasons, and subject rows. The runbook documents additive v1 default migration, v1/v2 coexistence, stop semantics, and the required roll-forward response instead of letting old code reinterpret v2. |

## Acceptance Criteria

- Reusing caller input cannot produce the same stored subject hash across two
  new experiments for the same pseudonym and version.
- Existing experiments retain their assignments; a new algorithm is explicit,
  immutable, testable, and used only at a version boundary.
- Exposures and outcomes cannot name a nonexistent/wrong experiment, variant,
  metric, scope, or retention window, and invalid floating-point values never
  enter stored or exported analytics.
- Retention/deletion removes the intended pseudonymous evidence without raw
  identity disclosure or cross-account report access.

## Verification

- Cross-experiment correlation, legacy/v2 assignment stability, allocation,
  invalid-number, caller-forgery, idempotency, scope, retention, and exact
  deletion tests.
- Disposable MySQL migration and mixed-version integration, reporting
  benchmark, full Go/vet/staticcheck/race gates, docs/public-data checks, and
  diff hygiene.

## Review-Fix Gate

- Iteration 1: two P2 findings. First, the new numeric contract incorrectly
  assumed one click per impression and one action per click even though the R02
  formulas and retained facts permit repeated clicks/actions; valid CTR/CVR
  values above one were rejected. Second, the database guard protected only
  algorithm/salt/version updates and variant update/delete: owner, metrics,
  retention/window fields and a late variant insert could still alter or deny
  a started experiment under the same version. Both findings were corrected;
  the full iteration-2 review remains pending.
- Iteration 2: one P2 finding. The new late-variant trigger read experiment
  status without a locking read, so a concurrent Draft-to-Running transition
  could commit before the insert's foreign-key lock while the trigger had
  already accepted the stale Draft snapshot. The same direct transition also
  lacked the service's complete-allocation check. The trigger/transition guard
  now serializes on the parent row and repeats the 2-20/10,000-basis-point start
  invariant; a disposable two-connection race proves the late insert waits and
  fails after start. The full iteration-3 review remains pending.
- Iteration 3: one P2 finding. The required recovery drill still seeded its
  experiment as Running before inserting variants, so the strengthened Draft-
  only variant guard correctly rejected the fixture and the recovery gate
  failed. The fixture now follows the production contract (Draft, complete
  allocation, then Running) with Created/Started audit evidence; the full
  95-table/6-routine/61-trigger restore and post-restore prune passed. The full
  iteration-4 review remains pending.
- Iteration 4: clean. The whole R03 implementation, preceding fixes, schema,
  privacy/export boundary, failure semantics, compatibility, operations,
  tests, and documentation had no P1, P2, or higher finding. Two nonblocking
  coverage refinements also pin the legacy v1 variant (not only its hash) and
  require the new variant-insert/value-domain schema guards by name. The
  bounded review-fix gate passed in four iterations.

## Closeout Evidence

- `GOWORK=off go test ./...`, `go vet ./...`, pinned `staticcheck ./...`, and
  the documented race set passed; the full pzdesign test/vet/staticcheck and
  `go test -race ./...` gates passed. Genelet was unchanged by R03.
- A clean local rebuild matched `etc/step4_init.sql` at 95 tables, 0 views, 6
  routines, and 61 triggers. Disposable MySQL tests passed legacy-v1 default,
  v1/v2 fact binding, full version immutability, start/variant concurrency,
  numeric CHECK, idempotency, exact erasure, and expiry pruning; the Summer
  aggregate SQL integration passed against the same schema.
- The 100,000-row MySQL 8.0.41 benchmark passed at advertiser 105/143 ms,
  publisher 110/125 ms, and operator 1696/1880 ms median/max. The recovery
  drill passed a 95-table/6-routine/61-trigger backup/restore, experiment
  integrity/prune, cache rebuild, and callback-recovery proof in 36 seconds.
- Documentation, public-data, and diff-hygiene guards passed. No production,
  provider, credential, traffic, or other external system was mutated.

## Downstream Reconciliation

- A03 now starts from the current R03 61-trigger schema and must keep report
  and experiment facts analytical, preserve all experiment guards/value
  domains, and never promote an experiment metric into monetary authority.
- Conditional I02 now explicitly consumes the server-only v2 assignment,
  proof, aggregate-export, and deletion boundaries. No named mobile
  integration exists, so this reconciliation does not trigger I02.
- The evolution log needs no successor to v29: R03 implements the already
  approved milestone target without changing product ownership or direction.

## Exclusions

- Experiments remain observational. Automatic bidding, pacing, budget, or
  creative optimization remains deferred.
- Monetary source migration and statement access belong to A03.

## Reconciliation From S05

- Every authenticated report/export or experiment-control action must consume
  Genelet's typed principal for the exact component, action, permission, and
  resource, including the server session's MFA deadline where required.
  Compatibility `_g*` values, report parameters, request-proof headers, and
  analytics pseudonyms are never authentication or delegated scope.
- New report/experiment maintenance commands derive only their documented
  effective-Unix principal and exact health/retention permission, or use the
  reviewed UID-to-administrator mapping where the existing database schema
  requires a numeric actor. No actor flag, wildcard grant, or synthesized
  recent MFA may reappear.
- Analytical creative fields remain escaped protocol/source data. R03 must not
  add a preview, raw DOM sink, WebView renderer, or outbound URL fetch; doing so
  reopens S05's creative-consumer and special-address review before the new
  surface is implemented.

## Reconciliation From O03

- Experiment retention, reconciliation, and export jobs must run under the
  renewable lease-owned context, stop at the last confirmed ownership window,
  and retain database idempotency as their durable correctness boundary.
  Fixed-cardinality process metrics and durable evidence cannot contain raw
  subjects, event identifiers, salts, report filters, or account data.
- Any generated analytical file uses the shared durable writer and restricted
  directory/file modes. A future Redis/static analytical generation must use a
  completeness-marked shadow plus validation-before-mutation publication; it
  cannot reset or incrementally repopulate a live family.
