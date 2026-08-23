# Status S06 - Public Account Abuse Protection

State: `[~]` In progress; repository implementation complete, production
activation pending a valid Cloudflare management credential.

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
| Human-verification boundary | `[+]` | Managed/interaction-only Turnstile widgets cover advertiser/publisher registration and recovery in Chinese and English. Server-side Siteverify runs before password hashing, Google credential checks, Redis, database mutation, or mail; exact hostname and fixed action must match, and tokens are removed from request form state. |
| Application quotas | `[+]` | One Redis script atomically admits fixed-window IP, normalized-email, and global limits. Email/IP keys are deployment-keyed HMAC digests, all keys expire, denial cannot partially increment another window, and Redis failure is a localized fail-closed `503`. |
| Trusted client address | `[+]` | Direct peers cannot supply forwarding identity. Only explicitly configured proxy CIDRs are removed from the right side of `X-Forwarded-For`; malformed trusted-proxy chains fail closed, and the derived address is used for Siteverify, quotas, and new-account IP storage. |
| Runtime and UI integration | `[+]` | `cmd/unify` validates the default-off environment configuration at startup, Genelet passes request-scoped runtime dependencies to filters, custom HTTP errors preserve localized text/status, and all eight public form variants render only the public site key and fixed action. |
| Metrics and operator contract | `[+]` | Fixed-cardinality expvar counters cover submissions/admissions, Turnstile rejection, quota rejection, and dependency failure. `docs/public-account-abuse-protection.md`, production/Chinese runbooks, README, architecture, and tech-stack memory define activation, rotation, rollback, alerts, and secret handling. |
| Cloudflare activation | `[!]` | The owner environment contains a Cloudflare token variable, but Cloudflare reports that token invalid. Create/reuse the exact W8M widget and zone rate-limit rule only after a replacement token has Turnstile Sites Write, Zone Read, and Zone WAF/Rulesets Write scope. Never put either token or the returned widget secret in Git or command output. |
| Production deployment and live proof | `[~]` | The activation-ready `unify` binary is installed, the user service restarted healthy, and all four Chinese production start pages were checked. Protection remains intentionally default-off, so they render without a widget until Cloudflare activation. After activation, write the returned keys and reviewed host/proxy settings to an owner-only `0600` environment file, add it to the user service, restart, then prove widget rendering, missing/invalid-token rejection with zero account/mail side effects, successful advertiser/publisher submissions, quota `429`, metrics, and rollback. |

## Acceptance Criteria

- Every enabled public registration or recovery submission needs a fresh
  Turnstile token whose hostname and role/action match the rendered form.
- Invalid or unavailable verification returns before Redis, database, Google
  OAuth, Gmail, or account mutation; Redis quota failure returns before Google,
  database, or mail.
- Direct clients cannot spoof the quota/Siteverify IP, and Redis/metrics/logs do
  not expose raw emails, IP addresses, Turnstile tokens, or secrets.
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

## Exclusions

- Turnstile does not replace CSRF, password policy, email verification, login
  throttling, or account approval.
- Activation links and signed password-reset completion pages prove email/link
  possession and do not receive another Turnstile widget.
- No CAPTCHA answer, raw identity, or account-existence result is retained.
