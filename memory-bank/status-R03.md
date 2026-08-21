# Status R03 - Experiment And Reporting Integrity

State: `[ ]` Planned

## Goal

Prevent controlled-experiment pseudonym correlation and reject malformed or
mis-scoped analytical facts without allowing reports or experiments to mutate
bids, delivery, accounting, or settlement.

## Dependencies

- R01 attribution identity, R02 metrics/experiments, S01 privacy, and S02 scoped
  authorization.
- O03 owns maintenance/retention job reliability; A03 owns exact money sources.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Assignment namespace | `[ ]` | Generate assignment salt only inside the trusted create operation and domain-separate the subject hash by experiment identity, algorithm version, experiment version, salt, and subject pseudonym. A caller-supplied salt must not create cross-experiment joinability. |
| Algorithm compatibility | `[ ]` | Store an explicit immutable assignment-algorithm version. Preserve existing v1 experiments and exposures until their bounded retention expires; new experiments default to the domain-separated version, and no running experiment changes buckets in place. |
| Exposure and outcome validation | `[ ]` | Bind recorded exposure/outcome fields to the stored experiment/version/variant/metric/retention contract, enforce account scope, and reject caller-built assignments or undeclared outcomes that cannot be proven consistent. Keep raw subject and event identifiers transient. |
| Numeric and allocation safety | `[ ]` | Reject NaN/Inf and out-of-contract ratios/metric values. Measure modulo allocation skew and replace it only if the acceptance bound requires unbiased sampling; deterministic assignment and documented basis-point allocation remain stable within an algorithm version. |
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
