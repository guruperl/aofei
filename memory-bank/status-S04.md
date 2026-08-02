# Status S04 - Template Escaping And XSS Audit

State: `[+]` Completed

## Goal

Complete the page-by-page escaping audit carried forward from M18 and prove
that public and authenticated Summer/Genelet pages cannot turn untrusted stored
or request data into executable markup.

## Dependencies

- S01 data classification and retention rules inform which stored values are
  sensitive or user-controlled.
- The sibling `../pzdesign` template and static-asset tree is in scope.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Rendering inventory | `[+]` | `../pzdesign/docs/rendering-security.md` inventories public account/mail, login/error, advertiser, publisher, agent, admin, report/chart, creative review, controller/filter/renderer, and direct SSP browser delivery entrypoints and data sources. |
| Contextual escaping audit | `[+]` | Ordinary values remain under `html/template`; hostile fixtures cover HTML text, attributes, URL filtering, inline event strings, report JavaScript values, and mail. Deep review replaced four decoded modal-title `.html()` sinks with `.text()`. No dynamic CSS context remains. |
| Stored-markup boundary | `[+]` | Advertiser, publisher, and agent management/review templates display creative markup and URLs only as escaped source. Genelet's fixed, contextually rendered CSRF hidden input is the sole trusted `template.HTML` conversion. The named `www/js/ads.js` delivery sink remains isolated as the P01/D02-owned `/pz` contract. |
| URL and attribute safety | `[+]` | Remote scripts/styles/frames and legacy IE shims were removed or replaced with reviewed local assets; unsafe static schemes and `javascript:void(0)` were removed. Genelet tests cover external, protocol-relative, backslash, unsafe-scheme, encoded-local, and script-root redirect cases. |
| Regression fixtures | `[+]` | Hostile strings now render through advertiser/publisher login, advertiser/admin account and campaign pages, publisher/agent creative review, JavaScript reports, and advertiser/publisher mail without injected markup, events, scripts, or unsafe URL attributes. |
| Guard tooling and docs | `[+]` | The template checker rejects assembled queries, unsafe schemes, remote executable/embedded resources, template data in fetch/execution elements, raw DOM page sinks, raw template types including aliased imports, and dot imports. Aofei, pzdesign, Summer UI, and Chinese content docs record the convention. |

## Acceptance Criteria

- Every intentional raw/stored HTML rendering path is inventoried, justified,
  sanitized or trusted at a named boundary, and covered by a regression test.
- Ordinary request, account, campaign, publisher, report, and error data stays
  under contextual `html/template` escaping.
- No template or helper can introduce `template.HTML`, `template.URL`, inline
  script, or equivalent raw output outside the approved boundary.
- Public and authenticated hostile-input fixtures do not execute markup,
  scripts, event handlers, or unsafe URL schemes.

## Verification

- Focused Aofei Genelet/Summer renderer tests.
- Full pzdesign package tests and both `.g`/`.e` template parsers.
- Staticcheck and scoped security regression tests in both repositories.
- Aofei documentation/public-data checks and both repository diff-hygiene
  checks.

## Exclusions

- This milestone does not rewrite third-party ad creative markup. D02 owns
  creative validation, and R01 may reopen cooperative measurement behavior
  only for a concrete product requirement.
- A site-wide CSP rollout requires a separate compatibility milestone if S04
  findings show it can be introduced without breaking supported creatives.

## Deep Review

- The first review found that server-escaped campaign/site/slot names were read
  from `data-title` and inserted with jQuery `.html()`, which decoded them back
  into executable DOM. All four modal-title paths now use `.text()`, and the
  template source guard rejects future raw DOM page sinks.
- The review also strengthened raw-template scanning to resolve aliased
  `html/template` imports and reject dot imports, broadened dynamic embedded
  resource detection beyond fields named `content`, and locked the intentional
  direct-SSP `ads.js` sink to one tested assignment.
- The final review found no remaining unowned raw/stored HTML path. Actual ad
  creative execution remains explicitly assigned to D02, publisher container
  isolation to P01, and future mobile web-view isolation to I02.

## Verification Results

- Go 1.23.5: full tests and vet passed in Aofei, pzdesign, and Genelet.
- Race: the documented Aofei scoped suite, pzdesign `./cmd/unify`, and full
  Genelet suite passed.
- Staticcheck v0.5.1: Aofei passed; pzdesign passed with its documented
  `ST1000`/`ST1003`/`ST1006` exclusions; Genelet passed after excluding its
  pre-existing legacy naming and simplification classes.
- Both pzdesign `.g` and `.e` template surfaces passed in one 265-template run;
  public-copy, pzdesign public-data, Aofei documentation, Aofei public-data, and
  all three repository diff-hygiene checks passed.
- No Docker, schema, cache, or deployment mutation was needed because S04
  changes only rendering/templates, local assets already tracked in pzdesign,
  tests, and documentation.

## Downstream Reconciliation

- Updated P01, I01, R01, S02, P02, and I02 as required by the goal impact map.
- Also updated D02, the owner of actual creative delivery validation and any
  future isolated behavioral preview, and I03, whose external management API
  must keep strings and generated documentation outside trusted HTML.
- No evolution entry is needed: S04 enforces the existing `html/template` and
  delivery/control-plane boundary without changing product direction or a
  public contract. No commit was created under the goal's default no-commit
  policy.
