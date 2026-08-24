# Traffic Quality And Anti-Fraud

This guide describes the optional, explainable S03 traffic-quality controls.
They are designed for human-reviewed invalid-traffic handling, not opaque or
automatic fraud scoring. The feature is disabled by default.

## Signal Contract

Rules use a closed signal taxonomy:

| Signal | Meaning |
|---|---|
| `Replay` | The same measurement identity is observed more than once. |
| `ImpossibleSequence` | A bounded aggregate reports an invalid event order. |
| `InvalidOriginApp` | The observed Web origin or App identity conflicts with reviewed inventory. |
| `MalformedIdentity` | A measurement identity fails the documented format or lineage contract. |
| `AbnormalRate` | A reviewed event count exceeds a rule's bounded time-window threshold. |
| `AbnormalCTR` | Click-through rate exceeds a reviewed threshold. |
| `Automation` | A trusted detector reports bounded automation evidence. |
| `PartnerPolicy` | A trusted partner-policy check reports a violation. |

MySQL, Redis, NATS, cache, DNS, timeout, overload, malformed partner response,
credential, and other infrastructure failures are never fraud signals by
themselves. Missing or partial evidence is not a valid zero. Any rule evaluated
with incomplete evidence is reduced to `Observe`, including rules configured to
block or change billing.

The `traffic-quality` command accepts only bounded aggregate windows. It does
not accept raw HTTP requests, IP addresses, cookies, device identifiers, auction
identifiers, bearer tokens, or partner credentials:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/traffic-quality -action=assess-window < aggregate-window.json
```

Input is strict JSON, is limited to 64 KiB, and rejects unknown fields and
trailing content. Output contains decision ids, stable rule metadata, safe
reason codes, scope, evidence state, action, and billing disposition; it does
not return identity digests.

## Rules, Rollout, And Actions

A rule is versioned and immutable in meaning. Change behavior by creating a new
version, never by rewriting historical decisions. The permitted rollout graph
is:

```text
Draft -> Observe -> Canary -> Active
                   |          |
                   +-> Observe <-+
Draft/Observe/Canary/Active -> Disabled
Disabled -> Observe
```

During a version rollout, more than one mode for the same stable `rule_key` may
legitimately coexist: an older `Active` version protects current traffic while
a newer version advances through `Observe` and `Canary`. Runtime assessment
selects the highest version independently for each `(rule_key, rollout_mode)`
pair and evaluates them in deterministic `Active`, `Canary`, then `Observe`
order. Older versions in the same mode are ignored. Each selected mode creates
its own immutable decision, so canary/observe evidence remains reviewable but
cannot hide established Active behavior. When the newer version reaches
`Active`, it becomes the highest Active version; operators may then disable the
superseded row under the ordinary rollout workflow.

This selection never weakens the evidence gate. `Partial` or `Missing` evidence
forces every selected Active, Canary, and Observe decision to applied action
`Observe` and billing disposition `Observe`. Returned decision ordering is a
stable inspection contract, not permission for one lower rollout mode to
overwrite another decision.

Moving a blocking rule (`Throttle`, `Reject`, or `Quarantine`) from `Canary` to
`Active` requires selected canary decisions, completed human review, and a
false-positive rate no greater than the rule's configured limit. Enforcement
activation and rollback require a recent-MFA administrator, a named permission,
and a safe audit reason.

Actions have deliberately narrow meanings:

- `Observe`: record an immutable outcome; do not affect serving or billing.
- `Flag`: open review evidence; do not block serving.
- `Throttle`: suppress the matching scope when a reviewed enforcement is active.
- `Reject`: reject the matching scope and recommend exclusion from billing.
- `Quarantine`: suppress the matching scope and recommend a billing hold.

Serving consumes a bounded immutable enforcement snapshot. It is refreshed on
an independent timeout and retains the last valid snapshot after a refresh
error. Once its configured maximum age expires, serving fails open rather than
using stale enforcement. Publisher enforcement is checked before candidate
lookup, advertiser enforcement filters local candidates, and partner
enforcement filters external middleman assignments. Scope ids, partner ids, and
event identities are never metric labels.

## Review, Appeal, And Authorization

Advertisers and publishers can read explicitly disclosed evidence only for
their own account and may submit an appeal. Agents and analysts need exact
read grants; analysts remain read-only. Administrators manage rule versions,
rollout, review resolution, appeals, enforcement, rollback, and billing
recommendations. Sensitive changes repeat authorization in the domain service
and require recent MFA, even when the UI already checked the session.

Rules, decisions, case events, and audit records are immutable or versioned.
Raw bounded evidence is hidden after expiry and removed only through the
retention workflow. Review pages expose safe summaries and classifications,
not raw identity material.

## Billing Boundary

Traffic quality never rewrites measurement or accounting facts. A decision may
recommend `Observe`, `Exclude`, `Hold`, or `Reverse`, but exclusion and reversal
stop at a reviewed recommendation. A hold may move only an A01 `Draft` or
`Confirmed` statement to `Held`, using a separate checker and immutable
accounting audit. Released, adjusted, settled, or paid statements are not
silently changed. Corrections and money movement remain owned by the A01/A02
accounting contracts. Approval re-checks the live case: an open or upheld
appeal blocks the change, while only `InvalidTraffic` or a denied appeal remains
eligible.

## Storage, Privacy, And Retention

S03 adds nine tables: `quality_rule`, `quality_decision`, `quality_evidence`,
`quality_case`, `quality_case_event`, `quality_enforcement`, `quality_billing`,
`quality_counter`, and `quality_audit`. Twelve triggers protect rule versioning,
decisions, evidence retention, case history, enforcement and billing identity,
and audit history. S03 originally added ten triggers; S05 adds narrow
protected-update triggers for `quality_enforcement` and `quality_billing`.
The current clean schema contains 95 tables, 6 routines, and 61 triggers after
R03 adds three experiment-only immutability triggers.

Enforcement updates keep the originating rule, decision, scope, action, canary
allocation, creator, creation time, and expiry fixed; only a Canary or Active
row may enter `RolledBack` with complete rollback attribution. Billing updates
keep the decision, statement, billable digest, accounting version, disposition,
recommender, recommendation reason, and creation time fixed; only a Recommended
row may be independently Approved, Rejected, or have a Hold applied. Terminal
rows cannot be rewritten. These database guards supplement rather than replace
service permissions, recent MFA, maker/checker checks, immutable quality audit,
and A01 accounting audit.

Raw event and partner keys exist only long enough to derive HMAC-SHA-256
digests. Configure the digest key through an environment variable whose name is
stored in configuration. The key value must decode from base64 or hexadecimal
to at least 32 bytes. Never put the value in JSON, Git, logs, metrics, command
output, or review pages. Evidence retention is rule-bounded from 1 to 720 hours;
aggregate outcomes are retained from 365 to 2555 days.

Expired evidence can be pruned in bounded batches:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/traffic-quality \
  -action=prune-evidence \
  -limit=1000 \
  -reason='scheduled evidence retention'
```

The command derives `admin:unix-uid:<effective-uid>` from the operating system
and grants only `quality.retention.prune`; it has no actor flag, wildcard
permission, or recent-MFA claim. Run it only from a restricted maintenance
host after independent change approval and protect configuration/environment
access.

## Enablement And Rollback

Keep the feature disabled until the S03 schema is installed, S02 permissions
are assigned, an external secret is provisioned, and observe-mode results have
been reviewed:

```json
"traffic_quality": {
  "enabled": true,
  "digest_key_env": "AOFEI_TRAFFIC_QUALITY_DIGEST_KEY",
  "enforcement_refresh_seconds": 30,
  "enforcement_max_age_seconds": 120
}
```

Then:

1. Export a unique base64 or hexadecimal key through the service manager or
   secret store; do not add it to an environment file tracked by Git.
2. Create rules in `Draft`, promote them to `Observe`, and validate decision
   volume, incomplete-evidence rate, and reviewer agreement.
3. Canary blocking rules at a bounded percentage and review false positives.
4. Activate only after the canary health gate succeeds.
5. Roll a rule back to `Observe`, or disable it, if false positives rise.
6. Set `traffic_quality.enabled` to `false` and restart for a complete runtime
   rollback. Historical evidence and audit rows remain governed by retention.

For a populated S03 database, pause traffic-quality mutations, add the
`quality_enforcement_protected_update` and
`quality_billing_protected_update` triggers from `etc/step4_init.sql`, exercise
rollback plus approval/rejection/application through the service, and resume
only after direct protected-column rewrites fail. A new deployment installs
all twelve triggers with the reviewed additive migration before enablement.

Startup is intentionally strict when enabled: missing schema, an invalid key,
or an initial snapshot load failure prevents service startup. This avoids
claiming that enforcement is active when it is not.

## Monitoring And Incident Response

The process exports fixed-cardinality counters:

- `aofei_quality_decisions_total`
- `aofei_quality_matched_total`
- `aofei_quality_action_observe_total`
- `aofei_quality_action_flag_total`
- `aofei_quality_action_throttle_total`
- `aofei_quality_action_reject_total`
- `aofei_quality_action_quarantine_total`
- `aofei_quality_dependency_error_total`
- `aofei_quality_rollback_total`

Per-rule decision, selection, review, and false-positive health stays in the
access-controlled database instead of unbounded metric labels. Query a bounded
lookback with:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/traffic-quality \
  -action=health \
  -since-hours=24
```

For suspected false positives, first stop promotion, roll the affected
enforcement back, preserve rule/version and case ids, and compare complete
evidence with reviewer outcomes. For dependency or snapshot failures, treat
the event as an availability incident, not fraud; confirm the snapshot age and
database health, and allow the documented fail-open behavior. Billing holds
require a separate accounting review before release or correction.

## Verification

Run focused checks before enabling or changing the feature:

```bash
GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./trafficquality ./dsp ./cmd/traffic-quality ./etc
GOWORK=off GOTOOLCHAIN=go1.23.5 go vet ./trafficquality ./dsp ./cmd/traffic-quality ./etc
./scripts/aofei-doc-check.sh
git diff --check
```

Schema changes must also pass a disposable MySQL restore, the traffic-quality
lifecycle integration test, and the repository's documented schema-drift
workflow before production rollout.
