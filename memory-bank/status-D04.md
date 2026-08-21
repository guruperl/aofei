# Status D04 - Delivery, Tracking, And Auction Integrity

State: `[+]` Completed

## Goal

Correct confirmed auction, audience, tracking, and frequency-cap defects without
weakening D01 delivery limits, changing the public OpenRTB or `/pz` response
shapes, or silently changing documented equal-price auction policy.

## Dependencies

- D01 delivery guardrails, D02 auction semantics, R01 measurement, and A01
  accounting identities remain authoritative.
- D03 middleman callbacks and O02 dependency-failure semantics constrain retry
  and fail-open behavior.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Finding disposition | `[+]` | Reproduced and classified the assigned claims below. Equal-CPM ordering and expired-reservation behavior remain documented policy; macro-time signature exclusions and exact Native MIME inference are compatibility constraints to harden and document without changing public callback or creative shapes. |
| Audience and input correctness | `[+]` | Balanced campaign-level publisher ACL SQL and covered the complete advertiser/campaign/item White/Black/Inherit precedence matrix. Banner, video, and Native dimensions outside `1..65535` now fail before cache lookup; nil audience/attribute/ACL inputs fail safely while valid empty audiences remain wildcards. |
| Local callback lifecycle | `[+]` | Win/loss now use token-owned processing claims: successful publication records completion before idempotent delivery work, failed publication releases ownership, completed retries repeat only delivery work, and budgeted in-flight requests remain retryable. A loss reservation stays active across publish failure and is released after a durable publication, including retry after post-publication Redis failure. |
| Middleman callback lifecycle | `[+]` | Added click idempotency and separate downstream-forward result versus local-publication markers for win/loss/bill/click. Local publish retries reuse the stored forward result without refiring downstream; dependency failures return `503`, while expired/malformed state remains `400`. A failed post-forward state write deliberately clears ownership and retains the documented at-least-once downstream boundary. |
| Frequency-cap time contract | `[+]` | Version-2 `BothCap` values retain the old 12-byte prefix and add authoritative UTC epoch-minute start/last fields. New readers accept old/new data, old readers consume the saturated prefix, and touched legacy state upgrades naturally. Negative/future time clamps safely; DST/timezone, 45-day, 90-day, and round-trip boundaries are tested. |
| Tracking and auction semantics | `[+]` | USD local and middleman callback records now use `match.CostTypeCPM`. Numbered caps without periods and out-of-range database values fail before cache publication, runtime Redis work, or measurement publication; standalone throttles remain valid. Macro replacement is longest-key/lexical deterministic with standard-key precedence. Tests preserve exchange-resolved win/loss macro compatibility and the documented campaign/item/advertiser equal-CPM order. |
| Residual bid/matching safety | `[+]` | Demo language extraction now handles nil requests/devices. Publisher helper IDs use cryptographic nonzero values and retry only confirmed primary-key collisions; `pub_id` remains a non-secret database identity. Required Native images without a recognized exact path extension fail closed without remote sniffing, and the operator contract preserves persistent budget after reservation expiry until evidence-led reconciliation. |
| Observability and operations | `[+]` | Added fixed-cardinality cap-format, callback-retry, claim-release, and middleman outcome metrics; removed raw request identifiers/errors from bid-path logs; documented rollout, retry, reconciliation, and rollback boundaries. The authorized follow-up fixed nil-demographic handling, restored the deterministic Docker bid fixture, passed a clean iteration-4 review, and reran the complete closeout gate. |

## Finding Disposition (2026-08-21)

The source audit below is the durable D04 input. It is intentionally complete
enough that implementation does not depend on an external review scratch file.

| Claim | Disposition | Owning task |
|---|---|---|
| Campaign-explicit publisher ACL SQL is missing a closing parenthesis when an item inherits | Confirmed defect; the sibling App predicate is balanced and the publisher predicate is not. | Audience and input correctness |
| Banner, video, and Native dimensions can truncate from signed 64-bit request fields to `uint16` | Confirmed defect; oversized and non-positive dimensions must fail before cache lookup. | Audience and input correctness |
| Nil audience/ACL inputs can panic or be treated inconsistently | Confirmed defensive-input defect; nil/empty lookups must return no match. | Audience and input correctness |
| Local win/loss notify-once state survives a failed NATS publication | Confirmed retry defect; it can suppress an unpublished fact and strand a losing reservation. | Local callback lifecycle |
| Middleman clicks lack replay ownership, while win/loss forwarding and local publication share one state | Confirmed retry/idempotency defect; downstream forwarding and local fact publication need independent completion state. | Middleman callback lifecycle |
| A failure after a middleman callback is forwarded but before durable retry-state update can redeliver | Accepted at-least-once compatibility boundary; retain it, expose bounded evidence, and document downstream idempotency requirements. | Middleman callback lifecycle; Observability and operations |
| Frequency-cap timestamps use `time.Local`, negative durations wrap, and 16-bit elapsed minutes cannot cover the 90-day retention contract | Confirmed time/range defect; introduce a versioned UTC representation with legacy reads during rollout. | Frequency-cap time contract |
| A numbered impression or click cap with a zero period is silently ineffective | Confirmed configuration defect; reject it before publication and make runtime evaluation fail safe. | Tracking and auction semantics |
| USD callback parsing assigns numeric cost type `1` although shared CPM is `match.CostTypeCPM` (`2`) | Confirmed accounting-semantic defect. | Tracking and auction semantics |
| Win/loss signatures omit exchange-resolved auction macros | Required OpenRTB compatibility constraint: unresolved macros cannot be signed at bid creation. Billing remains tied to separately signed impression callbacks; replay ownership must still bind the resolved bid identity and be observable. | Tracking and auction semantics; Observability and operations |
| Overlapping standard/custom macro replacement depends on Go map iteration order | Confirmed deterministic-output defect; use a defined longest-key/lexical order. | Tracking and auction semantics |
| Equal-CPM demand always follows the documented campaign/item/advertiser tie order | Confirmed behavior but deliberate published product policy, not an integrity defect. Traffic sharing requires a separately approved product change. | Tracking and auction semantics (test-only confirmation) |
| Native image MIME is inferred from an exact URL extension when explicit creative MIME is unavailable | Compatibility limitation, not content inspection; preserve exact allow-list matching and make rejection explicit. | Residual bid/matching safety |
| Demo language extraction can dereference a nil `Device` because of boolean precedence | Confirmed latent panic. | Residual bid/matching safety |
| `AddPub` uses non-cryptographic random publisher IDs | Confirmed collision/availability concern, not a credential weakness. Replace implicit random identity allocation without presenting publisher IDs as secrets. | Residual bid/matching safety |
| Expired delivery reservations do not automatically reopen total budget | Deliberate D01 fail-closed policy. Preserve it and document operator reconciliation rather than silently refunding spend/impressions. | Residual bid/matching safety; Observability and operations |

Geo-source precedence is owned by P03, callback/HTTP transport SSRF by S05,
cache publication primitives by O03, experiment numeric safety by R03, and
money storage by A03. Those findings are not duplicated into D04 behavior.

## Acceptance Criteria

- Every advertiser/campaign/item ACL inheritance combination produces valid SQL
  and the expected publisher/App audience.
- A transient NATS or Redis failure cannot permanently suppress an unpublished
  callback, double-publish a completed callback, or strand a losing delivery
  reservation.
- CPM tracking records use the shared CPM constant and reconcile through the
  A01 `CPM / 1000` boundary.
- Frequency caps remain stable in UTC throughout their supported lifetime and
  old/new cache data has a documented rolling compatibility path.
- Public request, response, Redis key, and measurement payload formats remain
  unchanged unless the status file is explicitly revised before implementation.

## Verification

- Focused ACL matrix, callback failure/retry/concurrency, reservation, cap-time,
  malformed-dimension, auction-policy, and tracking reconciliation tests.
- Disposable Redis/NATS/MySQL integration where the affected failure cannot be
  proved with an in-process fixture.
- Full Aofei and pzdesign tests, vet, pinned staticcheck, scoped race suites,
  documentation/template guards, benchmarks, cache smoke, and diff hygiene.

## Review-Fix Gate

- Iteration 1 (2026-08-21): three P2 findings. A signed redirecting click could
  still redirect after malformed packed tracking data because the redirect path
  treated every `serveStatus` error as a publication failure. Middleman local
  publication/billing markers were written as completed for the full callback
  TTL before NATS publication, so a process exit could suppress an unpublished
  fact; cleanup also inherited HTTP cancellation, and the forward-processing
  lease did not cover an arbitrarily configured callback timeout. Finally,
  `NewOpenRTBDemo(nil)` remained unsafe although the task evidence claimed nil
  requests were handled. These findings must be fixed and the full milestone
  reviewed again before closure.
- Iteration 2 (2026-08-21): two P2 findings after the iteration-1 fixes. The
  configured/default middleman callback context lifetime can end before the
  full accepted signature lifetime (including future clock skew) or before its
  processing lease. Also, typed local publication failures still become HTTP
  `400` on non-redirect callbacks even though the contract requires a retryable
  response. Enforce the TTL relationship and return a retryable server status,
  then review the full milestone again.
- Iteration 3 (2026-08-21): one P2 rollout finding. The checked-in local-runtime
  and recovery-drill config generators, plus two command fixtures, still emitted
  the old 86,400-second middleman callback TTL and would fail the new startup
  invariant. Update every maintained config producer/fixture to 86,700 seconds
  and rerun the cross-command verification before another full review.
- Iteration 4 (2026-08-21): one P2 cap-enforcement finding. A standalone
  positive impression throttle is valid configuration, but both the Redis
  requirement check and the cap-item loader considered only numbered
  impression/click caps. That allowed throttle-only demand to bypass its
  required state and made the throttle ineffective. Include throttles in both
  paths, cover the standalone form directly, and review the milestone again.
- Iteration 5 (2026-08-21): two P2 frequency-cap lifecycle findings after the
  iteration-4 fix. Generated impression/click URLs still omitted a standalone
  throttle because URL packing considered only numbered caps. Separately,
  bid-time expiry of a numbered impression window deleted the item's complete
  Redis `BothCap`, also erasing a still-active click window and independent
  throttle timestamp. Carry throttle state in signed tracking URLs and let the
  existing per-counter refresh logic reset expired windows without deleting
  sibling state, then review the full milestone again.
- Iteration 6 (2026-08-21): two P2 deadline-contract findings. The atomic cap
  event marker converted the signature deadline to relative Redis `EXPIRE`, so
  transaction latency could extend the marker beyond the exact accepted
  signature deadline. The controller's defensive middleman callback-TTL
  fallback also retained the old 24-hour value without the accepted five-minute
  future skew. Use an absolute marker expiry and align every fallback with the
  validated 86,700-second default before reviewing again.
- Iteration 7 (2026-08-21): one P2 operator-contract finding. The DSP workflow
  still instructed readers that an expired cap entry is deleted, contradicting
  the corrected shared `BothCap` lifecycle and risking loss of an active sibling
  click window or throttle timestamp. Document per-counter reset on the next
  accepted callback and idle-TTL retention, then review again.
- Iteration 8 (2026-08-21): one P2 validation-order finding. A capped callback
  decoded its packed auction-bid/user identity only after acquiring a replay
  claim, so a correctly signed but malformed payload could touch Redis before
  returning `400`. Decode every cap-mutation prerequisite before claim or cap
  state work, prove zero Redis calls, and review again.
- Iteration 9 (2026-08-21): one P2 Native-input finding. Native size extraction
  returned after the first usable image/video asset, so a negative or oversized
  dimension on a later requested asset could bypass validation before cache
  lookup. Validate every Native asset while retaining the first usable size,
  cover the multi-asset order, and perform the final bounded review iteration.
- Iteration 10 (2026-08-21): one unresolved P2 defensive-input finding. The
  Docker-backed `TestServeBidSmoke` loads a nonnil `demo.DemoAudience` while the
  request attribute has no demographic object; `DemoAudience.Has` dereferences
  that nil input and panics. This violates D04's nil-input acceptance. The
  ten-iteration review limit is exhausted, so D04 remains blocked and no
  downstream reconciliation or P03 implementation may begin without explicit
  user direction authorizing a new review cycle.

## Authorized Follow-Up Review Cycle

- Follow-up iteration 1 (authorized 2026-08-21): explicit user authorization
  permits remediation of the iteration-10 nil-demographic finding and a newly
  bounded review cycle. The original ten iterations above remain the historical
  bounded-gate record. Focused matcher tests passed and the panic disappeared,
  but the Docker bid smoke then returned `204`: its request fixture asked for
  Native while the deterministic sample's only App creative is explicitly
  Banner. This is a P2 closeout-fixture defect because the documented normal-bid
  proof cannot reach a `200` response. Align the request fixture with the seeded
  64x64 Banner without changing runtime media compatibility, then rerun affected
  verification and review the full milestone in follow-up iteration 2.
- Follow-up iteration 2 (2026-08-21): two P2 retry findings after the fixture
  correction restored the Docker smoke's expected `200`. First, a completed
  local impression/click/loss publication suppressed duplicates even when its
  Redis delivery finalization, click update, or loss release failed; retrying
  that side effect without republishing requires distinguishing a completed
  claim from an in-flight duplicate. Second, middleman local-publication failure
  still returned terminal HTTP `400`, while unavailable durable downstream
  retry cleared notify ownership but returned `204`; both statuses could prevent
  the caller from exercising the retry path. Preserve completed publication
  markers while retrying only delivery work, return `503` for retryable local or
  middleman dependency failures, retain `400` for malformed callbacks, rerun
  affected verification, and review the full milestone in follow-up iteration
  3.
- Follow-up iteration 3 (2026-08-21): two P2 dependency-boundary findings.
  First, failure to convert a successfully published local callback's processing
  claim into its completion marker returned before the delivery side effect; an
  immediate retry then treated that processing claim as a successful duplicate,
  which could strand an impression/click/loss update. Apply the idempotent
  delivery operation despite completion uncertainty and keep budgeted in-flight
  callbacks retryable until ownership resolves. Second, Redis failures while
  reading middleman callback or click context still returned terminal HTTP
  `400`; distinguish expired/malformed stored context from dependency failure so
  the latter returns `503`. Cover both paths, align the operator contract, and
  review the full milestone in follow-up iteration 4.
- Follow-up iteration 4 (2026-08-21): clean. The complete D04 diff was reviewed
  again for correctness, failure semantics, security/privacy, compatibility,
  operations, tests, and documentation after the iteration-3 fixes. No P1, P2,
  or higher-severity finding remains. The full closeout matrix below passed.

## Closeout Evidence (2026-08-21)

- Focused demographic, audience, tracking, delivery, and middleman callback
  tests pass, including nil demographics, post-publication delivery retry,
  uncertain claim completion, dependency/missing-state HTTP classification, and
  the unchanged populated-demographic match matrix.
- Go 1.23.5 passes `go test ./...`, `go vet ./...`, pinned staticcheck v0.5.1,
  and the complete scoped race suite from `AGENTS.md`.
- DSP benchmarks pass: two-impression `/bid` 385,460 ns/op, two-unit `/pz`
  402,396 ns/op, response marshal 1,751 ns/op, accepted traffic gate 5,089
  ns/op, and gzip traffic gate 53,358 ns/op. Parallel matcher selection remains
  zero-allocation at 181.3 ns/op.
- Aofei documentation and public-data guards pass. The sibling pzdesign checkout
  passes tests, vet, pinned staticcheck, 295 `.g`/`.e` template parses, public
  copy/data guards, the `cmd/unify` race suite, and diff hygiene.
- The Docker-backed cache smoke resets repository sample data, populates Redis
  and spread caches, and passes. The authenticated DSP `Test.*Smoke` suite then
  passes; its normal sample bid returns `200` and no nil-demographic panic.
- Final diff checks pass. `review-findings.md` remains untouched and untracked.
  No schema, public payload, cache format, deployment, production traffic, or
  external system was mutated. Evolution v27 already authorizes this exact
  remediation boundary, so no new evolution version is needed.

## Exclusions

- Automatic bidding, equal-price traffic sharing, and new commercial price
  models are product changes outside this integrity milestone.
- Exact monetary storage migration belongs to A03.
