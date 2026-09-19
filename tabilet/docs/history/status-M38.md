# Retired milestone M38 - Prebid/OpenRTB Pattern Adoption Review

**Milestone.** M38
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M38.md
**Source specification.** memory-bank/milestone.md#m38---prebidopenrtb-pattern-adoption-review
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M38 - Prebid/OpenRTB Pattern Adoption Review `[+]`

Create a documentation-only review of Prebid Server OpenRTB patterns that may
be worth adopting in `aofei`.

Scope:

- Use Prebid Server as an external design reference, not a dependency.
- Summarize relevant OpenRTB flow: request parsing, bidder splitting, adapter
  calls, bid normalization, validation, targeting/cache/debug response
  assembly, and observability.
- Classify adoption candidates for performance, matching, validation,
  security/privacy, and observability as `Adopt soon`,
  `Research/benchmark first`, `Only for middleman fanout`, or
  `Not applicable to aofei`.
- Name later implementation milestones without changing runtime code, schema,
  cache payloads, config, public APIs, or operator workflow.

Acceptance:

- [docs/prebid-openrtb-adoption.md](../prebid-openrtb-adoption.md)
  records the review and deferred implementation candidates.
- Performance and dependency recommendations are measurement-gated.
- `memory-bank/status-M38.md` tracks the documentation tasks and verification.
- Required verification is `./scripts/aofei-doc-check.sh` and
  `git diff --check`.
````

## Status record

````markdown
# Status M38 - Prebid/OpenRTB Pattern Adoption Review

State: `[+]` Completed

Create a documentation-only milestone that records which Prebid Server
OpenRTB concepts are worth adopting in `aofei`, why they matter, and which
later milestones should implement them. M38 must not change runtime behavior,
schema, cache payloads, config, public APIs, or operator workflow.

## Tasks

- `[+]` Inventory Prebid OpenRTB concepts relevant to `aofei`.
- `[+]` Write `docs/prebid-openrtb-adoption.md`.
- `[+]` Classify concepts by performance, matching, validation,
  security/privacy, and observability.
- `[+]` Identify later implementation milestones without changing code.
- `[+]` Update README and memory-bank links.
- `[+]` Run documentation verification.

## Acceptance

- `[+]` The adoption document summarizes the relevant Prebid Server OpenRTB
  flow: request parsing, bidder splitting, adapter calls, bid normalization,
  validation, targeting/cache/debug response assembly, and observability.
- `[+]` Each concept is classified as `Adopt soon`,
  `Research/benchmark first`, `Only for middleman fanout`, or
  `Not applicable to aofei`.
- `[+]` Recommendations that affect performance or dependencies are
  measurement-gated.
- `[+]` Later implementation candidates are named without changing code,
  schema, cache contracts, config, or runtime behavior.
- `[+]` README and memory-bank milestone entries point to the new reference.

## Deferred Implementation Candidates

- OpenRTB validation hardening milestone.
- Middleman adapter-boundary/fanout cleanup milestone.
- OpenRTB performance benchmark milestone.
- Supply-chain/privacy metadata milestone.

## Verification

- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`

## Deep Review Findings

- `[+]` Tightened the Prebid request-shaping summary so it no longer implies
  every `ext.prebid` block is stripped before adapter calls. The review keeps
  the adoption guidance focused on bidder-specific preprocessing and
  sanitization.
- `[+]` 2026-05-17 follow-up review fixes: direct SSP publisher caches now omit
  inactive `pub_site`/`pub_slot` tuples, and `/pz` only trusts forwarded IP
  headers from configured `trusted_proxy_cidrs`.
- `[+]` No open M38 review findings remain.

## Notes

- M38 treats Prebid Server as an external design reference, not a dependency to
  import.
- No Go test is required for M38 unless an implementation request adds runtime
  code changes.
- No `evolution/` entry was added because M38 records adoption candidates only;
  it does not change product direction, architecture boundaries, or public or
  private contracts.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Inventory Prebid OpenRTB concepts relevant to `aofei`. | `[+]` | |
| Write `docs/prebid-openrtb-adoption.md`. | `[+]` | |
| Classify concepts by performance, matching, validation, | `[+]` | |
| Identify later implementation milestones without changing code. | `[+]` | |
| Update README and memory-bank links. | `[+]` | |
| Run documentation verification. | `[+]` | |
| The adoption document summarizes the relevant Prebid Server OpenRTB | `[+]` | |
| Each concept is classified as `Adopt soon`, | `[+]` | |
| Recommendations that affect performance or dependencies are | `[+]` | |
| Later implementation candidates are named without changing code, | `[+]` | |
| README and memory-bank milestone entries point to the new reference. | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |
| Tightened the Prebid request-shaping summary so it no longer implies | `[+]` | |
| 2026-05-17 follow-up review fixes: direct SSP publisher caches now omit | `[+]` | |
| No open M38 review findings remain. | `[+]` | |

````
