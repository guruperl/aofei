# Prebid/OpenRTB Pattern Adoption Review

M38 uses Prebid Server as a design reference for OpenRTB request handling,
bidder isolation, validation, debug visibility, and observability. It does not
make Prebid Server a runtime dependency, and it does not change `aofei` schema,
cache payloads, public APIs, bid behavior, or operator workflow.

The goal is to identify which patterns are worth adopting later and which
should stay out of `aofei` unless the middleman fanout product needs them.

## Prebid Flow Reference

The relevant Prebid Server flow is:

1. Parse a Prebid-flavored OpenRTB request and apply request-level controls such
   as debug, tracing, targeting, cache, currency, floors, bidder params,
   supply-chain metadata, and passthrough fields.
2. Resolve bidder-specific request shape before adapter calls. Prebid
   preprocesses `ext.prebid` controls into adapter-visible fields where
   applicable, maps bidder-specific params into bidder extensions, can place
   bidder-specific user IDs in `request.user.buyeruid`, and can place
   bidder-specific schain data on `source.schain`.
3. Call bidder adapters with an explicit bidder boundary. Adapter construction
   validates endpoint/configuration up front, adapter instances are reused, and
   runtime state must remain thread-safe.
4. Normalize and validate bidder responses before response assembly. Prebid
   records bidder response time, bidder errors, rejected responses, no-bid
   reasons, original CPM/currency after conversion or adjustment, and optional
   debug HTTP-call data.
5. Assemble OpenRTB responses with seat bids, targeting metadata, optional cache
   references, debug output, passthrough material, and non-bid status metadata.

Primary references:

- Prebid Server `/openrtb2/auction` endpoint:
  <https://docs.prebid.org/prebid-server/endpoints/openrtb2/pbs-endpoint-auction.html>
- Prebid Server annotated auction request example:
  <https://docs.prebid.org/prebid-server/endpoints/openrtb2/auction-request-example.html>
- Prebid Server Go bidder adapter guide:
  <https://docs.prebid.org/prebid-server/developers/add-new-bidder-go.html>
- Prebid Server caching overview:
  <https://docs.prebid.org/prebid-server/features/pbs-caching.html>
- Prebid Server activity controls:
  <https://docs.prebid.org/prebid-server/features/pbs-activitycontrols.html>

## Adoption Summary

| Area | Pattern | Classification | Recommendation |
|---|---|---|---|
| Performance | Timeout budget trimming across parse, local match, fanout, and response assembly | Adopt soon | Middleman already computes a fanout budget from upstream `tmax`, route, bidder, and config timeouts. A later cleanup should expose remaining budget consistently in logs/metrics and avoid starting work that cannot finish inside budget. |
| Performance | Lower-allocation OpenRTB marshal/unmarshal paths | Research/benchmark first | Prebid recommends optimized JSON helpers for high-volume adapter code. `aofei` should benchmark current `github.com/prebid/openrtb/v20` encode/decode and response marshal cost before adopting generated encoders, alternate JSON libraries, or pooling. |
| Performance | Debug capture controls for outbound HTTP calls | Only for middleman fanout | Capturing downstream request/response bodies is useful during bidder integration, but it should be explicitly gated and redacted because it adds allocations, privacy risk, and response-size pressure. Local campaign matching does not need this. |
| Matching | Per-impression normalization before selection/fanout | Adopt soon | `aofei` already derives `match.Attribute` per impression. A later cleanup should make the normalized per-impression object explicit enough to feed local matching, SSP formats, and middleman fanout without re-reading raw request fields. |
| Matching | Bidder/route isolation | Adopt soon | Prebid hides bidder-specific data from other adapters. `aofei` should keep middleman route selection and request shaping isolated per bidder so one advertiser endpoint cannot observe another endpoint's params, credentials, or route diagnostics. |
| Matching | Rich non-bid reason taxonomy | Adopt soon | Current no-fill reasons are hard to tune across local matching, SSP validation, and middleman fanout. Adopt an internal taxonomy modeled on seat non-bid concepts for timeout, blocked media, privacy/policy, below floor, invalid response, unsupported currency, and no eligible demand. |
| Matching | Request/response metadata for tuning | Adopt soon | Response time, candidate counts, floor/currency decisions, and selected rejection reason should be available in metrics or audits where safe. Do not put sensitive debug details into normal public responses. |
| Validation | Currency and floor validation | Adopt soon | Local bids are USD only and unsupported `bidfloorcur` no-fills. Middleman response validation should keep explicit floor/currency rejection reasons and preserve original CPM/currency in internal metadata when conversion or adjustment is ever introduced. |
| Validation | Bidder response validation | Adopt soon | Downstream OpenRTB responses should continue rejecting invalid, inactive, late, non-USD, wrong-imp, malformed markup, missing IDs, and unsafe callback URLs, with structured reason counters. |
| Validation | Creative size and secure-markup checks | Adopt soon | Local creatives already depend on configured size. Middleman and SSP response formats need explicit checks that returned width/height/media type are compatible with the impression and that markup/callbacks do not downgrade required HTTPS behavior when secure inventory is requested. |
| Validation | Explicit request sanitation | Adopt soon | Prebid distinguishes public request extensions from adapter-visible data. `aofei` should define a narrow sanitation boundary for forwarded middleman requests, especially user data, debug fields, internal route metadata, and passthrough extensions. |
| Security/privacy | Per-bidder request sanitization | Only for middleman fanout | Local matching is not an external disclosure boundary. Middleman fanout is, so each outbound request should be independently scrubbed and shaped for the selected bidder. |
| Security/privacy | Privacy/user-data scrubbing controls | Adopt soon | Direct SSP and middleman fanout should centralize rules for cookies, SDK user IDs, IP, UA, IFA, buyer IDs, and future consent fields. Keep the control local and explicit rather than copying Prebid's full activity-control framework. |
| Security/privacy | Schain and seller metadata checks | Research/benchmark first | Prebid supports bidder-specific schain handling. `aofei` should first define the publisher/seller metadata model from M34/M35 before adding runtime schain validation or outbound schain generation. |
| Security/privacy | Panic isolation around external fanout | Adopt soon | External fanout should not let a buggy adapter boundary take down the bid handler. `aofei` should recover at the bidder-call boundary, convert the failure to a structured bidder error, and keep other candidates alive. |
| Observability | Structured bidder errors and warnings | Adopt soon | Prebid exposes bidder errors/warnings in response extensions and debug output. `aofei` should put structured middleman errors in metrics/audits and optionally in gated debug output, not in ordinary exchange responses. |
| Observability | Response-time metadata | Adopt soon | Per-bidder response time is immediately useful for route tuning and timeout budgets. It should be tracked internally and optionally exposed to admin health views. |
| Observability | Non-bid reasons | Adopt soon | A shared reason vocabulary will make local no-fill, SSP policy rejects, and middleman rejects comparable. Public response exposure should be explicit and likely debug-gated. |
| Observability | Optional debug HTTP-call capture | Only for middleman fanout | This is valuable for onboarding and incident analysis, but it must be disabled by default, size-bounded, credential-redacted, and unavailable for normal production traffic unless a later milestone defines the operator controls. |
| Response assembly | Prebid ad-server targeting keys | Not applicable to aofei | `aofei` returns exchange-facing OpenRTB and direct SSP response formats; it does not assemble GAM/header-bidding key-value targeting today. |
| Response assembly | Prebid Cache for creative/VAST retrieval | Not applicable to aofei | `aofei` already has Redis/static cache for runtime configuration and returns markup directly. A Prebid Cache equivalent should not be added without a concrete video/rendering requirement. |
| Runtime architecture | Import or embed Prebid Server adapters | Not applicable to aofei | Prebid Server remains an external design reference. `aofei` owns its bidder endpoint model, route cache, ACL reuse, callback proxy, and ledger behavior. |

## Later Implementation Candidates

OpenRTB validation hardening milestone:

- Define the internal rejection taxonomy.
- Add structured reason counters for local no-fill, SSP rejection, and
  middleman rejection.
- Harden request sanitation and response validation around currency, floors,
  impression IDs, creative size, secure markup, and callback URL safety.
- Add tests with malformed multi-impression requests and bidder responses.

Middleman adapter-boundary/fanout cleanup milestone:

- Introduce a bidder-call boundary that owns panic recovery, per-bidder request
  shaping, debug capture, response-time measurement, and structured errors.
- Ensure bidder-specific params, credentials, schain/user data, and debug data
  never leak across selected bidders.
- Keep all controls behind the existing `middleman_enabled` and route
  eligibility gates.

OpenRTB performance benchmark milestone:

- Capture baseline CPU, allocations, and latency for ADX `/bid`, SSP `/pz`,
  and middleman-heavy traffic.
- Benchmark OpenRTB marshal/unmarshal alternatives in a contained package.
- Adopt alternate JSON, generated encoders, or pooling only after a measured
  before/after win and compatibility tests.

Supply-chain/privacy metadata milestone:

- Build on M34/M35 publisher ownership and future taxonomy decisions before
  adding schain, seller, consent, or user-data export policy.
- Define which metadata is stored in MySQL, cached, forwarded to bidders, and
  audited.
- Keep privacy controls explicit for browser `/pz`, SDK `/pz`, ADX `/bid`, and
  middleman fanout traffic.

## Non-Adoption Decisions

- Do not import Prebid Server or its adapters into `aofei`.
- Do not add Prebid Cache as a new runtime dependency.
- Do not add Prebid ad-server targeting assembly without a product requirement
  for header-bidding key-value output.
- Do not expose debug HTTP-call bodies in normal production responses.
- Do not change schema, cache payloads, response contracts, or operator
  commands as part of M38.

## Verification Scope

M38 is documentation and backlog tracking only. Required verification is:

```bash
./scripts/aofei-doc-check.sh
git diff --check
```

No Go test is required unless a later implementation milestone changes runtime
code, schema, cache payloads, config, or public response behavior.
