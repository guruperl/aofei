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
