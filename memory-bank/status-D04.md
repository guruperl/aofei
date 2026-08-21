# Status D04 - Delivery, Tracking, And Auction Integrity

State: `[~]` In progress

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
| Local callback lifecycle | `[+]` | Win/loss now use token-owned processing claims: concurrent/completed duplicates have no side effects, successful publication finalizes once, and failed publication releases ownership. A loss reservation stays active across publish failure and is released only after the successful retry. |
| Middleman callback lifecycle | `[+]` | Added click idempotency and separate downstream-forward result versus local-publication markers for win/loss/bill/click. Local publish retries reuse the stored forward result without refiring downstream; in-flight/completed duplicates have no side effects. A failed post-forward state write deliberately clears ownership and retains the documented at-least-once downstream boundary. |
| Frequency-cap time contract | `[ ]` | Replace local-time reconstruction and 16-bit minute wrap with a versioned UTC representation whose range covers the configured retention. Preserve read compatibility through a bounded cache rollout and test negative, future, DST/timezone, 45-day, and 90-day boundaries. |
| Tracking and auction semantics | `[ ]` | Use `match.CostTypeCPM` rather than a numeric literal when parsing USD tracking prices. Recheck win/loss macro-signing limits, cap-without-period behavior, macro replacement determinism, and intentional equal-CPM ordering; change only behavior proven unsafe or inconsistent with the published contract. |
| Residual bid/matching safety | `[ ]` | Reproduce the exact-extension Native MIME limitation, demo nil-device panic, publisher-key randomness concern, and expired-reservation operator behavior. Correct actual runtime hazards, document intentional compatibility behavior, and avoid presenting non-secret identifiers as credentials. |
| Observability and operations | `[ ]` | Add fixed-cardinality retry, duplicate, claim-release, reservation, and cap-format evidence plus an operator migration/rollback note. No raw callback, user, auction, or inventory identifier may enter metrics or logs. |

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

## Exclusions

- Automatic bidding, equal-price traffic sharing, and new commercial price
  models are product changes outside this integrity milestone.
- Exact monetary storage migration belongs to A03.
