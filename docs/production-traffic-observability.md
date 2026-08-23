# Production Traffic Controls And Observability

This is the O01 operator contract for public auction admission, protected
metrics, dependency evidence, capacity baselines, alerts, canaries, and
rollback. It does not define a million-RPM target.

## Protected Metrics Boundary

`GET /debug/vars` is served on the unified listener but authorizes the direct
TCP peer only. `metrics_allowed_cidrs` defaults to IPv4/IPv6 loopback. Forwarded
headers are deliberately ignored, so a public client cannot spoof an allowed
address. Add only dedicated monitoring-node or private-network CIDRs.

The reverse proxy or Cloudflare route must also deny `/debug/vars` before it
reaches the service. Do not add a public Cloudflare Worker route for this path.
The scrape owner is the system-operator monitoring service; retain time-series
data for 30 days by default and incident aggregates for 13 months when policy
requires them. Dashboards may contain fixed route/outcome labels only—never
partner names, domains, URLs, request IDs, IPs, user IDs, consent strings,
credentials, or raw bodies.

Example direct-node check:

```bash
curl --fail --silent http://127.0.0.1:8080/debug/vars >/tmp/aofei-vars.json
```

A remote request receives `404`, including one with a forged
`X-Forwarded-For`. Scraping performs bounded 100 ms Redis and MySQL pings after
authorization and reads NATS connection state; public requests cannot trigger
those probes.

## Partner Admission Policy

The global policy applies to unlisted partners. Exact `adx:<path-domain>` keys
and the reserved `ssp` key can override individual fields:

```json
{
  "traffic_default": {
    "qps": 2000,
    "burst": 500,
    "max_concurrency": 256,
    "timeout_ms": 1000,
    "max_body_bytes": 1048576,
    "max_decompressed_body_bytes": 1048576
  },
  "traffic_partners": {
    "adx:exchange.example": {
      "qps": 500,
      "burst": 100,
      "max_concurrency": 64,
      "timeout_ms": 300
    },
    "ssp": {"qps": 1000, "burst": 200, "max_concurrency": 128}
  }
}
```

Zero override fields inherit the default. At most 256 exact partners may be
configured, preventing attacker-created limiter or metric cardinality. Listed
partners have independent token buckets and concurrency slots; unlisted ADX
paths share the bounded default pool. Admin, login, tracker, callback, and
metrics routes do not consume auction capacity.

`max_body_bytes` bounds bytes received from the peer. For `Content-Encoding:
gzip`, `max_decompressed_body_bytes` independently bounds decoded JSON;
unknown or stacked encodings are rejected. Successful JSON responses are gzip
encoded only when `Accept-Encoding` permits gzip (`q=0` never enables it).
No-bid `204` responses remain uncompressed.

Admission order and responses:

| Condition | Status | Behavior |
|---|---:|---|
| Declared or streamed body exceeds policy | `413` | body is bounded before JSON parsing |
| Unsupported or stacked content encoding | `415` | no auction parsing or matching |
| Malformed gzip stream | `400` | no auction parsing or matching |
| Decoded gzip body exceeds policy | `413` | decompression stops at the configured bound |
| Partner QPS/burst exhausted | `429` | immediate response with `Retry-After: 1` |
| Partner concurrency exhausted | `503` | immediate response with `Retry-After: 1` |
| Partner request time budget expires | `503` | context is cancelled; concurrency remains owned until the handler stops |

The protocol-wide maximum remains 1 MiB. `cmd/unify` also uses a 5-second
header timeout, 15-second read/write timeouts, 60-second idle timeout, and a
1 MiB header cap.

## Metric Contract

All map keys below are fixed source/outcome/reason categories:

- `aofei_traffic_requests_total`, `aofei_traffic_responses_total`,
  `aofei_traffic_rejections_total`, `aofei_traffic_in_flight`;
- `aofei_bid_path_latency_ms`: count, mean, approximate p50/p95/p99 and
  non-cumulative buckets for `adx`, `ssp`, `local`, `middleman`, `cap`,
  `audience`, `compressed`, `fill`, `no_fill`, `rejection`, and `overload`;
- `aofei_dependency_up`, `aofei_dependency_check_last_ms`,
  `aofei_dependency_check_errors_total`, and `aofei_db_pool`;
- `aofei_middleman_bidder_outcomes_total`: `fill`, `no_bid`,
  `invalid_response`, `dependency_error`, `timeout`, `overload`, or
  `configuration_error`, without a partner identifier;
- `aofei_middleman_bid_rejections_total`: fixed profile, endpoint, timeout,
  envelope, request-ID, currency, seat, impression, identity, price, floor,
  media, size, callback, markup, lateness, and fallback categories;
- `aofei_middleman_candidates_total`: fixed `considered`, `eligible`,
  `assigned`, `returned`, and `accepted` stages, plus
  `aofei_middleman_bidder_latency_ms` for bounded response-time percentiles;
- existing audit queue/drop/error, local-cache age/stale/reload, route-cache,
  tracking/cap, delivery reservation, SSP outcome, middleman callback, and S01
  privacy decision/invalid/blocked counters.
- S03 traffic-quality fixed counters:
  `aofei_quality_decisions_total`, `aofei_quality_matched_total`, the five
  `aofei_quality_action_*_total` outcomes,
  `aofei_quality_dependency_error_total`, `aofei_quality_rollback_total`, and
  the serving snapshot refresh/error/evaluation plus throttle/reject/quarantine
  counters. Rule, account, partner, event, and evidence ids never appear as
  metric labels; inspect per-rule false-positive health through the protected
  database/command surface.
- S06 public-account fixed maps:
  `aofei_public_account_submissions_total`,
  `aofei_public_account_turnstile_rejections_total`,
  `aofei_public_account_rate_limited_total`, and
  `aofei_public_account_dependency_errors_total`. Only four fixed actions,
  fixed quota scopes, and fixed dependency names are used; hostname, email, IP,
  account, provider error, and token values never become keys.
- A02 hosted-payment fixed counters cover provider requests/errors and webhook
  requests, invalid signatures, duplicates, applied/unresolved/ignored
  dispositions, plus newly recorded unresolved reconciliations. Provider,
  object, account, statement, event, operation, and token ids never become
  metric labels; inspect only authorized opaque details through the payment UI.

`compressed` records gzip decode work. A zero middleman/cap/compressed shape
means that request mix has not occurred, not that its latency is zero.

Operators can distinguish overload (429/503 admission counters), policy and
input rejection (4xx/fixed rejection counters), no demand (204/no-fill),
dependency failure (5xx plus dependency/error counters), invalid middleman
responses, timeouts, configuration errors, fixed partner-response reasons, and
successful fill.

Sampled partner diagnostics are disabled by default. Set
`openrtb_debug_enabled=true` and `openrtb_debug_sample_rate` in `(0,1]` only for
a bounded troubleshooting window. The default rate, when enabled without an
explicit value, is `0.01`. Captures contain a hashed request ID, internal bidder
ID, fixed outcome, counts, and elapsed milliseconds. They never contain raw
OpenRTB, consent, endpoint or callback URLs, credentials, or creative markup.

## Dependency And Backlog Evidence

- Redis/MySQL/NATS: alert from the dependency metrics above. Treat a configured
  dependency at `0` as unavailable; correlate Redis failures with cap,
  reservation, route, and tracking counters and MySQL failures with pool waits.
- Static cache: alert on `aofei_local_cache_stale=1`, reload errors, or age over
  the configured bound. D01 delivery-policy staleness is separately fail-closed.
- Audit: queue depth sustained above 75% of configured capacity, any growing
  dropped total, or recurring publish errors is actionable.
- Callback retry: run `cmd/mid-callback-retry -read -json`; alert when `due`
  grows for two intervals, `stale_processing` is non-zero twice, or any row is
  abandoned.
- Ledger: count unprocessed `winloss.<stamp>` intervals and compare the newest
  completed `ledger_log` interval with UTC time. Alert at two missed intervals.
- Files/disk: monitor the four configured log directories and spread root;
  warn at 70%, alert at 85%, and stop/correct ingestion before 95%.
- Singleton locks: inspect TTL/existence for `aofei:redis-cache`,
  `aofei:ledger`, `aofei:mid-callback-retry`, and `aofei:winloss`. A held lock
  beyond its owner timer plus grace is stale-owner evidence, not permission to
  delete it without checking the process.
- Hosted payments: run `cmd/hosted-payment -action=health`; alert on any
  approved operation whose statement is Held, stale `Submitting` or
  `Submitted` work, or an unresolved exception beyond the configured policy.
  Correlate with webhook/provider counters and the Stripe dashboard without
  copying raw event bodies, signatures, credentials, or payment details.
- Traffic quality: alert on rising dependency, snapshot refresh, or evaluation
  errors and treat them as availability incidents, never IVT. Alert on an
  expired enforcement snapshot and on any rule-health rollback recommendation.
  Serving retains the last valid snapshot only until its maximum age and then
  fails open. Compare action volume with case review and appeal outcomes before
  promoting a canary.

## Reproducible Capacity Baseline

Run `./scripts/aofei-capacity-baseline.sh`. It pins Go 1.23.5 by default and
prints time, CPU count, memory, kernel, request mix, dependency state, and three
benchmark samples with allocations.

Baseline recorded 2026-08-02 after S03 closeout on Linux x86-64, 8 logical Haswell CPUs,
24,021,612 KiB RAM, Go 1.23.5, local static snapshots, and no network
Redis/MySQL/NATS work:

| Profile | Observed range | Allocations |
|---|---:|---:|
| ADX, two local impressions, both filled | 408.9–423.4 µs/op | 327.3–328.7 KB, 1,899–1,900 allocs/op |
| SSP, two local ad units, both filled | 426.3–452.6 µs/op | 331.9–332.7 KB, 2,195–2,196 allocs/op |
| Accepted admission-gate request | 4.82–5.09 µs/op | 7.75 KB, 42 allocs/op |
| Parallel weighted selection | 187.0–201.0 ns/op | 0 B, 0 allocs/op |

These parallel benchmark averages are a same-host regression baseline, not
wire latency, p99, an availability SLO, or a production capacity promise.
Traffic quality was disabled for this run; its disabled path performs no event
digest or candidate-filter allocation. The increased end-to-end allocation
shape versus the earlier O01-only snapshot includes the subsequently completed
D/R/P/I control and reporting work and is the new comparison point, not a
capacity approval.
Every recorded operation returned its expected result; the benchmark fails on
an unexpected auction or admission result. Saturation was deliberately not
claimed by this dependency-free local run.
Redis cap, uploaded-audience, middleman network, compressed, no-fill, malformed,
and overload mixes must be measured in staging with the same report fields
before setting partner limits or making a capacity claim. Keep dependencies
healthy and raise traffic until either the latency/error objective is missed or
CPU, GC, Redis, network, or queue saturation is visible; record the first bound.

## Alerts, Canary, And Incident Actions

Initial actionable thresholds (tune only from measured traffic):

- page when dependency `up=0` for two consecutive scrapes;
- page when auction overload exceeds 1% for five minutes or p99 exceeds the
  partner timeout budget for five minutes;
- warn at audit queue 75%, page on drops; page on stale delivery/local cache;
- stop a quality canary and roll its enforcement back when its reviewed
  false-positive rate exceeds the named rule limit;
- page on sustained callback/ledger backlog or disk at 85%;
- warn on MySQL pool wait growth or in-use above 80% of max for ten minutes.

Canary one node with the intended config, verify fixed-cardinality metrics and
normal 2xx/204/4xx mix, then route 1%, 10%, 50%, and 100% only while error,
latency, dependency, queue, and spend evidence remains normal. Roll back the
node/config when overload or p99 breaches twice, a dependency counter rises,
or partner outcomes diverge from the control. Lowering QPS/concurrency is the
safe immediate containment; do not disable body/time limits or expose metrics.

Incident order: stop the canary increase; identify surface/outcome; verify
dependency and cache state; compare one unaffected partner; lower or isolate
the failing partner; preserve redacted metrics/log evidence; roll back; then
replay only documented retry/ledger workflows. Escalation owner is the on-call
system operator, with DSP maintainer support for bid-path defects and the
commercial owner notified for partner-specific throttling.

Regional availability, `/healthz` and `/readyz`, N-1 sizing, error-budget burn
rates, dependency-loss semantics, and recovery objectives are defined in
[single-region-availability.md](single-region-availability.md). The local
capacity baseline and recovery drill are regression evidence only; neither is
a production 99.9% availability claim.
