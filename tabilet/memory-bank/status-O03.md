# Status O03 - Job, Cache, And Filesystem Reliability

State: `[+]` Completed

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
| Redis cache primitive safety | `[+]` | Directly constructed Redis sinks no longer expose client/key fields; the reusable family constructor requires a non-empty generation namespace, while internal live item sinks reject family reset before Redis mutation. Raw-client RAdv compilation therefore cannot recreate the `DEL`-then-`HSET` window. Production calls the explicit `PublishRedisGeneration` API; private hash markers plus one server-side script validate every shadow before replacing the complete live generation. Miniredis guards prove rejection preserves live data, staged writes remain isolated, empty namespaces fail, build failure preserves the old generation, missing or partially recreated shadows preserve every old family, and complete swaps remove empty/obsolete families atomically. |
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

## Post-Closeout Release Reconciliation

- The 2026-08-25 W8M reload exposed that a standalone `unify` binary can be
  newer than its deployed Summer component metadata. The generic release
  boundary now requires one checksum-verified Aofei/Pzdesign/Genelet bundle
  containing the exact binary, config preflight, registered component JSON,
  templates, and static assets from clean published commits.
- Exact host paths, dependency image IDs, systemd scope, origins, health policy,
  atomic `current`-symlink activation, rollback, and credential-free deployment
  history are private infrastructure state. Release activation does not imply a
  schema/cache migration, feature enablement, Cloudflare mutation, or secret
  transfer.

## Bounded Review-Fix Gate

- Iteration 1 (2026-08-24): three findings remain open.
  - P2 resolved: `AcquireLock` now uses the exact millisecond PX duration,
    rejects a successful Redis response at or after the conservative deadline,
    and token-checks release before returning `ErrLeaseUncertain`. Delayed-
    acknowledgement repetition, race, and real-Redis ownership tests pass.
  - P2 resolved: `cmd/spread` now creates and flushes the generation receiver
    before fresh-node bootstrap, skips bootstrap when disk or subscribed state
    is already complete, and reconstructs Redis only while holding the exported
    cache-mutation lease used by every publisher. Bootstrap now compiles from
    MySQL rather than potentially older Redis cache payloads. Ordering and race
    tests pass.
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

- Iteration 6 (2026-08-24): six P2 findings remain open.
  - P2 resolved: when a normal renewal delay or retry backoff reaches the
    remaining confirmed window, the maintainer now waits for only half that
    window. A scripted slow-response test proves it makes a final successful
    attempt strictly before expiry instead of guaranteeing cancellation.
  - P2 resolved: the unused exported `ResetRedisStaticCaches` entry point is
    removed and its generic deletion helper is private and used only for
    isolated shadow cleanup. The generation publisher remains the sole
    supported full-family replacement boundary.
  - P2 resolved: complete spread publication now omits inactive and impression-
    capped publishers, matching Redis publication and legacy deletion
    semantics. A focused generation-builder test proves only publishable
    inventory is serialized.
  - P2 resolved: same-content MaxMind republishing now revalidates and retains
    the newest distinct rollback asset in addition to the selected digest;
    failed staging directories are ineligible. A focused repeated-publication
    test proves the prior valid asset remains available.
  - P2 resolved: attribute-log inventory discovery checks the lease context on
    every record and uses context-aware publisher/site/slot insert and collision
    lookup paths. A canceled-context test proves no inventory SQL is issued
    after lease cancellation; compatibility wrappers retain the old API.
  - P2 resolved: the exported legacy `SpreadGetPub` writer now accepts only the
    canonical publisher subject grammar and uses the shared durable writer with
    a `0750` parent and `0640` snapshot. Focused tests cover mode, content, and
    traversal rejection.

- Iteration 7 (2026-08-24): three P2 findings remain open.
  - P2 resolved: `begin` now prepares the higher generation completely before
    removing the prior active staging directory, and cleanup failure discards
    only the new root. An injected preparation-failure test proves the old
    manifest, files, and later commit remain consistent.
  - P2 resolved: fresh-node bootstrap rechecks committed state after acquiring
    the shared cache lease and compiles through the same MySQL spread-generation
    builder as publishers; Redis now supplies only the lease and monotonic
    sequence. A delayed spread-only delivery can therefore no longer be
    overwritten by older Redis payloads with a higher sequence.
  - P2 resolved: the supported route-only cache refresh now marshals both route
    versions before one atomic Redis `MSET`; nonempty distinct keys are
    mandatory. A focused Redis test proves both compatibility versions are
    published together.

- Iteration 8 (2026-08-24): one P2 finding was found.
  - P2 resolved: `DBGetAudiencesToCache` now carries the lease-owned context
    through item inventory, ACL construction, and targeting queries; middleman
    synthetic-audience compilation uses the same context-aware ACL path.
    Compatibility wrappers retain the old APIs, while focused canceled-context
    tests prove no audience SQL is issued after ownership cancellation.

- Iteration 9 (2026-08-24): two P2 findings were found.
  - P2 resolved: new publishers plus their default sites/slots and new
    site/slot pairs are now transactional. In-memory publisher inventory is
    updated only after commit, and a single-slot insertion records its map entry
    only after success. Focused failure tests prove rollback leaves neither a
    partial durable sequence nor phantom in-memory inventory.
  - P2 resolved: spread publisher packing, RAdv packing, and creative compile/
    sink loops now check the lease context. Focused cancellation tests prove
    those memory-only generation paths stop without packing later entries after
    ownership cancellation.

- Iteration 10 (2026-08-24): one P2 finding reached the original bounded review
  limit. On 2026-08-24 the user explicitly authorized at most five additional
  O03 review iterations (11-15); downstream reconciliation still requires a
  clean extended pass.
  - P2 resolved before iteration 11: fresh-node bootstrap called
    `receiver.install` after taking the
    renewable cache lease, but the full snapshot-write and pointer-commit loop
    did not accept or check the lease-owned context. Bootstrap installation,
    generation cleanup, and the atomic pointer replacement now carry that
    context; atomic replacement checks cancellation before rename but completes
    directory sync after any rename. Focused tests cancel after the first staged
    snapshot and prove no later file or generation pointer is installed, while
    a shared-writer test proves cancellation preserves the prior selected file.

- Iteration 11 (2026-08-24): one P2 finding was found.
  - P2 resolved: generation cleanup and pointer replacement now share a stable,
    context-aware cross-process file lock, and selection rejects non-increasing
    commits while holding that lock. Cleanup prunes only superseded lower
    generations so it cannot remove a concurrently staged successor, and a
    receiver reconciles its local floor when another process already selected a
    higher generation. Focused monotonic-selection, retention, cancellation,
    and race tests pass.

- Iteration 12 (2026-08-24): two P2 findings were found.
  - P2 resolved: every receiver now builds a same-sequence publication in a
    private staging directory, then renames that verified tree into the
    canonical generation while holding the exclusive selection lock. A lagging
    receiver therefore cannot erase another receiver's counted files, and
    focused overlap tests prove the selected snapshot stays complete.
  - P2 resolved: local-cache and diagnostic spread loads now resolve and read
    the complete snapshot while holding the shared side of the selection lock.
    Cleanup and pointer replacement retain the exclusive side; a deterministic
    lock test proves selection waits until the resolved read releases its root.

- Iteration 13 (2026-08-24): one P2 finding was found.
  - P2 resolved: the stable MaxMind sibling lock now lives in the shared
    package. JSON readers hold its shared side across both config parsing and
    opening the selected heap-backed City asset, while publication retains the
    exclusive side across staging, pruning, and config replacement. A focused
    concurrency test proves publication waits for the selected-asset reader;
    package, vet, static-analysis, and race checks pass.

- Iteration 14 (2026-08-24): one P2 finding was found.
  - P2 resolved: publishers create or tighten the stable sibling lock to
    `0640`, while readers open an existing lock read-only and never mutate the
    config directory. A static config with no generated lock loads
    optimistically; if a first publisher creates the lock during that load, the
    reader discards its result and repeats under the shared lock. Focused
    read-only-directory, first-publication retry, serialization, and race tests
    pass, and the operator contract documents the boundary.

- Iteration 15 (2026-08-24): one P2 finding reached the first user-authorized
  extension limit. On 2026-08-24 the user authorized at most five further O03
  review iterations (16-20); downstream reconciliation still requires a clean
  extended pass.
  - P2 resolved before iteration 16: `swapRedisStaticCaches` checked shadow-key
    existence before `MULTI`, then queued `RENAME` and `DEL` operations based
    on that stale observation. If an
    expected shadow disappears before `EXEC` (including under an allowed Redis
    eviction policy), its `RENAME` fails at execution time but Redis continues
    later queued operations, exposing a mixed live generation. A disposable
    Redis 7 proof left the first live key at `old1` while replacing the second
    with `new2` after the first queued `RENAME` returned `ERR no such key`; all
    uniquely named proof keys were removed. Every staged hash now carries a
    private completeness marker, and one Lua boundary validates all markers and
    both route keys before its first rename or deletion. Missing, evicted, and
    partially recreated shadows fail without changing live or obsolete keys;
    successful publication removes markers atomically so reader payloads remain
    unchanged.

- Iteration 16 (2026-08-24): clean. The full milestone review found no P1,
  P2, or higher-severity issue. Full Go tests, vet, staticcheck, the documented
  race suite, miniredis failure tests, a disposable Redis 7 `SCRIPT FLUSH`
  fallback proof, cache smoke, recovery drill, documentation/public-data
  guards, and diff hygiene passed. The user-authorized 16-20 extension stopped
  at the first clean pass.
