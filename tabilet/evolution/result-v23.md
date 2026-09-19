# Result V23

S02 establishes the optional W8M identity, two-factor, and authorization
boundary.

Implemented direction:

- Genelet pairs its verified signed role cookie with a random opaque
  database-backed session when `Identity.Enabled` is explicitly activated.
  Sessions have absolute/idle expiry, rotate on login, revoke on security
  changes/logout, and protect logout with POST+CSRF.
- RFC 6238 TOTP secrets are AES-256-GCM encrypted with a deployment-owned
  32-byte environment key. Recovery codes, session tokens, and failed-login
  identifiers persist only as domain-separated keyed digests; codes and TOTP
  steps are one-time.
- Every component action resolves a stable permission name and optional exact
  resource. Sensitive seller, bidder, route, credential-reference, report
  export, password, and account-security operations require recent
  reauthentication as configured. Existing foreign-key signatures retain
  advertiser/publisher ownership enforcement.
- The new analyst role is read-only and requires one active database grant for
  each permission/resource. It receives no report-export or mutation
  permission by default.
- Security events use bounded structured fields and immutable audit triggers;
  secrets and request bodies are excluded. The supported audit retention class
  is 365–2555 days, separate from short traffic logs.

Contract consequences:

- The clean baseline gains six identity/analyst tables and two triggers.
  Populated systems require a reviewed migration before identity activation;
  the baseline is not a production migration.
- The checked-in example remains `Identity.Enabled=false`. Activation requires
  one common environment key on all HTTP/maintenance nodes, reviewed role
  permissions, working SMTP recovery, clock health, a canary, and a controlled
  roll/rollback procedure.
- `../pzdesign/cmd/identity-admin` is the restricted non-HTTP boundary for
  analyst creation, exact grant/revoke, TOTP reset, and retention. Its actor id
  is audit attribution and does not replace operating-system authentication or
  maker/checker review.
- I03, S03, A02, and any named I02 integration inherit the authenticated actor,
  explicit permission/resource, MFA, redacted-audit, and no-internal-JSON-as-
  public-API rules.
