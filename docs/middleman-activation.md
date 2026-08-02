# External DSP / AdX Middleman Activation

This is the D03 operator runbook for activating an advertiser-owned OpenRTB
bidder in one W8M region. It turns on no traffic by itself. Database edits,
route publication, environment credentials, node rollout, and traffic gates
remain separate, reviewable actions.

## Safety Boundary

- The bidder belongs to an existing advertiser. Admin approval creates or
  verifies one inactive synthetic campaign/item/creative reporting chain owned
  by that advertiser.
- `credential_ref` is only a portable environment-variable name matching
  `[A-Za-z_][A-Za-z0-9_]*`, with at most 128 bytes. Its JSON header values
  remain outside MySQL, Redis, Git, command output, metrics, and UI.
- External disclosure requires both `middleman_enabled` and
  `privacy_contextual_middleman_enabled`. Every forwarded request is contextual.
- `Always` competition additionally requires `middleman_always_enabled`; it is
  never part of the first fallback canary.
- Partner calls use the fixed OpenRTB 2.5, USD CPM, timeout, body, endpoint,
  callback, and D02 creative-validation contracts. No activation exception may
  weaken them.

## Stage 0: Onboard Without Traffic

1. The advertiser creates the bidder endpoint under the advertiser portal.
2. An operator reviews endpoint ownership, exact OpenRTB 2.5 behavior, seat,
   timeout, privacy/retention/deletion obligations, incident contact, and
   prohibited raw request capture.
3. The operator approves the bidder with a scoped `credential_ref`. Confirm the
   synthetic reporting campaign/item/creative are inactive and form one
   same-advertiser chain.
4. Create a route group, bidder membership, and the narrowest publisher/site/
   slot/size target. Use `Fallback`, a bounded timeout, nonnegative margin, and
   low priority for the first canary.
5. Open `/goto/admin/g/midroute?action=health` and resolve every missing target,
   inactive bidder, credential-name, and synthetic-chain issue. This page never
   resolves or displays credential values.

Keep all three runtime gates false during onboarding.

## Stage 1: Publish And Run Read-Only Preflight

On the singleton cache node, publish only after the route review:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json \
  go run ./cmd/redis-cache -cache=routes
```

Then run the read-only check from each canary node's exact service environment:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json \
  go run ./cmd/redis-cache -validate-middleman \
  -activation-stage=preflight
```

The check rebuilds the active route model from MySQL, requires the published
Redis v2 checksum and database high-water mark to match, validates each partner
profile, and resolves every credential reference without printing header
values. It fails on an empty generation, stale/legacy publication, partial
synthetic chain, unsafe profile/header, missing credential, or incomplete
callback/signing config. Its manifest contains only counts, gate booleans,
route high-water time, and checksum.

`scripts/aofei-recovery-drill.sh` repeats this fallback preflight against a
restored synthetic route and a harmless environment header fixture in uniquely
named disposable MySQL/Redis containers. It proves recovery/preflight wiring;
the in-process partner suites prove network outcomes. Neither substitutes for
the named partner staging canary below.

Routes remain Redis-only. They are small shared operational state, change
independently of static publisher/demand snapshots, and already have bounded
refresh/error caching. Spread/local route snapshots would create a second
activation and revocation timeline, so D03 does not add them. Local/static bid
nodes still require Redis whenever middleman fanout is enabled.

## Stage 2: Fallback Canary

Use a reviewed config on one node outside the load balancer:

```json
{
  "middleman_enabled": true,
  "privacy_contextual_middleman_enabled": true,
  "middleman_always_enabled": false
}
```

Run the same command with `-activation-stage=fallback`. It additionally
requires both disclosure/runtime gates, requires `Always` to remain off, and
requires an active Fallback route. Only then add the ready canary node at 1%.

Use synthetic inventory and credential-free/partner-approved fixtures to prove:

- contextual per-impression request isolation and blocked COPPA/disabled
  disclosure fanout;
- valid Banner, Video, and Native fill; ordinary no-bid; floor/currency/seat/id
  rejection; timeout; malformed/gzip/oversize response; unsafe callback and
  markup rejection;
- local winner preservation on failure, and Fallback only after local no-fill;
- signed win/loss/bill/click proxying, billable-event idempotency, retryable
  callback queue/retry, and safe terminal 4xx behavior;
- complete interval/daily charge, pay, nonnegative margin, advertiser pay-side,
  publisher revenue, and A01 statement reconciliation.

Observe only fixed O01 metrics: admission/latency, local versus middleman
outcomes, validation reasons, privacy blocks, route refresh errors, callback
forward outcomes, retry backlog, audit drops, dependency state, and N-1
headroom. Retain the binary/config/cache versions, route checksum/high-water,
fixture mix, expected/actual results, and accounting query evidence. Never
retain raw bids, credentials, callbacks, consent strings, or creative markup in
the activation record.

Advance 1%, 10%, 50%, and 100% only while the O02 latency/error-budget and
capacity gates remain healthy. A staging fixture proves readiness, not a
production traffic or SLO claim.

## Stage 3: Optional Always Canary

`Always` is a separate commercial decision after Fallback acceptance. Add one
narrow `Always` route, enable `middleman_always_enabled` on one canary, and run:

```bash
GOWORK=off AOFEI=/etc/aofei/aofei.json \
  go run ./cmd/redis-cache -validate-middleman \
  -activation-stage=always
```

Prove a higher valid marked-up middleman CPM displaces a local winner and
releases its D01 reservation exactly once. Prove a lower, late, invalid, or
callback-setup-failed bid preserves the local winner. Reconcile both price sides
before expanding traffic.

## Disablement, Rotation, And Rollback

Immediate containment is to set either `privacy_contextual_middleman_enabled`
or `middleman_enabled` false and roll the canary config out with readiness
drain. This stops fanout without changing local eligibility. For one bidder,
deactivate its route membership/target or bidder, republish routes, verify the
new checksum on every node, and retain the database/audit history.

For credential rotation, create a new scoped environment reference, update the
approved reference, republish, run preflight from each canary service
environment, roll nodes, then revoke the old value. Never place an old secret
in a rollback file; rollback uses the prior reviewed reference only while it is
still valid.

Rollback order is: stop traffic expansion; disable the narrow gate or withdraw
the canary; restore the complete previous binary/config; republish the reviewed
route generation if route data changed; verify local auctions, callback backlog,
ledger freshness, and N-1 headroom. Do not replay callbacks or accounting facts
outside their idempotent commands.

## Activation Evidence

The approval record names the advertiser/bidder, partner owner/contact,
environment, binary/config versions, route checksum/high-water, target scope,
credential generation name (never value), privacy approval, fixture results,
metrics window, callback/retry results, charge/pay/margin reconciliation,
fallback/Always decision, rollback result, and final approvers. Production
revenue traffic remains off until every item has an owner and retained evidence.
