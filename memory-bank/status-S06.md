# Status S06 - Public Account Abuse Protection

State: `[+]` Complete

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
| Cloudflare activation | `[+]` | A valid scoped management token created and read back the managed `w8m-public-account` widget for only `w8m.com` and `www.w8m.com`. The widget keys and reviewed Cloudflare proxy ranges are in the owner-only `0600` deployment environment. API dry runs proved the current Free plan cannot express the original POST-only 10-request/10-minute edge target or use Managed Challenge. The owner chose to remain on Free, so the active version-1 `http_ratelimit` entry point contains exactly one final rule: the four exact UI paths, verified bots excluded, IP plus mandatory data-center characteristics, 10 requests/10 seconds, and a 10-second Block. Independent readback confirmed the complete rule and widget. Never put the management token or widget secret in Git or command output. |
| Production deployment and live proof | `[+]` | The owner-only protection environment is attached to `aofei-unify.service`; the service is active and healthy. All eight Chinese/English advertiser/publisher registration/recovery pages render exactly one action-bound managed widget with the real site key and no secret. The exact bare `https://w8m.com` origin is allowed alongside canonical `ServerURL=https://www.w8m.com`; browser-style missing-token posts from both hosts reach the S06 `400` boundary. Missing and invalid production tokens returned `400` with unchanged account/quota state and only the fixed submission/rejection metrics advancing. An isolated localhost clone plus controlled Siteverify response exercised successful advertiser/publisher registration and recovery through real MySQL, Redis, Google OAuth, and Gmail API dependencies; Gmail accepted each send and exact inactive fixtures were removed afterward. A two-request email-hour test returned an atomic `429`, retained positive TTLs and unchanged aggregate quota values across denial, and exposed no raw email/IP. Trusted-loopback malformed forwarding failed closed while an untrusted direct peer's spoof was ignored. The Free edge rule produced ten `200`s then two `429`s while `/pz` remained outside the rule. Gmail-first rollback, protection removal, restart, and full real-key restoration were proved. |

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
- Completed production deployment evidence: the installed and staged `unify`
  binary hashes matched; the real-key service is active with `/healthz` and
  `/readyz` returning `204`; all four fixed-cardinality metric maps are
  registered; and all eight protected start pages render the expected widget,
  action, and public site key without the secret.
- Completed iteration-4 repository verification: hostile `_gadmin` publisher
  submissions, safe/internal error rendering, canonical MIME headers, per-key
  OAuth refresh concurrency and invalidation, Redis `EVALSHA`/`NOSCRIPT`
  fallback, and the full Aofei, pzdesign, and Genelet
  tests/vet/race/staticcheck/documentation/diff gates.
- Completed Cloudflare activation: final independent readback found the
  managed two-hostname widget and exactly one enabled version-1 zone rule with
  the four exact UI paths, verified-bot exclusion, IP/data-center
  characteristics, 10 requests/10 seconds, and 10-second Block.
- Completed live proof: missing/invalid production tokens had no account,
  Redis-quota, or Gmail admission side effect; controlled valid advertiser and
  publisher registration/recovery reached real deployed dependencies; the
  email quota returned an atomic `429`; fixed metrics and Redis privacy/TTL
  evidence matched the contract; direct-origin forwarding spoof checks passed;
  the edge burst blocked only account UI paths; and Gmail-first
  rollback/real-key restoration returned the service to healthy protected
  operation. Temporary account fixtures, validation credentials, simulator,
  configs, and browser artifacts were removed; bounded quota keys were left to
  expire normally.
- Completed final closeout: full Aofei, pzdesign, and Genelet tests/vet/race
  gates, pinned staticcheck, documentation/public-data guards, and all three
  diff checks passed. Focused post-review proof confirmed browser-style posts
  from both allowed production hostnames reach the missing-token `400` boundary
  with no account/quota mutation. The evolution log was checked; no successor
  to v28 is needed because activation fulfills its existing product and
  architecture direction rather than changing that direction.

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
5. Iteration 5 (2026-08-24): one P2 production compatibility finding. The
   Turnstile widget and hostname allowlist intentionally covered both
   `w8m.com` and `www.w8m.com`, but Summer's canonical `ServerURL` was
   `https://www.w8m.com` and production `CORSOrigins` was empty. A same-site
   browser POST from bare `https://w8m.com` was therefore rejected with `403`
   before the S06 boundary. Add only the exact bare origin to the active Summer
   allowlist, restart, and prove both hostnames reach the expected missing-token
   `400` before beginning iteration 6.
6. Iteration 6 (2026-08-24): clean. Production `CORSOrigins` now contains only
   exact bare `https://w8m.com` in addition to the canonical `ServerURL`
   allowance. The restarted real-key service is healthy; both hostnames passed
   browser-origin submission proof; all eight protected pages retained their
   exact widgets/actions; Cloudflare, Redis privacy/atomicity, Gmail,
   direct-origin, rollback, fixture-cleanup, repository verification, and
   documentation evidence were reviewed together. No P1, P2, or
   higher-severity finding remains.

## Exclusions

- Turnstile does not replace CSRF, password policy, email verification, login
  throttling, or account approval.
- Activation links and signed password-reset completion pages prove email/link
  possession and do not receive another Turnstile widget.
- No CAPTCHA answer, raw identity, or account-existence result is retained.
