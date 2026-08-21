# Status D04 - Delivery, Tracking, And Auction Integrity

State: `[ ]` Planned

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
| Finding disposition | `[ ]` | Reproduce every assigned review claim and record confirmed, policy, compatibility, or rejected status before changing behavior. Keep deterministic equal-CPM ordering unless a separately approved product decision replaces it. |
| Audience and input correctness | `[ ]` | Fix the unbalanced campaign-level publisher ACL query and cover every advertiser/campaign/item inherit and explicit White/Black combination. Reject size/native dimensions that cannot be represented instead of truncating to `uint16`, and make nil/empty audience lookups fail safely. |
| Local callback lifecycle | `[ ]` | Make win/loss replay ownership follow publication outcome: duplicates have no side effects, a successful publish finalizes once, and a failed publish releases retry ownership. A loss reservation must be released exactly once after durable publication and remain retryable after a transient publish failure. |
| Middleman callback lifecycle | `[ ]` | Add click idempotency and separate downstream-notification state from local NATS-publication state. Retries must neither suppress an unpublished local fact nor resend an already successful downstream callback. Preserve the documented at-least-once boundary when a post-forward durable-state update fails. |
| Frequency-cap time contract | `[ ]` | Replace local-time reconstruction and 16-bit minute wrap with a versioned UTC representation whose range covers the configured retention. Preserve read compatibility through a bounded cache rollout and test negative, future, DST/timezone, 45-day, and 90-day boundaries. |
| Tracking and auction semantics | `[ ]` | Use `match.CostTypeCPM` rather than a numeric literal when parsing USD tracking prices. Recheck win/loss macro-signing limits, cap-without-period behavior, macro replacement determinism, and intentional equal-CPM ordering; change only behavior proven unsafe or inconsistent with the published contract. |
| Residual bid/matching safety | `[ ]` | Reproduce the exact-extension Native MIME limitation, demo nil-device panic, publisher-key randomness concern, and expired-reservation operator behavior. Correct actual runtime hazards, document intentional compatibility behavior, and avoid presenting non-secret identifiers as credentials. |
| Observability and operations | `[ ]` | Add fixed-cardinality retry, duplicate, claim-release, reservation, and cap-format evidence plus an operator migration/rollback note. No raw callback, user, auction, or inventory identifier may enter metrics or logs. |

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
