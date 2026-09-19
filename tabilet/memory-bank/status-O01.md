# Status O01 - Production Traffic Controls And Observability

State: `[+]` Completed

## Goal

Establish the protected metrics, traffic controls, alerting, and measured
capacity envelope required before expanding production marketplace traffic.

## Dependencies

- Foundation work with no new lane dependency.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Metrics boundary | `[+]` | `/debug/vars` now checks the direct peer against a validated CIDR allowlist before bounded dependency probes. The operator contract requires an edge deny, fixed dimensions, scrape ownership, retention, and no identifiers, URLs, consent, credentials, or bodies. |
| Partner controls | `[+]` | `TrafficGate` enforces exact configured ADX/SSP QPS, burst, concurrency, body, and timeout budgets; unknown ADX keys share the bounded default pool, while non-auction routes remain isolated. |
| Bid-path evidence | `[+]` | Fixed histograms publish counts, mean, approximate p50/p95/p99, and buckets for all required shapes. Middleman bidder outcomes distinguish fill, no-bid, invalid response, dependency error, timeout, overload, and configuration error. |
| Dependency evidence | `[+]` | Authorized scrapes report bounded Redis/MySQL probes, NATS state, and DB pool evidence; the runbook defines cache, audit, callback, ledger, disk, and singleton-lock checks. |
| Capacity baseline | `[+]` | The executable local baseline records hardware, configuration, fixed request mix, dependency state, validated errors, throughput/latency, and allocations. It explicitly records that local saturation is unclaimed and requires staging saturation evidence before a capacity promise. |
| Alerts and runbooks | `[+]` | The operator contract defines thresholds, ownership, escalation, staged canary/rollback, dependency/backlog checks, and incident actions under the S01 data boundary. |

## Acceptance Criteria

- Traffic exceeding a partner limit is bounded without destabilizing unrelated
  partners or internal/admin traffic.
- Capacity claims include hardware, configuration, request mix, latency/error
  SLO, and dependency state.
- Operators can distinguish policy rejection, no demand, dependency failure,
  timeout, invalid partner response, and overload.
- Metrics endpoints and dashboards do not disclose secrets or personal data.

## Verification

- Rate/concurrency/timeout/body-limit tests, representative load profiles,
  expvar/alert checks, race tests, failure injection, public-data scans, and full
  closeout gates.

## Exclusions

- Million-RPM engineering remains deferred in [docs/defer.md](../docs/defer.md).

## Reconciliation From S01

- Alerting and debug capture must preserve S01 redaction and log-retention
  boundaries. Capacity/load fixtures default to identifier-free contextual
  requests and use synthetic consent only in isolated policy tests.

## Completion Review

- Deep review corrected the timeout admission lifecycle so capacity remains
  owned until a timed-out handler actually stops, reset stale dependency/pool
  gauges when a dependency is absent, and added the missing fixed-cardinality
  invalid-partner-response classification.
- Configured partners have independent bounded state; unconfigured partner
  strings cannot allocate limiter entries or metric labels. The metrics
  boundary ignores forwarded headers and fails closed on an unusable allowlist.
- Local benchmark evidence is deliberately a regression baseline. Redis,
  uploaded-audience, middleman, compressed, overload, and saturation profiles
  remain mandatory staging gates in P01/I01/D03 before revenue expansion.

## Closeout Verification

- Go 1.23.5 full tests and vet passed in Aofei, pzdesign, and Genelet. Aofei
  full staticcheck, pzdesign staticcheck with established style exclusions,
  and Genelet staticcheck with established legacy exclusions passed.
- The documented Aofei race suite, pzdesign `cmd/unify` race test, and Genelet
  race suite passed. Rate, concurrency, timeout cancellation, streamed and
  declared body limits, partner isolation, protected metrics, Redis/MySQL
  dependency probes, invalid bidder responses, and fixed shapes have focused
  coverage.
- The reproducible capacity script completed three samples per profile on the
  recorded host. Both template parsers, the public-copy/data guard,
  documentation guard, and all three repository diff checks passed.
- No Docker stack, schema, cache publication, deployment, or external service
  was mutated. No evolution entry is required because O01 implements the
  existing operations boundary without changing product direction.

## Downstream Reconciliation

- I01, O02, D03, R02, I03, S03, and P01 now carry O01's per-process admission,
  protected fixed-cardinality metrics, capacity-evidence, response taxonomy,
  and staging-gate constraints.
