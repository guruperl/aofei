# Deferred Product Investments

This document records W8M product investments that are intentionally outside
the active D/P/R/I/S/A/O milestone lanes. Deferral is not a claim that the work
has no value. It means the prerequisites, evidence, or risk controls needed to
justify implementation do not exist yet.

Deferred items do not reserve a status ID. When a reconsideration trigger is
satisfied, create the next unused ID in the owning lane and record the evidence
that opened it.

## Automatic Bidding And Machine Learning

**Why deferred:** Reliable automated optimization requires authoritative budget
enforcement, trustworthy conversion/action collection, stable attribution,
sufficient representative traffic, offline evaluation, bounded exploration,
and an operator rollback path. The current fixed CPC/CPA/ROI-to-eCPM factors are
not such a system and must not be described as machine learning or outcome
optimization.

**Current alternative:** Use manually configured CPM/eCPM campaigns, D01
delivery guardrails and deterministic pacing, D02 auction semantics, and
R01/R02 measurement and reporting. Controlled A/B reporting may be built in
R02 without allowing a model to change bids or budgets.

R01 now provides signed, idempotent action facts and deterministic same-lineage
attribution. That satisfies only the measurement-contract prerequisite; it
does not establish representative volume, causal lift, fraud resistance,
offline model quality, bounded exploration, or safe automated spend control.

**Reconsider when:** D01, R01, and R02 are complete; conversion volume and data
quality meet a recorded threshold; offline backtests demonstrate improvement
against a fixed policy; and operators have spend limits, canary controls, model
versioning, explanations, and immediate rollback.

## Internally Operated Payment-Card Processing

**Why deferred:** Collecting, storing, or directly processing payment-card data
creates PCI DSS, breach, fraud, chargeback, key-management, retention, and audit
obligations. The legacy schema includes fields capable of storing full card and
bank details and is not a safe payment-processing foundation.

**Current alternative:** A01 provides auditable manual invoicing and publisher
settlement. A02 now provides a disabled-by-default Stripe Checkout/Connect
hosted boundary so Aofei receives opaque provider identifiers and signed status
events rather than full payment credentials. A02 does not make W8M a card
processor and does not enable live mode by itself.

**Reconsider when:** A regulated business requirement cannot be met by hosted
or tokenized providers, qualified payment/security owners approve the scope,
and an independently reviewed PCI and incident-response program is funded.

## Multi-Region Deployment

**Why deferred:** Multi-region bidding multiplies MySQL, Redis, NATS, cache,
frequency-cap, callback, ledger, routing, failover, and consistency concerns.
The current deployment has not first established a measured single-region SLO
or proved that network distance is losing auction opportunity.

**Current alternative:** O02 builds multi-node HTTP availability, singleton job
ownership, dependency failover, backup/restore, and recovery exercises inside
one region.

**Reconsider when:** O02 is complete; production p95/p99 network latency consumes
a material portion of partner bid budgets; opportunity loss is measured by
region; and data-ownership, replication, reconciliation, and failover semantics
have an approved design.

## Million-Requests-Per-Minute Engineering

**Why deferred:** A headline throughput target is not useful without a defined
request mix, response SLA, hardware profile, cache mode, dependency latency,
and forecast demand. Premature technology swaps would add operational risk
without proving which component limits capacity.

**Current alternative:** O01 establishes reproducible local and production-like
load profiles, per-partner traffic controls, p50/p95/p99 latency, allocation and
dependency metrics, and a measured capacity envelope. Optimize the first
observed bottleneck using the existing performance roadmap.

**Reconsider when:** Forecast or observed sustained demand approaches the
measured safe capacity ceiling, the target request mix and latency/error SLO are
written down, and profiling identifies specific bottlenecks with before/after
acceptance criteria.
