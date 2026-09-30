# Milestones

This file stays at milestone level. Detailed task rows live in the matching
`memory-bank/status-<lane><number>.md` files. Do not recreate an aggregate
`memory-bank/status.md` file.

## Status ID Pattern

The original single-digit M-lane history is normalized to `M00` through `M09`;
`M10` and later already satisfy the two-digit minimum. New product work is
organized into zero-padded domain lanes:

```text
D01, D02, ...  DSP demand, campaigns, auctions, creatives, and middleman bidding
P01, P02, ...  Publisher inventory, direct SSP, floors, and supply transparency
R01, R02, ...  Measurement, attribution, reporting, and experimentation
I01, I02, ...  OpenRTB integrations, management APIs, and mobile SDKs
S01, S02, ...  Privacy, identity, authorization, and traffic quality
A01, A02, ...  Accounting, billing, settlement, funding, and payouts
O01, O02, ...  Operations, observability, capacity, availability, and recovery
```

The two-digit minimum keeps each lane sorting naturally after it reaches `10`.
Always use the zero-padded form for new lane files, and never reuse an ID after
its status file exists. Cancelled work keeps its file and is marked cancelled.

## W8M Marketplace Roadmap

The W8M marketplace roadmap prioritizes commercial correctness, privacy,
accounting safety, and production controls before expanding automation or
scale. D01 through A02 in the original sequence below, D04, and D05 are
complete. A 2026-08-23 follow-up review opened D05 for post-D04 auction
compatibility and hot-path remediation; D05 is now complete. S06 repository
hardening, managed Cloudflare widget, constrained Free-plan edge rule,
production deployment, and live proof are also complete. P03, S05, and O03 are
complete; R03, A03, and the Review3 cross-lane M46 remediation are also
complete. I02 remains demand-gated and starts only when a named Android or iOS
integration supplies supported OS/version and lifecycle requirements. Matching
lane status files are the authoritative completion record; the early M-lane
milestones (M00 through M45) are retired to the
[history index](../docs/history/index.md). S07 account-identifier protection is now
in progress as a default-off successor to S02/S06; it does not activate or
migrate production by implication.
O04 is complete under the owner-directed deployment-boundary split: Aofei owns
a generic deployment/bootstrap engine, while exact W8M host policy remains
private. O04 neither depends on nor activates S07.
O05 is complete as O04's clean-bootstrap successor. It removes the generic
engine's dependency on a healthy legacy process while keeping each existing
target's migration procedure private. O05 neither depends on nor activates
S07.

Delivery sequence:

1. Foundation prerequisites: D01, S01, S04, and O01.
2. Foundation completion: A01 after D01, then P01 after D01/S01/A01/O01.
3. Core expansion: D02, I01, R01, and O02, then staged D03 after I01.
4. Product expansion: R02, P02, S02, and S03, then I03/A02 after S02.
5. Follow-up review remediation: D04 callback/runtime history, D05
   auction/cap/hot-path fixes, S06 repository plus production activation, and
   P03 direct-SSP authenticity, S05 runtime trust boundaries, and O03
   job/cache/filesystem reliability, R03 experiment/report integrity, and A03
   exact monetary sources and M46 cross-lane follow-up remediation are
   complete without reopening those completed source milestones.
6. Demand-gated mobile delivery: P03/S05/A03/M46 repository prerequisites are
   complete, but I02 starts only when a named mobile integration supplies
   supported-platform and lifecycle needs.
7. Current security migration: S07 protects interactive account identifiers
   through additive schema, dual reads/writes, opaque account-action tokens,
   and a separately authorized plaintext-retirement gate.
8. Independent deployment tooling: O04 replaces duplicated target deploy logic
   with a strict generic Aofei engine and synthetic state-machine tests.
9. Clean first activation: O05 replaces O04's legacy-process bootstrap bridge
   with an uninstalled-state transaction while preserving ordinary deploy and
   rollback behavior.

The strict serial order is:

```text
D01 -> S01 -> S04 -> O01 -> A01 -> P01
-> D02 -> I01 -> R01 -> O02 -> D03
-> R02 -> P02 -> S02 -> I03 -> S03 -> A02
-> D04 -> D05 -> S06 -> P03 -> S05 -> O03 -> R03 -> A03
-> M46
-> O04 -> O05
-> I02 (only after repository prerequisites and a named mobile integration)
```

S07 is an independent successor to M46 and may proceed concurrently with
O04/O05; it is not an O05 dependency and does not change I02's named-mobile-
integration gate.

Controlled direct-SSP and middleman staging may begin with existing runtime
features, but revenue-bearing activation must satisfy the prerequisite lane
acceptance criteria recorded in the corresponding status files.

| ID | State | Status file | Summary |
|---|---|---|---|
| D01 | Completed | [status-D01.md](status-D01.md) | Campaign delivery guardrails. |
| D02 | Completed | [status-D02.md](status-D02.md) | Auction, pricing, and creative correctness. |
| D03 | Completed; activation-gated | [status-D03.md](status-D03.md) | External DSP / AdX middleman activation. |
| D04 | Completed | [status-D04.md](status-D04.md) | Delivery, tracking, and auction integrity. |
| D05 | Completed | [status-D05.md](status-D05.md) | Post-D04 auction compatibility and hot-path remediation. |
| P01 | Completed; publisher activation-gated | [status-P01.md](status-P01.md) | Direct SSP commercial readiness and activation. |
| P02 | Completed | [status-P02.md](status-P02.md) | Supply metadata and seller transparency. |
| P03 | Completed; disabled by default | [status-P03.md](status-P03.md) | Direct SSP request authenticity, scoped App proofs, portal/cache integration, and rollout/rollback evidence. |
| R01 | Completed | [status-R01.md](status-R01.md) | Conversion, action, and attribution measurement. |
| R02 | Completed | [status-R02.md](status-R02.md) | Marketplace analytics and experimentation. |
| R03 | Completed | [status-R03.md](status-R03.md) | Experiment and reporting integrity. |
| I01 | Completed | [status-I01.md](status-I01.md) | OpenRTB partner interoperability. |
| I02 | Planned; demand-gated | [status-I02.md](status-I02.md) | Android and iOS publisher SDKs. |
| I03 | Completed; disabled by default | [status-I03.md](status-I03.md) | External campaign management API. |
| S01 | Completed | [status-S01.md](status-S01.md) | Privacy, consent, and data disclosure. |
| S02 | Completed; disabled by default | [status-S02.md](status-S02.md) | Identity, two-factor authentication, and RBAC. |
| S03 | Completed; disabled by default | [status-S03.md](status-S03.md) | Traffic quality and anti-fraud. |
| S04 | Completed | [status-S04.md](status-S04.md) | Template escaping and XSS audit. |
| S05 | Completed | [status-S05.md](status-S05.md) | Runtime trust-boundary hardening. |
| S06 | Completed; active on W8M | [status-S06.md](status-S06.md) | Public account abuse protection. |
| S07 | In progress; disabled by default | [status-S07.md](status-S07.md) | Account identifier lookup, encryption, rotation, and retirement. |
| A01 | Completed | [status-A01.md](status-A01.md) | Billing and manual settlement safety. |
| A02 | Completed; disabled by default | [status-A02.md](status-A02.md) | Hosted funding and publisher payout integration. |
| A03 | Completed | [status-A03.md](status-A03.md) | Exact monetary source migration. |
| O01 | Completed | [status-O01.md](status-O01.md) | Production traffic controls and observability. |
| O02 | Completed; production claims evidence-gated | [status-O02.md](status-O02.md) | Single-region availability, recovery, and SLO. |
| O03 | Completed | [status-O03.md](status-O03.md) | Job, cache, and filesystem reliability. |
| O04 | Completed | [status-O04.md](status-O04.md) | Generic immutable-release deployment and bootstrap engine. |
| O05 | Completed | [status-O05.md](status-O05.md) | Generic clean first activation from an uninstalled service state. |
| M46 | Completed | [status-M46.md](status-M46.md) | Review3 cross-lane correctness and hygiene remediation. |

Retired milestones and their frozen specifications, final status documents,
and review evidence live in the [history index](../docs/history/index.md).
Do not recreate a retired status file or specification here or in
`memory-bank/`; retired IDs stay reserved across active and retired storage.

Status markers:

| Symbol | Meaning |
|---|---|
| `[ ]` | Pending |
| `[+]` | Completed |
| `[~]` | In progress |
| `[!]` | Blocked |
| `[X]` | Cancelled |
| `[-]` | Closed Historical: a consumed failed attempt or superseded row retained for audit; it is never retried, does not block its accepted successor, and its notes name that successor. |

## Review Finding Severity

P1 and P2 are engineering review priorities, not product-domain lane priority,
milestone execution order, or status markers. When a narrower linked review
policy does not define them:

- **P1** is a severe defect in milestone acceptance, correctness,
  security/privacy, data integrity, or a public compatibility contract.
- **P2** is a material defect in supported behavior, reliability,
  compatibility, operations, or required verification/documentation that does
  not rise to P1.

Classify from impact, likelihood, and affected scope rather than implementation
or fix size. P1, P2, and any higher-severity finding block milestone closure and
cannot be carried into a later milestone. A lower-severity finding may be
carried only with a named pending owner and explicit rationale.

## New Review Intake

An engineering review received outside a milestone's closing gate is untrusted
planning evidence, not executable truth. Before implementing any of its
findings: revalidate each against current code; classify it (confirmed,
partially confirmed, resolved, duplicate, unsupported, outside ownership,
decision-dependent, or deferred); present the dispositions and owners for
approval; and record source and lineage in the owning status notes. Never
reopen completed milestone history because a later review concerns it — create a
remediation milestone with lineage instead.

## Milestone Review Procedure

After implementation and automated verification, review the complete milestone
for correctness, failure semantics, security/privacy, compatibility,
operations, tests, and documentation. The initial review is iteration 1. Record
each iteration number and its findings in the active status notes before fixing
anything. After every P1/P2-or-higher fix, rerun affected verification and
review the whole milestone again.

The gate passes only when an iteration finds no P1, P2, or higher-severity
issue. Run no more than 10 iterations, without resetting for a new session or
reviewer. If iteration 10 still has a blocking finding, leave the milestone
incomplete, mark it through the status file's blocked mechanism, report the
limit, and do not begin downstream reconciliation without explicit user
direction. [GOAL.md](../GOAL.md) owns the full multi-milestone execution
protocol.

## Closeout Checklist

Use this order when closing a milestone:

1. Run the milestone's required verification commands.
2. Update code-adjacent docs and the memory-bank files that changed behavior,
   contracts, tools, or operator workflow. Maintain `lessons.md` for applicable
   learning with evidence links; keep a still-applicable lesson even after its
   supporting milestone completes.
3. Pass the bounded milestone review gate. Resolve every P1/P2-or-higher
   finding; carry a lower-severity finding only with a named pending owner and
   explicit rationale in the matching lane status file.
4. Reconcile every affected pending status file and recompute the remaining
   dependency order according to [GOAL.md](../GOAL.md).
5. Check whether `evolution/` needs a new prompt/result version.
6. Mark the matching lane status tasks, review findings, and milestone state
   complete only after verification and a clean review-fix iteration pass.
7. Commit the milestone only when the execution request includes commit
   handling.

## M46 - Review3 Cross-Lane Remediation `[+]`

Resolved confirmed exact-money eligibility, middleman schema, filesystem
ownership, experiment concurrency, hot-path, creative-boundary, and
static-analysis findings without reopening completed source milestones or
weakening their fail-closed contracts. Detailed dispositions, verification,
and the two-iteration review gate are in [status-M46.md](status-M46.md).

## D01 - Campaign Delivery Guardrails `[+]`

Make budgets and schedules authoritative bid eligibility, not merely stored UI
or ledger data. Exhausted total/daily budgets and out-of-window campaigns or
items must not bid, including under concurrent traffic and stale-cache risks.
Detailed tasks and verification are in [status-D01.md](status-D01.md).

## D02 - Auction, Pricing, And Creative Correctness `[+]`

Align public pricing claims with runtime behavior, select the highest qualified
campaign effective CPM, reserve weights for creative rotation inside the
winner, complete native authoring, and validate media/size/secure markup.
Detailed tasks and verification are in [status-D02.md](status-D02.md).

## D03 - External DSP / AdX Middleman Activation `[+]`

The config-gated middleman path now has read-only database/Redis/credential
preflight, stricter topology and header safety, and a staged production runbook.
Fallback and optional `Always` require distinct evidence gates; checked-in and
deployed traffic remains off until a named partner is approved. Detailed tasks
and verification are in [status-D03.md](status-D03.md).

## D04 - Delivery, Tracking, And Auction Integrity `[+]`

Correct confirmed ACL SQL, callback publication/idempotency, CPM type, cap-time,
and bounded matching defects while preserving D01/A01 reconciliation and the
documented deterministic auction policy. Detailed tasks and verification are in
[status-D04.md](status-D04.md).

## D05 - Post-D04 Auction Compatibility And Hot-Path Remediation `[+]`

Correct legacy cap-time interpretation, isolate invalid capped demand without
weakening cache publication, restore compatible optional OpenRTB dimension
handling, and remove avoidable audience logging and macro-plan rebuilds from
the bid path. Preserve D04 callback, redirect, cap-lifecycle, and wire-format
contracts. Detailed tasks and verification are in
[status-D05.md](status-D05.md).

## P01 - Direct SSP Commercial Readiness And Activation `[+]`

Make configured publisher slot floors authoritative and prove the existing
browser and SDK-style `/pz` contracts from approved inventory through cache,
auction, tracking, reporting, rollout, and rollback. Detailed tasks are in
[status-P01.md](status-P01.md).

## P02 - Supply Metadata And Seller Transparency `[+]`

Implement the additive supply taxonomy selected by ADR 0001 and introduce
seller/supply-chain metadata without changing the existing publisher account
boundary. Detailed tasks are in [status-P02.md](status-P02.md).

## P03 - Direct SSP Request Authenticity `[+]`

Version direct-SSP inventory token integrity and add a distinct authenticated,
fresh publisher/App request boundary for SDK-style traffic. Public browser
locators remain replayable identifiers rather than publisher authentication.
The accepted threat and compatibility contract separates browser integrity,
browser provenance, SDK/server authentication, replay, compromise, rotation,
and inventory revocation while preserving active-cache authority.
The default-off `pz2` codec now binds complete inventory identity under an
epoch-selected current/previous HMAC key ring, supports measured dual reads,
and exposes an explicit legacy-disable gate without changing generated tags.
The independent default-off SDK/server gate now requires an App-scoped Ed25519
signature over the exact body and canonical request context, bounded freshness,
an immutable public-key snapshot, exact active-cache publisher/App scope, and a
shared one-use Redis nonce claim. S02-scoped lifecycle controls issue the
private value once, store only the public verifier, require named permissions
and recent MFA, and transactionally audit issue, rotation, and revocation.
Valid proofs remain subordinate to Web/App type, active inventory, App
identity, browser provenance, privacy, admission, media/size/floor, and
server-owned seller-chain policy. Pre-auction inventory and policy failures are
generic, and publisher-cache dependency failure is a retryable `503`.
Unauthenticated SDK compatibility requests are contextual before matching;
authenticated publisher-asserted body identity and coarse geo still require an
independent S01 grant. Uploaded-audience writers use a closed canonical marker
set with bounded legacy read/delete aliases.
Detailed tasks and verification are in [status-P03.md](status-P03.md).

## R01 - Conversion, Action, And Attribution Measurement `[+]`

Add signed, idempotent conversion and post-click action collection, attribution
semantics, privacy-aware retention, ledger integration, and advertiser-facing
measurement. Detailed tasks are in [status-R01.md](status-R01.md).

## R02 - Marketplace Analytics And Experimentation `[+]`

Expand reporting dimensions and commercial metrics, define reporting freshness,
and add controlled A/B assignment only after conversion attribution is
reliable. Detailed tasks are in [status-R02.md](status-R02.md).

## R03 - Experiment And Reporting Integrity `[+]`

Make new experiment assignment namespaces server-owned and cross-experiment
unlinkable, preserve existing assignments through an explicit algorithm
version, and reject malformed or mis-scoped analytical facts. Detailed tasks
and verification are in [status-R03.md](status-R03.md).

## I01 - OpenRTB Partner Interoperability `[+]`

Harden bounded gzip handling, OpenRTB 2.5 partner compatibility, sanitation,
floor/currency/media validation, rejection reasons, and response-time evidence.
Detailed tasks are in [status-I01.md](status-I01.md).

## I02 - Android And iOS Publisher SDKs `[ ]`

Build versioned native wrappers, sample applications, privacy propagation, and
release guidance around the stable `/pz` API after a named mobile integration
justifies maintained SDKs. Detailed tasks are in [status-I02.md](status-I02.md).

## I03 - External Campaign Management API `[+]`

Expose a versioned, scoped, quota-controlled, idempotent, and auditable API for
advertiser campaign management and reporting. Detailed tasks are in
[status-I03.md](status-I03.md).

## S01 - Privacy, Consent, And Data Disclosure `[+]`

Centralize consent interpretation, user-data use and disclosure, middleman
request sanitation, retention, deletion, and audit redaction across `/bid`,
`/pz`, trackers, and external fanout. Detailed tasks are in
[status-S01.md](status-S01.md).

## S02 - Identity, Two-Factor Authentication, And RBAC `[+]`

Add TOTP-based two-factor authentication, recovery controls, granular account
permissions, a read-only analyst role, session hardening, and security audit
events. Detailed tasks are in [status-S02.md](status-S02.md).

## S03 - Traffic Quality And Anti-Fraud `[+]`

Add explainable rule-based invalid-traffic detection, review/quarantine tools,
partner enforcement, and auditable counters without introducing automatic ML
decisions. Detailed tasks are in [status-S03.md](status-S03.md).

## S04 - Template Escaping And XSS Audit `[+]`

Audit every public and authenticated Summer/Genelet rendering path, inventory
intentional stored-markup previews, and centralize the narrow sanitized
safe-HTML boundary without weakening contextual `html/template` escaping.
Detailed tasks are in [status-S04.md](status-S04.md).

## S05 - Runtime Trust-Boundary Hardening `[+]`

Harden special-use address rejection, injected HTTP clients, creative consumers,
principal provenance, quality-rule version selection, and protected database
columns without stripping legitimate sandboxed ad scripts or blocking valid
state transitions. The completed result uses one DNS-rebinding-safe outbound
transport, a closed creative-consumer inventory, typed exact request principals,
restricted Unix maintenance identities, per-mode quality-rule selection, and
narrow enforcement/billing triggers. Detailed tasks and review evidence are in
[status-S05.md](status-S05.md).

## S06 - Public Account Abuse Protection `[+]`

Require scoped Turnstile verification before registration/recovery work, apply
atomic pseudonymous Redis quotas, derive client identity only through reviewed
trusted proxies, and layer a Cloudflare edge rate limit over public account
POSTs. Review2 reopened the trusted admin marker, anonymous error rendering,
Gmail MIME/concurrency, and quota-script tasks; all were remediated. The managed
widget, owner-selected Free-plan exact-path 10-second burst rule, production
service configuration, live provider/dependency proof, and rollback/restore
evidence are complete. The Free rule still cannot distinguish GET from POST.
Detailed tasks and verification are in
[status-S06.md](status-S06.md).

## S07 - Account Identifier Protection `[~]`

Protect advertiser, publisher, administrator, agent, and analyst identifiers
with a dedicated versioned lookup/encryption key ring while retaining bcrypt as
the sole password verifier. Roll out through additive schema, offline
backfill/verification, dual reads/writes, shared pseudonymous login throttling,
opaque action tokens, and an explicitly separate plaintext-retirement
migration. Detailed tasks and gates are in [status-S07.md](status-S07.md).

## A01 - Billing And Manual Settlement Safety `[+]`

Define charge/pay and CPM/eCPM accounting contracts, support auditable manual
invoicing and publisher settlement, and retire unsafe collection of full card
or bank credentials. Detailed tasks are in [status-A01.md](status-A01.md).

## A02 - Hosted Funding And Publisher Payout Integration `[+]`

Integrate hosted/tokenized external funding and payout providers with
idempotent webhooks, reconciliation, refund/chargeback handling, and secret
management. Aofei must never store full card or bank credentials. Detailed
tasks are in [status-A02.md](status-A02.md).

Result: default-off Stripe Checkout and Connect Express integration now uses
mandatory independently approved opaque bindings, maker/checker operation
states, stable provider idempotency, signed replay/order-safe webhooks,
connected-account isolation, exact Balance Transaction reconciliation, explicit
refund/dispute/payout exceptions, restricted maintenance, fixed-cardinality
metrics, and the A01 manual outage fallback. At A02 closeout the clean baseline
was 94 tables, 6 routines, and 55 triggers. Recorded/disposable verification is complete; live
provider sandbox, migration, governance, and production enablement remain
external go-live gates rather than repository completion claims.

## A03 - Exact Monetary Source Migration `[+]`

Exact money now extends through authoritative demand, reservation, ledger,
daily, management, statement, and hosted reconciliation sources using a
versioned, auditable migration that does not invent precision for historical
float data. Detailed tasks and verification are in
[status-A03.md](status-A03.md).

## O01 - Production Traffic Controls And Observability `[+]`

Operationalize protected metrics and alerts, add partner QPS/concurrency and
overload controls, record timeout/rejection/latency evidence, and establish a
repeatable capacity baseline. Detailed tasks are in
[status-O01.md](status-O01.md).

## O02 - Single-Region Availability, Recovery, And SLO `[+]`

Multi-node lifecycle readiness/failover, renewable singleton ownership,
durable ledger identities, dependency semantics, clean-room restore/cache
rebuild, recovery objectives, and an evidence-gated 99.9% SLO contract are
complete. Production 99.9% and provider RPO/RTO achievement remain explicitly
unclaimed until a named production window supplies retained evidence. Detailed
results are in [status-O02.md](status-O02.md).

## O03 - Job, Cache, And Filesystem Reliability `[+]`

Harden singleton renewal, atomic reusable cache publication, spread-generation
ordering, callback recovery evidence, filesystem permissions/atomicity, and
malformed geodata handling without weakening O02 split-brain safety. Detailed
tasks and verification are in [status-O03.md](status-O03.md).

## O04 - Generic Release Deployment And Bootstrap `[+]`

Move reusable immutable-release verification, strict environment validation,
preflight, atomic activation, rollback, history, and one-time bootstrap behavior
from the W8M realization into Aofei. New release bundles use a generic contract;
legacy W8M bundles remain read-compatible for rollback only. Exact host names,
paths, units, dependency identities, health policy, and history remain private.
Detailed tasks and review evidence are in [status-O04.md](status-O04.md).

## O05 - Generic Clean Bootstrap `[+]`

Replace O04's migration-oriented bootstrap bridge with a target-neutral first
activation from an absent release selection and uninstalled systemd unit.
Snapshot target-owned base configs, install and start one immutable release,
verify it, and restore an uninstalled state on failure. Migration from an
existing target service remains a private maintenance operation. Ordinary
deploy and verified rollback behavior do not change. Detailed tasks and review
evidence are in [status-O05.md](status-O05.md).

## Deferred Product Investments

[docs/defer.md](../docs/defer.md) records automatic bidding/ML, internally
operated payment-card processing, multi-region deployment, and million-RPM
engineering together with their current alternatives and evidence-based
reconsideration triggers. Deferred work has no reserved lane ID until its
trigger is satisfied.

## Historical Middleman Carry-Forward

- D03 owns the decision whether middleman routes need spread/local snapshots.
- A01 and A02 own invoicing and payment execution from `daily_mid` facts.
- Arbitrary downstream markup impression/click rewriting remains closed unless
  R01 identifies a measurement requirement that cooperative click notification
  cannot satisfy.


S08.4a offline retirement-aware deployment admission is accepted in the continuing cross-package owner checkout (local source acceptance 9967e6b, preparation review 1/10, no open P1/P2). This source branch publishes only that generic source/documentation prerequisite, without the unrelated layout ancestry. Release build, key recovery, live configuration, Identity canary and goal closure remain pending in the existing owner ledger; this branch is not a second execution owner.
