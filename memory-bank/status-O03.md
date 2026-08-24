# Status O03 - Job, Cache, And Filesystem Reliability

State: `[~]` In progress

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
| Renewable singleton liveness | `[+]` | Renewable locks now retry transient Redis errors with bounded exponential backoff only inside the last confirmed TTL, cancel the lease-owned work context immediately on token mismatch or at the conservative uncertainty deadline, and expose distinct confirmed-loss/uncertain errors. `WithLock` releases through a bounded independent context before returning work or lease failures; cache, ledger, callback-retry, and simulator commands all use it. Ledger SQL/file aggregation and simulator delays now honor the lease context. Scripted-clock tests cover recovery, mismatch, deadline stop, and explicit release; miniredis failure tests and disposable real-Redis renewal/exclusion/reacquisition pass. |
| Redis cache primitive safety | `[+]` | Directly constructed Redis sinks no longer expose client/key fields; the reusable family constructor requires a non-empty generation namespace, while internal live item sinks reject family reset before Redis mutation. Raw-client RAdv compilation therefore cannot recreate the `DEL`-then-`HSET` window. Production now calls the explicit `PublishRedisGeneration` API and retains one-transaction shadow-family replacement. Miniredis guards prove rejection preserves live data, staged writes remain isolated, empty namespaces fail, build failure preserves the old generation, and complete swaps remove empty/obsolete families atomically. |
| Spread generation proof | `[+]` | The race is real outside the narrow single-connection/singleton assumption: a receiver reconnect can miss reset/data messages, and separate producer connections can interleave. Full snapshots are now compiled before publication, assigned the monotonic Redis `aofei:spread:generation` fence, and sent as a bounded count/SHA-256 manifest, isolated data messages, and a flushed commit. `cmd/spread` stages each sequence, accepts idempotent duplicates, rejects changed entries, ignores stale/overlapping sequences, preserves the selected generation across reconnect gaps, retains current/previous files, and atomically switches `.aofei-current` only after manifest verification. Bootstrap uses the same disk-generation path; readers resolve one pointer before loading. Legacy direct subjects are receiver-first rollout compatibility only and are ignored after activation. Deterministic ordered, missing/reconnect, duplicate, overlap, stale-legacy, retention, publish-failure, counter-floor, pointer, Docker cache-smoke, and real-NATS reconnect tests cover the contract. |
| Filesystem safety | `[+]` | Spread roots/generation directories are created or tightened to at most `0750`, files use `0640`, and unsafe root/current-directory/relative-parent mutable targets are rejected. The ineffective flock on a process-private temp file is gone. A tested shared writer now performs encode/write, file sync, close, rename, and parent-directory sync; durable directory creation syncs each new entry, and spread pointers/snapshots plus generated MaxMind JSON use it. `city_file` may be explicit absolute but defaults nowhere: the checked-in relative value resolves against its JSON directory, while the generator also accepts `AOFEI_GEOLITE_CITY_FILE`. The audited supported geodata readers are heap-backed and retain neither mmap nor file descriptors; explicit opens remain scoped and closed. Permission, preservation-on-write-failure, temp cleanup, and path validation/resolution tests cover the contract. |
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
