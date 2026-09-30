# Status S08 — W8M Production Identity And TOTP Activation

State: `[ ]` Planned; queued after S07.

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
| S08.1 Reconcile the exact production rollout contract | `[ ]` | Confirm all W8M nodes, active schema, application/maintenance database grants, role permission matrix, analyst `RequireGrant`, SMTP recovery, HTTPS, clock, monitoring and rollback owner. Preserve the current identity-disabled login behavior until the canary gate. |
| S08.2 Prepare and rehearse the online Identity migration | `[ ]` | Derive a reviewed migration for the six S02 tables and two immutable-audit triggers; never replay `etc/step4_init.sql` against the populated deployment. Back up and restore-test first, rehearse on a disposable baseline copy, and verify object inventory, grants, trigger immutability and existing application compatibility. No production database action is implied. |
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
- Do not modify Cloudflare/DNS, certificate, browser profile, account seed or
  recovery codes as part of S08.


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
