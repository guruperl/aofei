# Status S05 - Runtime Trust-Boundary Hardening

State: `[~]` In progress

## Goal

Harden outbound network, creative rendering, authenticated-principal, and
traffic-quality boundaries while preserving the intentional opaque-origin ad
container and legitimate third-party advertising behavior.

## Dependencies

- S01 data disclosure, S02 identity/RBAC, S03 traffic quality, and S04 template
  rendering contracts.
- D02 owns creative acceptance; D03 owns middleman transport and callbacks;
  completed P03 owns direct-SSP request authenticity.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Outbound address policy | `[+]` | `internal/safehttp` now normalizes IPv4-mapped addresses and applies one `net/netip` prefix policy to URL validation and every dial. It rejects private, loopback, link-local, unspecified, multicast, CGNAT, benchmarking, documentation, protocol-transition, reserved, and future non-public ranges; IPv6 is limited to public `2000::/3` after the complete `2001::/23` IETF reservation and other reviewed exclusions, with only the IANA globally reachable exceptions restored. Mixed DNS answers fail closed. Boundary fixtures cover every denied class and explicit exception, and the operator contract links the authoritative IANA IPv4/IPv6 registries. |
| Injected-client safety | `[+]` | Controller construction, bidder calls, live callback forwarding, and retry forwarding now normalize every injected client through `safehttp`. Supported `*http.Transport` settings are cloned while proxy/custom dial paths, insecure TLS, certificate-name overrides, oversized response headers, cookie jars, HTTPS downgrade, unsafe redirect destinations, and cross-authority credentials are rejected or removed. Redirect checks preserve an immutable pre-hook history and enforce validation/credential stripping before and after an injected hook. Arbitrary network round trippers fail closed; explicitly marked socket-free test doubles stay injectable but cannot bypass URL validation. Complete DNS answers are checked before any dial, including the rebind pass. Client context/timeouts and existing bidder/callback body bounds remain caller-owned and tested. |
| Creative rendering boundary | `[+]` | `docs/creative-rendering-boundary.md` inventories management source views, browser HTML, structured `/pz`, OpenRTB, audit, and P03 sample consumers and confirms that only pzdesign `ads.js` executes HTML; no maintained native renderer exists. Source/Node fixtures lock one opaque-origin `srcdoc` iframe without `allow-same-origin`, add a fixed sensitive-feature deny policy, and contain hostile strings without a host-page HTML sink. Repository guards reject another raw DOM sink and scan current web plus Android/iOS/cross-platform source languages for native WebView APIs. D02 contained-markup validation now covers compatible `srcset` URL tokens, `ping`, legacy `background`, and entity-decoded event attempts to address top/parent while retaining approved scripts. The document defines mandatory ephemeral WebView, navigation, storage, bridge, VAST, Native, lifecycle, and hostile-test rules for demand-gated I02. A universal CSP/script sanitizer is correctly deferred pending compatibility evidence and migration. |
| Principal provenance | `[+]` | Genelet now scrubs every caller-supplied `_g*` value, requires the Gate-validated opaque session, authorizes the server-configured component/action/permission/resource, and only then binds an exact typed principal whose provenance and request-context keys are package-private. Summer security, advertiser/publisher credential, traffic-quality, and hosted-payment actors require that capability, verify it still matches dispatch/resource scope, and derive recent MFA only from its server deadline. Direct `Controller.Handle`, header, form, and Summer-filter spoof tests fail closed. Traffic-quality and hosted-payment CLIs removed actor flags and derive exact-permission `unix-uid:<effective-uid>` principals that service boundaries restrict to health/retention and reject for quality mutation, reconciliation, or money movement. `identity-admin` now requires a restricted `Identity.MaintenanceActors` UID-to-admin mapping and prefixes the launcher UID in every audited reason. Existing maker/checker service rules remain intact, with explicit offline-principal denial tests and the full contract in `docs/principal-provenance.md`. |
| Dormant surface review | `[+]` | Closed the dormant findings with executable boundary evidence. Summer loopback tripwires prove site-review and creative source/image URLs are parsed/stored without a request, and an AST guard rejects outbound HTTP clients/transports/convenience fetches in the source-only campaign/item/site/creative packages; private-host syntax on these unfetched fields is not SSRF. Publisher/App replay, cap, and tracking-claim tests replace static marker contents and prove existence can suppress a duplicate but cannot authenticate, own a tracking claim, publish, or mutate caps; random owner tokens remain mandatory where ownership matters. New dependency-free `cmd/config-preflight` applies production bid validation, rejects both checked-in tracking-secret examples, surrounding whitespace, and values below 32 bytes without printing material; the checked-in config fails as intended while a deployment-owned fixture passes. |
| Traffic-quality version selection | `[+]` | Runtime assessment now selects the highest immutable version independently for each `(rule_key, rollout_mode)` and returns coexisting modes in deterministic Active, Canary, then Observe order. Older rows in the same mode cannot hide the selected version, and a newer Observe/Canary rollout cannot displace established Active behavior. Complete evidence preserves per-mode actions; partial or missing evidence forces every mode to Observe for action and billing. |
| Database integrity | `[+]` | Added narrow protected-update triggers for `quality_enforcement` and `quality_billing`. Enforcement rollback preserves its rule, decision, scope, action, canary allocation, creator, creation, and expiry; billing review preserves its decision, statement, digest, accounting version, disposition, recommender, reason, and creation evidence while enforcing an independent checker. Legitimate Canary/Active rollback and Recommended-to-Approved/Rejected/Applied service transitions still pass; direct protected-column, same-actor, and terminal rewrites fail. The baseline rebuilds at 95 tables, 0 views, 6 routines, and 57 triggers with SQL guard/drift checks clean. Disposable MySQL lifecycle tests, full three-repository tests/vet/staticcheck, the documented race suites, documentation/template/public-data guards, and diff hygiene pass on Go 1.23.5. |

## Deep Review

- Iteration 1 (2026-08-24): P2, resolved — the outbound policy denied selected
  IPv6 protocol ranges but not their IANA-reserved `2001::/23` parent, and the
  redirect policy permits HTTPS-to-HTTP downgrade. Reserved descendants can
  therefore pass the claimed registry boundary, while a 307/308 downgrade can
  expose a bidder request body even after cross-authority headers are removed.
  The complete parent range now fails closed except for the registry's explicit
  globally reachable children, and every redirect rejects HTTPS downgrade
  before a second transport call; focused tests, race, vet, and staticcheck
  pass.
- Iteration 1 (2026-08-24): P2, resolved — Genelet previously exported
  string-valued form markers as principal provenance and `Controller.Handle`
  accepted caller-set forwarded session/MFA headers. Genelet now carries the
  validated session and exact authorized principal under private request-
  context keys, with an unexported provenance bit; direct callers cannot bind a
  verified value. Summer consumes the typed principal and checks the exact
  component/action/permission/resource rather than authenticating marker
  strings. Genelet `fad25a9` and pzdesign `f6bc0c9` pass their full tests, vet,
  documented staticcheck, race, template, and public-data gates.
- Iteration 1 (2026-08-24): P3, resolved — the creative boundary previously
  checked the known `ads.js` sink without scanning the remaining first-party
  JavaScript or command/Summer/template trees. Pzdesign `8f8a341` adds a
  repository-wide guard for raw DOM insertion APIs and Android/iOS WebView
  renderer APIs, allowing only the one exact reviewed `ads.js` assignment; the
  focused test, vet, staticcheck, template, and public-data gates pass.
- Iteration 2 (2026-08-24): P2, resolved — an injected `CheckRedirect` hook ran
  before the mandatory downgrade and authority checks and received mutable
  `via` requests, allowing it to disguise a downgrade or cross-authority hop.
  The mandatory policy now snapshots the original redirect URLs and validates
  plus strips against that immutable history both before and after the hook.
  Adversarial history-rewrite/body/credential tests and focused test, vet,
  staticcheck, and race gates pass.
- Iteration 2 (2026-08-24): P2, resolved — Genelet removed reserved `_g*`
  values from `Request.Form`, while URL-encoded and pre-parsed multipart
  requests could retain the same public values in alternate standard views.
  Genelet `bbeef88` now scrubs `Form`, `PostForm`, and `MultipartForm.Value` at
  both public and direct-Handle boundaries while preserving ordinary fields;
  full tests, vet, documented staticcheck, race, and diff hygiene pass.
- Iteration 2 (2026-08-24): P2, resolved — contained-markup `srcset`
  validation split every comma, so it could reinterpret and reject a valid
  HTTPS image URL containing an internal comma. Candidate parsing now follows
  URL-token boundaries, preserves internal/trims only trailing delimiter
  commas, and continues validating every later URL. Compatibility and unsafe-
  later-candidate fixtures plus focused test, vet, staticcheck, and race gates
  pass.
- Iteration 2 (2026-08-24): P3, resolved — the initial native-renderer
  inventory scanned only Go, JavaScript, and template extensions under current
  web trees. Pzdesign `d8d09c5` adds a repository-wide platform-source scan for
  Java, Kotlin, Swift, Objective-C/C++, C#, Dart, TypeScript/JSX, and XML while
  excluding dependency/build trees, and expands the reviewed WebView/bridge
  marker set. Focused test, vet, staticcheck, template, and public-data gates
  pass.
- Iteration 3 (2026-08-24): clean — re-reviewed the complete S05 implementation
  across Aofei, Genelet, and pzdesign, including every iteration-1/2 fix, for
  network and redirect bypass, principal forgery, resource/MFA scope, creative
  compatibility and consumer discovery, quality-rule precedence, database
  transitions, maintenance authority, tests, and operator documentation. No
  P1, P2, higher-severity, or carry-forward finding remains.

## Acceptance Criteria

- Callback and bidder clients cannot reach denied special-use addresses through
  direct IPs, DNS rebinding, redirects, injected transports, or proxies.
- Approved browser markup remains functional only inside the documented
  opaque-origin boundary; no first-party consumer executes it in its own origin.
- Sensitive actions receive server-derived principals and cannot defeat
  maker/checker or recent-MFA policy by changing request or CLI actor strings.
- An Observe/Canary rollout cannot silently remove an existing Active quality
  rule, and protected database columns cannot be rewritten outside their
  documented lifecycle.

## Verification

- Exhaustive IPv4/IPv6 special-range, resolver/rebind, redirect, proxy, custom
  transport, and timeout tests.
- Hostile creative fixtures across browser, management/review, JSON/OpenRTB,
  and any named native consumer; pzdesign/Genelet authentication tests.
- Disposable MySQL trigger/state-machine tests, S03 rule-version/canary tests,
  full cross-repository tests/vet/staticcheck/race, security guards, and diff
  hygiene.

## Exclusions

- Sanitizing arbitrary third-party markup into a different creative language is
  not assumed to be compatible and requires a separate approved contract.
- Automatic or learned fraud scoring remains deferred.

## Dormant-Surface Finding Dispositions

- **Private-host management/review URLs — dismissed as SSRF on the current
  surface.** `summer/site.TestPresetTreatsPrivateHostReviewURLAsUnfetchedMetadata`
  and
  `summer/creative.TestManagementCreativeURLsAreValidatedWithoutFetching`
  send loopback URLs through the real filters and observe zero requests;
  `tools.TestSourceOnlyManagementPackagesHaveNoOutboundHTTPPrimitive` rejects
  a future production management HTTP client/transport/fetch primitive. A
  future preview, crawler, or verifier reopens the outbound-address review.
- **Static Redis `"1"`/`"done"` values — dismissed as hard-coded
  credentials.**
  `publisherauth.TestRequestProofBindsBodyFreshnessScopeAndSharedReplay`,
  `match.TestMustRefreshBothCapEventMarkerValueIsNonAuthoritative`, and
  `dsp.TestTrackingMarkerLabelsNeverGrantClaimOwnership` replace marker labels
  and prove they never create authentication, publication, cap mutation, or
  claim ownership. Redis write access can still cause denial and remains an
  internal access-control boundary.
- **Checked-in tracking secret in a live configuration — confirmed and
  remediated.** `dsp.TestProductionValidationRejectsPublicOrWeakTrackingSecrets`
  and `cmd/config-preflight.TestRunRejectsCheckedInExampleTrackingSecret`
  enforce the production-only rejection without changing deterministic local
  validation. Running the preflight against `etc/aofei.json` exits nonzero with
  a fixed non-secret diagnostic.

## Reconciliation From P03

- P03 now supplies two distinct reviewed inputs: public versioned inventory
  locators and an App-scoped Ed25519 request principal. S05 principal review
  preserves that separation: Summer credential actors come only from Genelet's
  typed verified component/action/permission/resource capability and session
  MFA deadline, never from request-proof headers, locators, Redis markers,
  compatibility `_g*` fields, or client account values.
- The current App integration output is a textual request sample, not a native
  renderer. No Android/iOS package or first-party WebView consumer was added by
  P03, so S05 still owns the complete creative-consumer inventory and hostile-
  markup isolation proof; conditional I02 remains untriggered.
- P03's immutable credential snapshot now serializes full reload construction
  with local issue/rotation/revocation mutation. S05 changes to principal or
  credential provenance must retain that immediate local containment and the
  bounded cross-node fail-closed refresh contract.
