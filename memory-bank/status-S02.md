# Status S02 - Identity, Two-Factor Authentication, And RBAC

State: `[+]` Complete

## Goal

Protect administrative and commercial accounts with two-factor authentication,
granular permissions, read-only analysis access, and auditable session behavior.

## Dependencies

- S01 data classification, retention, and security-event policy.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Permission model | `[+]` | Admin, advertiser, publisher, agent, and grant-scoped read-only analyst roles have an explicit component-action permission matrix and exact resource tuples. |
| Authorization enforcement | `[+]` | Genelet enforces named permissions, account/resource scope, report-export separation, and recent MFA before protected component hooks/actions; hostile caller audit metadata is discarded. |
| TOTP enrollment | `[+]` | RFC 6238 enrollment/confirmation uses AES-256-GCM secrets, replay-safe time-step claims, one-use digested recovery codes, audited reset, and documented operator policy. |
| Session hardening | `[+]` | Opaque digested database sessions enforce absolute/idle expiry, rotation/revocation, recent reauthentication, secure cookies, POST-plus-CSRF logout, and bounded cleanup. |
| Account recovery | `[+]` | Mail delivery is required before the atomic reset transaction; MFA recovery consumes one code, hashes the password, revokes sessions, and audits the change or rolls back completely. |
| Security audit | `[+]` | Fixed-field redacted security events cover identity, privilege, credential-reference, seller, payment, experiment, and route changes. Trigger immutability, separated application/maintenance database grants, and a bounded 365–2555-day retention command protect the evidence. |

## Acceptance Criteria

- Every protected action has a named permission and cross-account denial tests.
- TOTP secrets and recovery codes are never logged or stored in plaintext.
- The analyst role can inspect authorized reports but cannot mutate delivery,
  credentials, routes, payments, or accounts.
- Recovery cannot bypass 2FA or mutate account state when its delivery channel
  is unavailable.

## Verification

- Authn/authz matrix, session, CSRF, TOTP/recovery, account isolation, audit,
  secret/public-data, template, and full closeout suites.

## Result

- Genelet now owns the optional database-backed identity boundary; the checked-
  in example leaves it disabled until the schema, shared environment key, SMTP,
  role matrix, and rollout procedure are ready.
- At S02 closeout, the clean baseline contained 75 tables, 6 routines, and 28
  triggers; I03 subsequently extended the current baseline. Six S02
  tables hold analysts, encrypted TOTP, digested recovery/session material,
  exact grants, and immutable redacted security evidence.
- `../pzdesign/cmd/identity-admin` provides restricted analyst creation,
  grant/revoke, TOTP reset, and retention operations. Summer exposes account
  security and read-only analyst surfaces without making internal UI JSON a
  public API.
- The complete deployment, user, recovery, least-privilege, rollback, and
  maintenance contract is in
  [identity-access-security.md](../docs/identity-access-security.md).

## Closeout Review

- Cross-account resource tests deny before application mutation, and registry
  coverage proves every protected Summer action has a role permission.
- Hostile template fixtures, local-return validation, per-form CSRF injection,
  logout method enforcement, audit-metadata scrubbing, TOTP replay, recovery
  atomicity, and connection cleanup were reviewed and tested.
- A disposable MySQL 8.0.41 restore proved the schema inventory, active-grant
  uniqueness, immutable audit triggers, analyst lifecycle, bounded pruning, and
  cleanup of the connection-local retention flag. The container was removed.
- Go 1.23.5 tests and vet passed in Aofei, Genelet, and Pzdesign; scoped race,
  pinned staticcheck, template/public-copy, documentation/public-data/SQL, and
  all three `git diff --check` gates passed. Genelet staticcheck used documented
  exclusions for pre-existing naming/style diagnostics only.
- Deep review findings were resolved in S02. Evolution v23 records the new
  account/security schema and configuration boundary.

## Reconciliation From S01

- S01's data inventory, least-privilege, encryption/key ownership, backup
  deletion, redaction, and fixed-cardinality evidence rules are mandatory for
  authentication data, TOTP secrets, recovery material, sessions, and security
  events. S02 must document any longer statutory/security retention separately.

## Reconciliation From S04

- Enrollment, login, recovery, denial, and audit pages/mail retain contextual
  escaping and Genelet's one fixed CSRF trusted-HTML boundary. QR/TOTP and
  recovery data must not be inserted through `template.HTML`, data-HTML URLs,
  remote scripts, or executable error messages.
- Extend hostile rendering fixtures for every new identity/security template
  and keep local-return redirect validation in the authentication controller.

## Reconciliation From A01

- Portal/API financial permissions must preserve A01's maker/checker rule:
  statement creator, confirmer, and settlement recorder are distinct trusted
  principals. A caller-supplied display name cannot replace the authenticated
  actor, and direct Summer funding modules remain retired.
- Financial audit views are read-only over immutable A01 facts, expose no full
  card/bank credential, and require reasoned, reauthenticated actions. The
  current effective-Unix-UID command boundary remains the safe fallback until
  equivalent portal RBAC and 2FA are proven.

## Reconciliation From D03

- Define separate permissions for bidder-profile approval, portable credential
  reference assignment, route/group/target edits, route publication, fallback
  activation, and the higher-risk `Always` gate. Sensitive approval,
  credential, publication, and activation actions require reauthentication and
  auditable actor identity; analysts remain read-only.
- Credential values stay in deployment-owned configuration and are never UI or
  database fields. RBAC may expose a bounded environment-variable name and
  readiness state, but no permission grants access to the JSON header value.

## Reconciliation From R02

- Name distinct permissions for advertiser reports, publisher reports,
  operator commercial/route reports, authenticated JSON export, experiment
  inspection, and experiment create/start/stop/complete. Preserve the current
  server-side `_grole` plus session-account scope and add cross-account denial
  tests; no request `admin_id`, `adv_id`, or `pub_id` can elevate scope.
- The analyst role may read only explicitly delegated report scopes and bounded
  experiment aggregates. It cannot see assignment salts, subject hashes,
  idempotency digests, audit reasons, private route/credential data, or mutate
  experiments. Experiment transitions and bulk/report exports require named
  audit permissions and sensitive-action reauthentication.

## Reconciliation From P02

- Define separate least-privilege permissions for publisher supply-metadata
  proposals, operator seller review/authorization, and read-only supply/report
  inspection. A publisher, advertiser, analyst, or agent cannot grant
  `seller_authorized`, alter another account's taxonomy, or infer access from a
  seller type.
- Security audit events record the actor, publisher, exact seller tuple hash,
  prior/new authorization state, reason, and time without copying credentials,
  private contracts, or raw request source data. Publisher edits that revoke
  authorization and operator reapproval are distinct events.
