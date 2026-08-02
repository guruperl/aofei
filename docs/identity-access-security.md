# Identity, Two-Factor Authentication, And Access Control

This document is the deployment and maintenance contract for the optional
Summer/Genelet identity boundary implemented by S02. The checked-in example
keeps `Identity.Enabled` false so a schema load cannot silently change a live
login flow. Production enablement requires the schema, one deployment-owned
encryption key, reviewed role permissions, SMTP recovery delivery, and an
explicit rolling plan.

## Security Model

When identity is enabled, the existing signed role cookie is paired with an
opaque database-backed session. Genelet validates both on every authenticated
request. The database stores only a keyed SHA-256 digest of the 256-bit session
token. Sessions have independent absolute and idle deadlines, are rotated on
each login, and are revoked on logout, password change/recovery, TOTP disable,
or administrator TOTP reset. Logout is POST-only and CSRF-protected.

The authenticated role and account id come from verified cookies, never from a
request form. Existing component foreign-key checks preserve advertiser and
publisher ownership. Each component action has a permission name: an explicit
`permission`/`permission_<role>` value, or the stable
`<component>.<action>` default. An action may also define a resource tuple and
an `mfa` or recent-reauthentication requirement. JSON report access changes a
`report.*.read` requirement to the distinct `report.*.export` permission and
requires recent MFA.

The intended role boundary is:

| Role | Boundary |
|---|---|
| `admin` | Platform administration. The example grants `*`; sensitive bidder, seller, route, credential-reference, export, and experiment actions still require recent MFA and produce security audit rows. |
| `adv` | Own advertiser profile, demand objects, bidder proposals, and scoped reports. It cannot approve its bidder profile or edit another advertiser. |
| `pub` | Own publisher profile, inventory, supply proposals, and scoped reports. It cannot authorize its seller tuple or edit another publisher. |
| `agent` | Reviewed advertiser/campaign/creative read and approval surfaces only; it is not an advertiser account. |
| `analyst` | Read-only permissions plus an exact active database grant for every resource. It cannot mutate accounts, delivery, routes, bidder credentials, seller approval, payments, or experiments. |

Hosted funding/payout actions use the separate `payment.*` permission family.
Reading, customer/payout binding, binding approval, funding/payout/refund
proposal, operation approval/execution/cancellation, dispute handling,
reconciliation, and secret-readiness checks are independent grants. Every
mutation requires recent MFA and exact advertiser/publisher scope; the maker,
checker, and executor separations remain enforced inside the A02 service even
when an administrator has `*`. See
[hosted-funding-payout.md](hosted-funding-payout.md).

Do not infer authorization from a hidden link or an HTML template. Server-side
component rules and database grants are authoritative for HTML and JSON.

## Data And Cryptography

S02 adds `analyst`, `auth_mfa`, `auth_recovery_code`, `auth_session`,
`auth_permission_grant`, and `auth_security_audit` to the clean baseline.

- TOTP follows RFC 6238 with SHA-1, six digits, and a 30-second period. The
  accepted clock window is configurable from zero to two periods. A database
  step claim prevents replay, including across HTTP nodes.
- TOTP secrets are encrypted with AES-256-GCM. The 32-byte key comes only from
  the environment variable named by `Identity.KeyEnv` and may be base64 or
  hexadecimal. Startup fails closed when identity is enabled and the key is
  missing or malformed.
- Recovery codes are generated from cryptographic randomness, shown once, and
  stored only as domain-separated keyed SHA-256 digests. Each code is consumed
  once.
- Login identifiers in failed-login evidence are stored only as a keyed,
  normalized digest. Security audit rows do not store passwords, TOTP secrets,
  recovery codes, session tokens, request bodies, or external bidder
  credentials.
- `auth_security_audit` is immutable through database triggers. The ordinary
  HTTP application database principal must have `SELECT` and `INSERT`, but no
  `UPDATE` or `DELETE`, on this table. Only a separate maintenance principal
  may run the bounded retention operation, which temporarily enables deletion
  on its dedicated connection. The supported retention range is 365 through
  2555 days; the example uses 400 days. Expired/revoked sessions and used
  recovery codes are operational state and are removed after 30 days by the
  same maintenance command.

The identity key must be identical on every `unify` node and on the
`identity-admin` maintenance host. Store it in the deployment secret manager,
restrict it to the service/operator identities, back it up separately from the
database, and never put its value in JSON, shell history, logs, tickets, or Git.
There is no transparent dual-key rotation. A lost key makes enrolled TOTP
secrets unrecoverable. A compromise response must coordinate a key change,
session invalidation, and administrator-driven TOTP reset/re-enrollment for
every affected account.

## Configuration

The reviewed example is `etc/summer.example.json`:

```json
"Identity": {
  "Enabled": false,
  "KeyEnv": "GENELET_IDENTITY_KEY",
  "Issuer": "W8M",
  "SessionAbsoluteSeconds": 43200,
  "SessionIdleSeconds": 1800,
  "ReauthSeconds": 600,
  "RecoveryTTLSeconds": 900,
  "TOTPWindow": 1,
  "RecoveryCodeCount": 10,
  "RequiredTOTP": ["admin", "adv", "pub", "agent", "analyst"],
  "AuditRetentionDays": 400
}
```

Each role also declares `Permissions`; `analyst` declares
`RequireGrant: true`. Each database issuer uses `Password_hash: "passwd"`.
`Legacy_password_upgrade: true` permits an exact legacy plaintext match once
and replaces it atomically with bcrypt immediately after successful login.
This compatibility switch is for a measured migration window; confirm no
plaintext values remain, then disable it.

Public advertiser and publisher recovery additionally require a complete
Summer `Blks._gmail` block. Recovery links expire after
`RecoveryTTLSeconds`. An MFA-enabled account must also provide one unused
recovery code. Password update, code consumption, session revocation, and audit
insert commit in one transaction. A missing/failed email delivery cannot reach
that transaction, and a missing/invalid recovery code rolls it back.

All newly created and changed passwords must contain 12 through 128 Unicode
characters, no control characters, and at most 72 UTF-8 bytes (the bcrypt input
limit). Stored credentials are bcrypt hashes. Never create an active production
account by inserting a plaintext password.

## User Workflow

After the first password login, a role listed in `RequiredTOTP` receives a
limited session and is directed to **Account security**. The user copies the
one-time setup secret/standard `otpauth` URI into an authenticator, confirms a
current code, and saves the generated recovery codes offline. Until enrollment
is complete, no other protected component is available.

Subsequent logins accept either a current TOTP or one unused recovery code in
the same authentication-code field. TOTP steps cannot be replayed. Recovery
codes should be rotated after one is used or exposed. Disabling TOTP requires a
current TOTP plus a reason and revokes every session.

Password recovery starts from the public login page and requires delivery to
the account email. For an MFA-enabled account, the reset page also requires an
unused recovery code. If the user has neither TOTP access nor a recovery code,
an operator must verify identity outside application logs and use the audited
administrator reset procedure; support staff must never ask the user to send a
TOTP secret or recovery code.

## Operator Command

Build `../pzdesign/cmd/identity-admin` onto a restricted maintenance host. The
command has no HTTP endpoint. Give it a separate Summer configuration whose
`ConnectArray` selects the maintenance database principal; do not give the
HTTP service that credential. The command reads `SUMMER`/`-config`, the identity
key from the configured environment variable, and a new analyst password only
from `IDENTITY_NEW_PASSWORD`. Every mutation requires an administrator id and
a single-line reason of at most 255 bytes.

```bash
SUMMER=/etc/aofei/summer.json \
IDENTITY_NEW_PASSWORD='use-a-secret-input-channel' \
/opt/aofei/bin/identity-admin \
  -action=create-analyst -actor-admin-id=42 \
  -login=reader@example.invalid \
  -reason='approved operations read-only access'

SUMMER=/etc/aofei/summer.json /opt/aofei/bin/identity-admin \
  -action=grant -actor-admin-id=42 \
  -subject-role=analyst -subject-id=7 \
  -permission=report.marketplace.read \
  -resource-role=marketplace -resource-id=0 \
  -reason='quarterly marketplace review'

SUMMER=/etc/aofei/summer.json /opt/aofei/bin/identity-admin \
  -action=revoke -actor-admin-id=42 \
  -subject-role=analyst -subject-id=7 \
  -permission=report.marketplace.read \
  -resource-role=marketplace -resource-id=0 \
  -reason='review assignment ended'

SUMMER=/etc/aofei/summer.json /opt/aofei/bin/identity-admin \
  -action=reset-totp -actor-admin-id=42 \
  -subject-role=adv -subject-id=123 \
  -reason='identity verified under incident case reference'

SUMMER=/etc/aofei/summer.json /opt/aofei/bin/identity-admin \
  -action=prune-audit -actor-admin-id=42 -limit=1000 \
  -reason='scheduled account-security retention'
```

The `-actor-admin-id` is an audit attribution, not a substitute for operating
system authentication. Limit command execution and config/key access to named
administrators, record the change ticket externally, and use a separate
checker for privileged access. Never pass a password or key on a command line.

## Enablement And Rollback

For a populated deployment, do not replay `etc/step4_init.sql`. Apply a reviewed
online migration that creates the six S02 tables and two immutable-audit
triggers. Back up and restore-test first. Then:

1. Deploy code and templates with `Identity.Enabled=false`; verify ordinary
   bcrypt login, registration, password recovery, and both portals.
2. Provision the same 32-byte key to every HTTP node and the maintenance host.
   Verify SMTP delivery, clocks/NTP, the role permission matrix, secure cookies,
   and the public HTTPS origin.
3. Enable identity on a canary with a test account, enroll TOTP, save and use a
   recovery code, verify idle/absolute expiry, POST logout, password reset, and
   audit rows. Confirm a cross-account request and an analyst mutation both
   receive denial.
4. Roll all HTTP nodes with the same configuration. Require enrollment role by
   role only after support and recovery procedures are staffed. Monitor login
   failures, permission denials, recovery-code use, administrator resets, and
   audit insertion/database errors.
5. After every active credential is bcrypt, set every issuer's
   `Legacy_password_upgrade` to false and roll again.

Emergency rollback is `Identity.Enabled=false` on every node using a controlled
rolling change. It restores legacy signed-cookie behavior and therefore lowers
the security boundary; record and time-bound the exception. Do not drop S02
tables or triggers, delete audit evidence, revert bcrypt hashes to plaintext,
or remove TOTP data during a binary rollback. Restore identity only after the
database, common key, clocks, and mail path are healthy.

## Verification

The current clean baseline after A02 contains 94 tables, 6 routines, and 55
triggers. At S02 closeout it contained 75 tables, 6 routines, and 28 triggers;
I03 subsequently brought the inventory to 79 tables and 33 triggers, S03 adds
nine tables and ten triggers, and A02 adds six tables and twelve triggers. S03 evidence reads use
exact account/resource grants; rule rollout, review resolution, enforcement,
rollback, and billing recommendations require named permissions and recent
MFA. The S02 verification gate includes:

```bash
GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./...
(cd ../genelet && GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./...)
(cd ../pzdesign && GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./...)
(cd ../pzdesign && GOWORK=off GOTOOLCHAIN=go1.23.5 \
  go run ./tools/check-templates.go -ext=.g,.e)
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh diff-schema
git diff --check
```

Also restore the baseline into a uniquely named disposable MySQL 8 instance,
verify the object counts, prove direct update/delete of
`auth_security_audit` fails, and exercise analyst creation/grant/revocation
with a disposable key and account. Never run that destructive fixture against
a configured development or production database.
