# Status S08 — W8M Production Identity And TOTP Activation

State: `[~]` In progress; S08.1 contract accepted, S08.2 rehearsal next.

## Goal

Enable the already implemented S02 Identity boundary on W8M through a reviewed,
staged production rollout. The dedicated advertiser test account may enroll
TOTP voluntarily; the W8M production configuration must exclude `adv` from
`RequiredTOTP`. This is a W8M-specific rollout choice and does not change the
general S02 role-policy example.

## Dependencies And Ownership

- S07 must close before S08 begins, avoiding concurrent account-identifier and
  authentication-state migrations.
- Aofei owns the application/schema implementation and rollout contract.
- The W8M application operator owns private runtime configuration, the Identity
  key and database-operation authorization.
- w8m-infrastructure owns only its documented immutable release and service
  deployment workflow. It does not own schema migration, feature activation or
  Cloudflare/DNS changes.
- Task order: S08.1 -> S08.2 -> S08.3 -> S08.4 -> S08.5 -> S08.6.
- This status grants no database, secret, configuration, deployment, browser or
  account-mutation authority. Each production action requires its own exact
  authorization and current readiness.

## Tasks

| Item | State | Notes |
|---|---:|---|
| S08.1 Reconcile the exact production rollout contract | `[+]` | Confirm all W8M nodes, active schema, application/maintenance database grants, role permission matrix, analyst `RequireGrant`, SMTP recovery, HTTPS, clock, monitoring and rollback owner. Read-only inspection finds all six Identity tables and both audit triggers already present with zero Identity rows, but the HTTP principal has database-wide ALL PRIVILEGES and the runtime has no permissions or analyst grant requirement. Review restricted application/maintenance principals before activation; owner confirms yixin is the only node and no usable non-advertiser canary account is available. Owner selects a dedicated analyst canary via the audited maintenance CLI. Task contract review 1/10 passes; no production mutation or canary acceptance is implied. Preserve the current identity-disabled login behavior until the canary gate. |
| S08.2 Prepare and rehearse the online Identity migration | `[ ]` | Reconcile the existing six S02 tables and two immutable-audit triggers against their authoritative definitions, including the S07-retired analyst schema. Derive only necessary non-destructive corrections and restricted application/maintenance grant changes; do not recreate matching objects or replay `etc/step4_init.sql` against the populated deployment. Back up and restore-test first, rehearse on a disposable baseline copy, and verify object inventory, grants, trigger immutability and existing application compatibility. No production database action is implied. |
| S08.3 Apply schema and verify readiness | `[ ]` | After backup/restore rehearsal and separate authorization, apply the exact reviewed online migration before enabling Identity. Verify schema, grants, immutable triggers and preserved application data. Stop on drift, missing prerequisites or uncertain outcome; do not auto-retry a mutation. |
| S08.4 Deploy identity-disabled release and provision common key | `[ ]` | Deploy code/templates with `Identity.Enabled=false` and verify ordinary bcrypt login, registration, recovery and both portals before activation. Then provision one 32-byte Identity encryption key to every `unify` node and the restricted maintenance host through the approved secret channel. Keep its value out of JSON, repositories, command arguments, logs and evidence. Confirm key-version parity without exposing values, SMTP recovery, clock/NTP, secure cookies and role permissions. |
| S08.5 Enable Identity on a canary | `[ ]` | Under separate authorization, enable Identity on one canary with `adv` excluded from `RequiredTOTP`. Use an owner-approved operator-controlled non-advertiser canary account to verify required TOTP enrollment, login, one-use recovery, session expiry, POST/CSRF logout and audit insertion; also verify cross-account denial and analyst mutation denial. Do not enroll the dedicated W8M advertiser account here; that remains W27.4 after W27.3. Confirm healthy service and readiness; preserve an immediate reviewed rollback to `Identity.Enabled=false`. |
| S08.6 Roll out, monitor and close readiness | `[ ]` | After canary acceptance and owner authorization, roll the exact configuration to every production node. Verify consistent key/config versions, login and recovery paths, permissions, audit insertion, clock and readiness; monitor failures and denials. Record only sanitized evidence and rollback readiness, then provide the W8M W27.3 owner with the evidence reference. No seed, code, recovery code, key value, account identifier or raw production configuration enters this repository. |

## Acceptance Criteria

- A reviewed online migration is exercised against a restored disposable copy
  and applied to the authorized production database without replaying the clean
  baseline or losing existing data.
- All service and maintenance nodes use the same protected Identity key
  version; its value never appears in repository or operational evidence.
- Required role permissions, analyst grants, recovery mail, HTTPS, clocks,
  secure cookies and rollback are reviewed and verified.
- Identity is enabled and healthy on every production node; `adv` is absent
  from `RequiredTOTP`; the separately authorized non-advertiser canary passes
  required TOTP enrollment and recovery/security checks. The dedicated W8M
  advertiser account remains un-enrolled until W27.4 after W27.3.
- W8M receives a sanitized evidence reference sufficient to complete W27.3's
  separately authorized read-only check. W8M W27.4 enrollment and later browser
  capture/live count remain separately owned and authorized.
- Any failed or uncertain production mutation stops the rollout. Rollback
  disables Identity only; it does not drop S02 tables/triggers or delete TOTP
  data/audit evidence.

## Required Verification

- Run the S02 focused identity, permission, recovery, session, audit and
  template checks across Aofei, Genelet and Pzdesign.
- Rehearse the exact online migration on a uniquely named disposable MySQL 8
  instance restored from a backup copy; verify object inventory, privileges,
  immutable audit triggers and cleanup before removing the fixture.
- Complete the identity-disabled baseline checks, then the authorized canary
  and all-node rollout checks from
  [identity-access-security.md](../../docs/identity-access-security.md).
- Run affected repository checks, syntax/JSON validation and `git diff --check`;
  perform the bounded milestone review before completion.

## Stop Conditions

- Stop before production mutation if S07 is incomplete, the actual schema or
  grants differ from rehearsal, backup restore is unproven, key delivery is
  inconsistent, recovery email or clock is unhealthy, role permissions are
  unreviewed, or rollback is unavailable.
- A task row never grants authority for the next external action. Obtain exact
  authorization and recheck readiness before migration, secret/config change,
  canary activation, or rollout.
- Do not modify Cloudflare/DNS, certificates, the private W27 browser profile
  or the dedicated W27 advertiser seed/recovery codes as part of S08. The
  separately authorized S08.5 non-advertiser canary enrollment/recovery checks
  remain owner-controlled and are not W27.4 acceptance.


## S07 dependency accepted — 2026-09-30

S07 closes at whole-milestone review 8/10 with no open P1/P2. W8M production
uses the protected retired schema on the unchanged immutable release; fresh
owner sign-in/display and recovery-mail/page canaries pass. The continuously
writer-frozen encrypted snapshot and complete ring were independently restored
before retirement and remain the recovery baseline. Current/previous key
retention remains separate from Identity key provisioning. Default-off rollback
for account protection requires the pre-drop schema restore; S08 rollback must
leave account protection enabled and PlaintextRetired=true. Identity is still
disabled. No S08 implementation or activation is accepted by this dependency
update. Remaining order: S08 -> W27.


## S08.1 focused production contract inspection

Read-only SSH inspection under the continuing authorization finds one manifest-
declared unify service, NTP synchronized, an HTTPS canonical application URL
and the Gmail recovery block. The fresh S07 advertiser recovery mail/page
canary passes. The owner was asked to confirm whether additional production
nodes exist and which operator-controlled non-advertiser role is available
for the later S08.5 TOTP/recovery canary; no credentials were requested.

All six S02 tables (analyst, auth_mfa, auth_recovery_code, auth_session,
auth_permission_grant, auth_security_audit) and both named immutable-audit
triggers already exist. All six tables are empty. Analyst has protected
login_hmac/login_cipher and no plaintext login, as required by S07. Therefore
S08.2 must validate existing object shapes and rehearse only required changes,
not create duplicate tables or replay the clean baseline. This inspection
alone does not prove full column/index/trigger compatibility.

The current HTTP database principal has ALL PRIVILEGES on the application
database. It therefore fails the Identity contract requiring audit SELECT/
INSERT without UPDATE/DELETE; a separate maintenance principal and restricted
application privileges must be prepared and rehearsed before activation.
Production role permissions are empty and analyst RequireGrant is false/
absent. The approved role matrix and a protected analyst issuer must be added
through the reviewed private configuration, with adv excluded from RequiredTOTP.
Identity remains disabled and no key, account, grant, schema, config or service
was mutated. Incorrect connection-shape assumptions failed read-only before
output; corrected inspection handles the configured string-array DSN and
prints no credential, principal identifier or raw grant value.

Production recovery state, S07 key retention and its proven rollback baseline
remain unchanged. S08.1 is the sole active task; no qualification, full test
suite, Identity migration or activation was started.


## Owner inventory and canary preparation — 2026-09-30

The owner confirms yixin is the sole production application node and neither
a usable publisher nor administrator account is available. The owner notes
the existing administrator-password-reset CLI. Source inspection finds
`../w8m-infrastructure/bin/reset-w8m-admin-password`: its reset action replaces
one active administrator bcrypt password using an optimistic prior-hash guard;
it does not create a canary or reset TOTP. Its status action still queries
admin.login, which S07 removed. Do not invoke that obsolete status action
against the retired schema. Before using administrator recovery, reconcile
this compatibility issue and the new Identity session-revocation requirements.
No administrator credential or session has been changed by this inspection.

The existing Pzdesign identity-admin CLI can create an audited, protected
read-only analyst through a separate maintenance configuration. Genelet's
CreateAnalyst has an explicit PlaintextRetired insertion path and an existing
focused regression test. Analyst RequireGrant gates every report resource
while allowing its own account-security actions; it is suitable for TOTP,
one-use recovery and analyst-denial checks without granting administrator
privileges to the canary. Proposed sequence: rehearse grants/config/key and
protected analyst creation on the restored fixture; prepare a maintenance-only
Unix-UID attribution to the existing active administrator; provision the
reviewed maintenance key/config; then create the owner-controlled analyst once
under S08.5 readiness and verify the live canary. The owner was asked to choose
this path or existing-administrator recovery. Credentials, seed and recovery
codes remain private; no account is created by selecting this plan.

A fresh aggregate-only SSH inspection confirms exactly one active administrator
with a protected identifier and supported bcrypt credential. The read-only
inspection receipt is outside Git in the owner-only S08 preparation store.
Identity and its six tables remain unused, and production protection remains
enabled/retired. yixin/peter is the rollout and rollback operator; the reviewed
rollback is Identity disabled with account protection still enabled/retired,
retaining tables, TOTP data, audit evidence and the protected recovery baseline.
The single node is both the canary and later production rollout target, so no
second node is inferred from a rolling-plan template.

Role policy will use the reviewed S02 five-role permission matrix, a protected
analyst issuer with RequireGrant=true, adv excluded from mandatory TOTP, and
secure cookies from the existing HTTPS canonical URL. Monitoring must cover
readiness, login/permission/recovery and audit/database failures without raw
account logs. The existing database-wide runtime privilege fails the audit
immutability contract; S08.2 must rehearse separate restricted runtime and
maintenance credentials plus audit update/delete denial. Gmail delivery and
fresh advertiser recovery continuity are confirmed by S07; identity-disabled
publisher portal and canary recovery/security checks remain S08.4/S08.5 inputs,
not completed evidence here. No live reset, enrollment, grant or activation
was performed. Whole-milestone S08 review has not begun.


## S08.1 accepted contract and task review

The owner selects Dedicated analyst canary. The administrator-password-reset
command will not be used for this canary; no existing administrator password
needs changing. Its obsolete plaintext status projection remains a documented
compatibility finding, to be reconciled before any later use rather than
ignored or treated as valid post-retirement evidence. S08.2 will bind the
explicit selected analyst path, one confirmed node, existing Identity objects,
restricted audit privileges, reviewed role matrix, separate common Identity
key, protected maintenance attribution and S07-preserving rollback.

Contract task review 1/10 finds no open P1/P2 in this inspection/preparation
unit. It identifies readiness deficiencies for subsequent tasks rather than
claiming activation: restricted database grants, runtime permissions, analyst
issuer, common key, restored rehearsal and canary checks remain required.
Documentation and diff checks pass. No application bytes, target account or
production feature changed; no test suite is needed for this contract record.
The S08 whole-milestone review counter has not started. S08.1 is accepted;
S08.2 is the next sole execution task.
