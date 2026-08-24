# Status R03 - Experiment And Reporting Integrity

State: `[~]` In progress

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
| Algorithm compatibility | `[+]` | `assignment_algorithm_version` defaults legacy rows to v1 while trusted creates store v2. Assignment dispatch preserves a literal v1 golden hash and uses the new domain only for v2; list/report output exposes the non-secret version but never the salt. Database triggers prevent algorithm/salt/experiment-version rewrites and any variant update/delete, so a running experiment cannot change buckets in place. Mixed-version schema and runtime tests cover both paths. |
| Exposure and outcome validation | `[+]` | `Assign` binds stored owner scope and a private salt-derived proof to the experiment/version/algorithm/hash/variant/times. Exposure recording reloads and locks the exact database contract, validates owner, algorithm, variant, window, and calculated retention, preserves concurrent idempotency, and permits stopped-state retries only for an identical prior exposure. Outcome recording repeats the proof/scope/time checks against the immutable exposure and accepts only declared metrics and an exact idempotency tuple. Caller-built or altered assignments fail before storage; raw subjects/events remain transient. |
| Numeric and allocation safety | `[+]` | The metric registry now declares a value domain. Checked ratio derivation rejects impossible counts, negative/non-finite sources, and overflow; experiment outcomes reject NaN/Inf, negative zero, fractional/negative counts, CTR/CVR outside 0..1, ROI below -1, negative money/CPM/ROAS, and noncanonical DECIMAL input, with a matching baseline CHECK. The exact 64-bit modulo bucket spread is `10000/2^64` (about 5.42e-16), below the 1e-12 acceptance bound, so v1/v2 hashes and basis-point allocation remain stable. |
| Privacy, deletion, and operations | `[ ]` | Prove cross-experiment unlinkability for v2, exact subject/experiment retention deletion, idempotent exposure/outcome recording, fixed-cardinality health, and privacy-safe export. Document migration, stop/rollback, and mixed-v1/v2 operation. |

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
