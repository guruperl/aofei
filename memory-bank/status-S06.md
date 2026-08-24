# Status S06 - Public Account Abuse Protection

State: `[~]` In progress; repository remediation (iteration 4) is complete.
The managed Cloudflare widget now exists. The constrained Free-plan edge rule,
production enablement, and live proof remain pending.

## Goal

Protect public advertiser and publisher registration and password-recovery
mail workflows from automated submissions without adding a permanent checkbox,
leaking account existence, trusting spoofable forwarding headers, or storing
raw email/IP identities in Redis.

## Dependencies

- S04 contextual template escaping and its narrow reviewed remote-resource
  exception.
- The completed Gmail API account-delivery boundary and the shared Redis
  runtime used by `cmd/unify`.
- Cloudflare Turnstile plus zone rate-limiting capability for production edge
  activation; repository/local verification does not require live credentials.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Human-verification boundary | `[+]` | Genelet reserves `_gadmin` (strips any client value before role resolution) and sets it only for an authenticated configured admin role; the publisher filter derives `who` from the immutable resolved `RoleValue` instead of the client-controlled marker. Proved by `TestControllerScrubsCallerSuppliedAdminMarker` (Genelet `8c22274`) and `TestPublisherProtectionIgnoresClientSuppliedAdminMarker` (pzdesign `ea7be6e`). |
| Application quotas | `[+]` | The same `_gadmin` bypass is closed and the per-request raw `EVAL` assembly is replaced with a package-level `radix.NewEvalScript` (EVALSHA with NOSCRIPT fallback), preserving atomic denial, TTL, privacy, and metrics (pzdesign `5814f64`). |
| Trusted client address | `[+]` | Direct peers cannot supply forwarding identity. Only explicitly configured proxy CIDRs are removed from the right side of `X-Forwarded-For`; malformed trusted-proxy chains fail closed, and the derived address is used for Siteverify, quotas, and new-account IP storage. |
| Runtime and UI integration | `[+]` | Added `genelet.ClientSafeError`/`ClientSafeErr`: reviewed localized errors retain status/message while ordinary `Gerror` (<1000) and unexpected internal errors render generic status text; `303` redirects preserved. Public-account errors converted to `ClientSafeErr`. Proved by `TestControllerPreservesLocalizedHTTPError` and `TestControllerScrubsInternalHTTPErrorDetail` (Genelet `2698ac9`; pzdesign `1e5a9ae`). |
| Gmail MIME construction | `[+]` | `gmailRawMessage` canonicalizes header names case-insensitively, rejects canonical duplicates, and supplies `MIME-Version`/`Content-Type`/`Content-Transfer-Encoding` defaults only when absent, preserving caller-selected content types (Genelet `27b9d57`). |
| Gmail token concurrency | `[+]` | `accessToken` holds the cache lock only around state, coalesces refreshes per credential key (different keys refresh in parallel), and uses per-key generations so `401` invalidation cannot restore a stale in-flight token. Proved under `-race` by `TestGmailTokenCacheCoalescesConcurrentRefreshes` and `TestGmailTokenInvalidationPreventsStaleInFlightToken` (Genelet `6a0fa64`). |
| Metrics and operator contract | `[+]` | Fixed-cardinality expvar counters cover submissions/admissions, Turnstile rejection, quota rejection, and dependency failure. `docs/public-account-abuse-protection.md`, production/Chinese runbooks, README, architecture, and tech-stack memory define activation, rotation, rollback, alerts, and secret handling. |
| Cloudflare activation | `[~]` | A valid scoped management token created the managed `w8m-public-account` widget for only `w8m.com` and `www.w8m.com`; API readback found no pre-existing widget or rate-limit entry point. The widget keys and reviewed Cloudflare proxy ranges are in the owner-only `0600` deployment environment. A dry run proved the current Free plan cannot express the original POST-only 10-request/10-minute edge target: it permits Path/Verified Bot matching and a 10-second period only. The owner chose to remain on Free; create and read back the constrained exact-path 10-request/10-second Managed Challenge profile documented in `docs/cloudflare-w8m.md`. Never put the management token or widget secret in Git or command output. |
| Production deployment and live proof | `[~]` | The activation-ready `unify` binary is installed, the user service is healthy, and all four Chinese production start pages were checked. The owner-only protection environment now exists but has not been attached to the service, so protection remains intentionally default-off and pages still render without a widget. After the constrained edge rule is active, add the environment file to the user service, restart, then prove widget rendering, missing/invalid-token rejection with zero account/mail side effects, successful advertiser/publisher submissions, quota `429`, metrics, and rollback. |

## Acceptance Criteria

- Every enabled public registration or recovery submission needs a fresh
  Turnstile token whose hostname and role/action match the rendered form.
- Invalid or unavailable verification returns before Redis, database, Google
  OAuth, Gmail, or account mutation; Redis quota failure returns before Google,
  database, or mail.
- Direct clients cannot spoof the quota/Siteverify IP, and Redis/metrics/logs do
  not expose raw emails, IP addresses, Turnstile tokens, or secrets.
- A client-supplied `_gadmin` value cannot change public-account protection or
  authorization behavior; only Genelet's authenticated role resolution may set
  the trusted compatibility marker.
- Anonymous HTML and JSON responses contain only explicitly reviewed
  client-safe messages. Internal type, configuration, database, and provider
  error strings remain server-side.
- Gmail messages contain one canonical copy of each MIME header, and a slow or
  failed OAuth refresh for one credential does not serialize unrelated keys.
- The application remains default-off for local/open-source use but cannot
  start with partial enabled configuration. Production enablement is not
  claimed until the live Cloudflare and W8M evidence is retained.

## Verification

- Completed with Go 1.23.5: full Aofei, Genelet, and pzdesign tests and vet;
  pinned staticcheck v0.5.1 for Aofei and pzdesign with its documented legacy
  style exclusions; the documented Aofei and pzdesign scoped race suites and
  Genelet's full race suite.
- Completed: Turnstile context/failure tests, miniredis atomic
  quota/TTL/privacy tests, trusted-proxy tests, Chinese/English rendering
  tests, both template parsers, the exact Turnstile-only remote-script check,
  public-copy/public-data guards, documentation guard, and all three
  repository diff-hygiene checks.
- Completed production-safe deployment evidence: the installed and staged
  `unify` binary hashes matched, the restarted service was active, `/healthz`
  succeeded, all four fixed-cardinality metric maps were registered, and all
  four Chinese production start pages remained available without a widget
  while the feature is disabled.
- Completed iteration-4 repository verification: hostile `_gadmin` publisher
  submissions, safe/internal error rendering, canonical MIME headers, per-key
  OAuth refresh concurrency and invalidation, Redis `EVALSHA`/`NOSCRIPT`
  fallback, and the full Aofei, pzdesign, and Genelet
  tests/vet/race/staticcheck/documentation/diff gates.
- Completed Cloudflare discovery: token verification, exact active-zone
  resolution, empty-widget/rate-rule readback, managed-widget creation and
  owner-only secret handoff, and a non-persisting rule dry run that recorded
  the Free-plan 10-second/method-matching limitation.
- Pending closeout: live Cloudflare widget/rule inspection, enabled-form and
  no-side-effect tests, Gmail delivery, quota/metrics evidence, rollback proof,
  and the final milestone review after production activation is complete.

## Review Iterations

1. The repository review found that successful human verification needed an
   unforgeable request-local marker and that Cloudflare had to be documented as
   an external security processor. It also required bounded token/response
   sizes and a byte-narrow template-check exception. Those findings were fixed
   in code, tests, rendering policy, and privacy documentation.
2. The follow-up repository review found no unresolved P1/P2 implementation
   issue. Cloudflare activation and live production evidence remain open tasks,
   not waived findings.
3. The 2026-08-23 `review2.md` pass reopened repository work. It confirmed a P1
   client-controlled `_gadmin` publisher bypass and a P1 anonymous error-detail
   boundary, plus P2 Gmail MIME duplication and process-global OAuth refresh
   serialization. Raw quota `EVAL` was accepted as a P3 reuse/efficiency
   finding. These findings remain pending; after their fixes and focused
   verification, review the complete S06 repository diff again as iteration 4.
4. Iteration 4 (2026-08-23): clean. The complete S06 repository diff was
   reviewed again for correctness, failure semantics, security/privacy,
   compatibility, operations, tests, and documentation after fixing the
   `_gadmin` boundary, quota EvalScript reuse, client-safe error rendering,
   Gmail MIME construction, and Gmail token concurrency. No P1, P2, or
   higher-severity finding remains. Cloudflare activation and production
   live-proof are the only open tasks and remain blocked on a valid Cloudflare
   credential and external-mutation authority, not waived.

## Exclusions

- Turnstile does not replace CSRF, password policy, email verification, login
  throttling, or account approval.
- Activation links and signed password-reset completion pages prove email/link
  possession and do not receive another Turnstile widget.
- No CAPTCHA answer, raw identity, or account-existence result is retained.
