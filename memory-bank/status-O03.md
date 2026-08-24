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
| Geodata robustness | `[+]` | Legacy `.dat` loading now validates the 16-byte header, 32/24-bit offset arithmetic, complete index/prefix regions, non-overlapping ordered IP ranges, location bounds/minimum Geo size, unique bounded prefix ranges, and strict IPv4 input before lookup. Every slice uses checked 64-bit bounds, index zero is no longer confused with no-match, malformed fixtures/fuzz seeds return errors without panic, and legacy database export is ordered, size-checked, mode `0640`, and atomically replaced. Active JSON/MMDB remains preferred; only an explicit `.dat` `ips` suffix selects the compatibility reader, so malformed preferred data cannot silently fall through. `cmd/maxmind` now serializes on a stable sibling lock, hashes and atomically copies the City source into a content-addressed sibling generation, verifies copy stability and MMDB metadata, retains current/prior assets, then atomically switches JSON. Copy/validation/prune failure preserves the prior selection. |
| Retry and recovery evidence | `[+]` | Callback claims and terminal updates now require exact affected-row counts. The stable text/JSON report adds `forwarded` and `state_errors` and is emitted before a failed exit; an identifier-free typed error distinguishes state failure before versus after a guarded downstream attempt, preserving S05 request/dial address checks, proxy/TLS policy, redirect credential stripping, and bounded response draining. Durable `last_error` values are selected from a closed outcome vocabulary rather than copying URL-validation, DNS, dial, redirect, or response detail. Closed expvar maps cover lease, cache, spread generation, durable filesystem, and callback forward/state outcomes without resource/request labels; docs state their process-local scope and require timer exit/journal evidence. The disposable recovery drill restores a synthetic stale `Processing` claim and proves read-only visibility without forwarding. |
| Tool and documentation inventory | `[+]` | Tech-stack memory now names the imported `github.com/mileusna/useragent` v1.3.5 dependency instead of the unrelated retired package. The identity contract inventories the exact five interactive roles (`admin`, `adv`, `pub`, `agent`, and `analyst`) with their account IDs, and distinguishes public `web`, traffic-source `ssp`, Unix-UID maintenance actors, advertiser API credentials, and publisher App signers as non-account principals. Config and documentation tests reject added/missing role names, mismatched account identities, incomplete TOTP coverage, stale user-agent package names, or drift from the module imports. |

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

## Bounded Review-Fix Gate

- Iteration 1 (2026-08-24): three findings remain open.
  - P2 resolved: `AcquireLock` now uses the exact millisecond PX duration,
    rejects a successful Redis response at or after the conservative deadline,
    and token-checks release before returning `ErrLeaseUncertain`. Delayed-
    acknowledgement repetition, race, and real-Redis ownership tests pass.
  - P2 resolved: `cmd/spread` now creates and flushes the generation receiver
    before fresh-node bootstrap, skips bootstrap when disk or subscribed state
    is already complete, and reconstructs Redis only while holding the exported
    cache-mutation lease used by every publisher. Ordering and race tests pass.
  - P3 resolved: the legacy `.dat` exporter now records every `/8` crossed by
    an ordered IP range and extends each prefix's bounded index interval. A
    round-trip fixture proves lookups on both sides of an octet boundary.

- Iteration 2 (2026-08-24): two P2 findings remain open.
  - P2 resolved: the lease maintainer now derives its first renewal solely from
    the exact server TTL, with a nanosecond zero guard rather than a 10 ms
    floor. A short-lease test proves the first delay remains inside ownership.
  - P2 resolved: receiver startup now verifies that the pointed generation is
    a directory. A missing directory preserves its numeric sequence as the
    monotonic floor but remains unselected so bootstrap or a newer subscribed
    generation can repair it; a focused repair test passes.

- Iteration 3 (2026-08-24): one P2 finding remains open.
  - P2 resolved: every maintainer wait is now capped to the remaining confirmed
    window before sleeping, including the first wait after a delayed acquire or
    renewal response. A scripted late-response test proves cancellation occurs
    at, never after, the conservative deadline.

- Iteration 4 (2026-08-24): three P2 findings remain open.
  - P2 resolved: renewal responses are now timestamped and rejected at or after
    the conservative deadline before success can extend the local window. A
    scripted late-success test proves the work context is canceled as uncertain.
  - P2 resolved: `Release` now selects the maintainer channel against the
    caller-provided context and returns a fixed lease error when that wait
    expires; a deliberately stuck-maintainer test proves the bound.
  - P2 resolved: failed Redis-generation shadow cleanup now uses a five-second
    independent timeout, and a context-blocking client test proves cleanup
    returns at its bound so the command can proceed to lease release.

- Iteration 5 (2026-08-24): one P2 finding remains open.
  - P2 resolved: callback retry state now derives `last_error` from a closed
    forward-outcome vocabulary at both initial enqueue and every later retry;
    guarded URL, DNS, dial, redirect, response, and injected-client errors are
    no longer copied into durable evidence. Focused tests prove arbitrary
    detail maps to the generic outcome and a secret-bearing transport error is
    stored only as `callback request failed`.
