# Status O03 - Job, Cache, And Filesystem Reliability

State: `[ ]` Planned

## Goal

Remove avoidable singleton, cache-publication, spread-file, callback-retry, and
geodata failure modes without weakening split-brain protection or replacing
atomic generation publication with partial live writes.

## Dependencies

- M14 cache architecture, D03 callback retry, O01 observability, and O02
  single-region ownership/recovery contracts.
- D04 defines callback publication state; S05 defines safe outbound transport.
- Completed P03 defines immutable publisher-credential snapshot refresh and
  short-lived hashed replay-state contracts used by the HTTP tier.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Renewable singleton liveness | `[ ]` | Retry transient lease-renewal failures with bounded backoff inside the remaining lease safety window. Distinguish dependency uncertainty from token mismatch/confirmed ownership loss, release explicitly before fatal exit where ownership is still known, and never permit work after the safe lease window. |
| Redis cache primitive safety | `[ ]` | Prevent direct `RedisCacheSink` callers from exposing a `DEL`-then-`HSET` partial live family. Keep production shadow-family publication atomic and make the safe generation operation the default reusable API. |
| Spread generation proof | `[ ]` | Reproduce the alleged cleanup/write race with ordered single-producer, reconnect, duplicate, and overlapping-producer tests. If ordering and singleton ownership prove it impossible, document and close it; otherwise add an explicit generation/reset protocol so one refresh cannot delete another refresh's files. |
| Filesystem safety | `[ ]` | Replace 0777 directory creation and ineffective flock behavior, use atomic write/fsync/rename semantics where recovery requires them, close mmap/file resources, and validate paths without embedding environment-specific absolute locations. |
| Geodata robustness | `[ ]` | Bound every legacy `.dat` index before slicing, reject malformed input without panic, and make multi-file MaxMind publication recoverable rather than exposing a partially replaced generation. Preserve the supported `.mmdb` preference and documented fallback. |
| Retry and recovery evidence | `[ ]` | Make post-forward callback state failures visible under the documented at-least-once contract, add fixed-cardinality lease/cache/spread/filesystem metrics, and extend clean-room recovery and operator incident steps without capturing payloads or identifiers. |
| Tool and documentation inventory | `[ ]` | Correct the documented user-agent dependency and complete the current account-role inventory. Add focused guards where a stale package or role name would mislead deployment or security review. |

## Acceptance Criteria

- A transient Redis renewal error does not unnecessarily stop a singleton, and
  uncertain ownership never permits split-brain work.
- No supported cache writer exposes an empty or partial live Redis/static
  generation.
- Spread cleanup is either disproved with enforceable assumptions or corrected
  with generation-aware ordering under reconnect and overlapping publication.
- Malformed geodata and filesystem errors return bounded errors rather than
  panics, leaked mappings, world-writable state, or partial replacement.

## Verification

- Fake-clock lease tests plus disposable Redis ownership/failure integration.
- Redis/spread ordering, duplicate, reconnect, overlapping-generation, crash,
  permission, fsync, and malformed-geodata tests.
- Cache smoke, recovery drill, operational command tests, full Go gates, scoped
  race suites, documentation/public-data checks, and diff hygiene.

## Exclusions

- JetStream/KV cache distribution, multi-region coordination, and million-RPM
  engineering remain deferred unless their existing evidence triggers are met.

## Reconciliation From P03

- P03 credential reload keeps the previous complete immutable generation on
  query/build failure, fails request authentication closed after its bounded
  maximum age, and serializes query/build/install with local lifecycle
  mutation. O03 cache/job abstractions must not replace that with partial
  in-place publication or allow stale reload completion to undo a revocation.
- P03 replay keys contain only a domain-separated hash of public id plus nonce
  and expire through the accepted timestamp window. Recovery tooling must not
  scan, export, persist, or bulk-delete them; dependency failure remains a
  fixed-cardinality retryable authentication outcome rather than a
  credentialless fallback.

## Reconciliation From S05

- Callback retry and any reusable outbound job client must preserve S05's
  mandatory `safehttp` URL/dial boundary. Injected clients cannot restore a
  proxy, custom dial path, insecure TLS, HTTPS downgrade, unsafe redirect, or
  cross-authority credential; an injected redirect hook is checked against an
  immutable pre-hook history. Socket-free test doubles remain explicitly
  marked and URL-validated.
- Retry/recovery evidence may record only fixed-cardinality disposition and
  dependency state. It must not persist callback URLs, arbitrary headers,
  request bodies, DNS answers, credential refs, or redirect history in spread
  files, caches, metrics, or incident artifacts.
- O03 command/tool cleanup must retain the S05 maintenance boundary: restricted
  retention work derives its effective Unix principal and exposes no caller-
  selected actor or wildcard/recent-MFA claim. Reliability retries cannot turn
  an offline health/retention principal into HTTP or financial authority.
