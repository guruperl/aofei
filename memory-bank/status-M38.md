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
- `[+]` No open M38 review findings remain.

## Notes

- M38 treats Prebid Server as an external design reference, not a dependency to
  import.
- No Go test is required for M38 unless an implementation request adds runtime
  code changes.
- No `evolution/` entry was added because M38 records adoption candidates only;
  it does not change product direction, architecture boundaries, or public or
  private contracts.
