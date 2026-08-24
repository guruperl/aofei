# Status S05 - Runtime Trust-Boundary Hardening

State: `[ ]` Planned

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
| Outbound address policy | `[ ]` | Replace implicit `IsGlobalUnicast` assumptions with one reviewed IPv4/IPv6 special-use deny policy covering private, loopback, link-local, unspecified, multicast, CGNAT, benchmarking, documentation, reserved, and future non-public ranges. Apply it before and during every dial. |
| Injected-client safety | `[ ]` | Preserve test/custom-client injection without bypassing DNS re-resolution, redirect-hop validation, proxy policy, TLS constraints, body/time limits, or credential stripping. Middleman bidding and callback forwarding must use the same safe transport contract. |
| Creative rendering boundary | `[ ]` | Inventory every first-party HTML/JSON/OpenRTB/SDK consumer. Prove the browser renderer uses an opaque-origin sandbox without `allow-same-origin`; specify equivalent WebView/navigation/storage/bridge isolation for native consumers. Strengthen URL/tag/event/CSP checks where compatible, but do not deploy a script-stripping sanitizer that breaks the approved ad contract without measured evidence and a migration. |
| Principal provenance | `[ ]` | Prove HTTP actors, permissions, scopes, and recent-MFA state originate only from verified Genelet sessions. Bind maintenance identities to an authenticated OS/service principal or an explicitly restricted, audited launcher instead of treating a supplied actor label as authentication. Preserve maker/checker separation at service and entry-point boundaries. |
| Dormant surface review | `[ ]` | Prove management URLs are never fetched before classifying private-host validation as SSRF, keep Redis marker strings non-authoritative, and add production preflight evidence that public example tracking secrets cannot satisfy a live deployment. Record dismissed findings with their enforcing tests. |
| Traffic-quality version selection | `[ ]` | Evaluate the highest eligible version for each `(rule_key, rollout_mode)` so a newer Observe/Canary rule cannot hide an older Active rule. Define deterministic precedence when multiple modes apply and keep incomplete evidence observe-only. |
| Database integrity | `[ ]` | Add column-level identity/evidence immutability for quality enforcement and billing while allowing documented state transitions, rollback, approval, retention, and correction workflows. Prove direct SQL cannot rewrite financial or quality authority without introducing blanket triggers that block valid lifecycle changes. |

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

## Reconciliation From P03

- P03 now supplies two distinct reviewed inputs: public versioned inventory
  locators and an App-scoped Ed25519 request principal. S05 principal review
  must preserve that separation, prove Summer credential actors still come
  only from Genelet's verified `_grole`/account/permission/MFA state, and never
  reinterpret request-proof headers, locators, Redis markers, or client account
  fields as control-plane authority.
- The current App integration output is a textual request sample, not a native
  renderer. No Android/iOS package or first-party WebView consumer was added by
  P03, so S05 still owns the complete creative-consumer inventory and hostile-
  markup isolation proof; conditional I02 remains untriggered.
- P03's immutable credential snapshot now serializes full reload construction
  with local issue/rotation/revocation mutation. S05 changes to principal or
  credential provenance must retain that immediate local containment and the
  bounded cross-node fail-closed refresh contract.
