# Status S08 — W8M Production Identity And TOTP Activation

State: `[+]` Completed; sole-node Identity enabled with adv optional; canary and monitoring accepted; whole-milestone review 2/10 passes with no open P1/P2.

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
- Task order: S08.1 -> S08.2 -> S08.3 -> S08.4a -> S08.4b -> S08.4c -> S08.4 -> S08.5 -> S08.6.
- This status grants no database, secret, configuration, deployment, browser or
  account-mutation authority. Each production action requires its own exact
  authorization and current readiness.

## Tasks

| Item | State | Notes |
|---|---:|---|
| S08.1 Reconcile the exact production rollout contract | `[+]` | Confirm all W8M nodes, active schema, application/maintenance database grants, role permission matrix, analyst `RequireGrant`, SMTP recovery, HTTPS, clock, monitoring and rollback owner. Read-only inspection finds all six Identity tables and both audit triggers already present with zero Identity rows, but the HTTP principal has database-wide ALL PRIVILEGES and the runtime has no permissions or analyst grant requirement. Review restricted application/maintenance principals before activation; owner confirms yixin is the only node and no usable non-advertiser canary account is available. Owner selects a dedicated analyst canary via the audited maintenance CLI. Task contract review 1/10 passes; no production mutation or canary acceptance is implied. Preserve the current identity-disabled login behavior until the canary gate. |
| S08.2 Prepare and rehearse the online Identity migration | `[+]` | Reconcile the existing six S02 tables and two immutable-audit triggers against their authoritative definitions, including the S07-retired analyst schema. Derive only necessary non-destructive corrections and restricted application/maintenance grant changes; do not recreate matching objects or replay `etc/step4_init.sql` against the populated deployment. Back up and restore-test first, rehearse on a disposable baseline copy, and verify object inventory, grants, trigger immutability and existing application compatibility. Accepted: restored-copy schema/grant stages, fresh corrected protected analyst CLI, authoritative six-table/trigger comparison and preparation review pass; failed whole attempts remain failed. Matching Identity objects need no recreation. No production database action is implied. |
| S08.3 Apply schema and verify readiness | `[+]` | Window 02 passed after fresh continuously frozen backup, independent complete restore/runtime-ring recovery and credential escrow recovery. Separate unused runtime/maintenance principals have exact reviewed rights; both audits deny forbidden updates/deletes. All 96 tables’ counts/checksums and schema inventory remain identical. Matching Identity objects need no DDL. Original HTTP credentials/configuration stay unchanged; Identity remains disabled, account protection retired. Writer resume/readiness 204 and task review 1/10 pass. Failed window 01 remains failed before restore/grants. |
| S08.4a Reconcile deployment admission with the retired account schema | `[+]` | Offline prerequisite accepted: newly built v2 bundles declare retirement support; target preflight requires protected/retired configuration and exact read-only retired account/routine metadata. Clean six-routine baseline and older immutable manifests remain unchanged. Ten public preflight cases, manifest compatibility and seven shell verifier cases pass; the exact metadata query passes read-only on yixin. Affected deployment tests, vet, syntax, docs and diff checks pass. Preparation review 1/10 has no open P1/P2. Local source commit is authorized under task policy; clean publication/build and live application remain S08.4 requirements. |
| S08.4b Correct the frozen-build frequency-cap fixture timezone | `[+]` | Release build 02 stopped at TestFcap: its UTC input disagreed with assertions for local legacy fields on the Los Angeles workstation. The fixture now constructs local midnight; production code and assertions are unchanged. The exact test passes freshly under UTC, Los Angeles and Shanghai, and the affected match package passes in 0.101 seconds. Preparation review 1/10 has no open P1/P2. This accepts the fixture source correction, not the failed release build; a fresh clean published source/build remains required by S08.4. |
| S08.4c Prepare target-owned deployment and administrator compatibility support | `[+]` | The manifest names the future required Identity environment file and two retained production routines; the drop-in contains only the secret-file path. Retired-schema numeric administrator status works, and Identity-enabled direct resets fail before password/account access. The guard validates/reasserts an inherited exclusive descriptor for nested adapter calls without downgrading it. Focused local/remote kernel and synthetic administrator fixtures, strict manifest validation, source-byte comparison and full browser-free backend conformance pass on yixin using supplied offline sources/Go. Preparation review 1/10 has no open P1/P2. Infrastructure source publication excludes the unrelated layout migration; live installation remains S08.4 work. |
| S08.4 Deploy identity-disabled release and provision common key | `[+]` | Accepted: thirteen-file independent recovery, immutable capable release installation through infrastructure, restricted shared runtime DB, common running-service/maintenance key parity and read-only maintenance UID/socket startup pass. HTTP Identity stays false; account protection and the retained ring remain unchanged. Fresh owner advertiser sign-in/display, recovery mail/page without password change, Publisher registration/email activation and both portal entries pass. NTP, HTTPS secure-cookie configuration and reviewed role permissions pass. Task review 1/10 has no open P1/P2; whole-S08 review remains unstarted. |
| S08.5 Enable Identity on a canary | `[+]` | Accepted under exact authorization: sole-node Identity enabled with adv optional; audited analyst creation without grants; owner enrollment, fresh TOTP, POST logout/CSRF and cross-account/analyst/grant denials pass. Recovery first use and reuse refusal are verified by owner results and separate audit/session counts. Both controlled idle/absolute expiries force HTTP sign-in; fresh TOTP restores valid access. Final health/key/NTP and preserved disabled rollback/recovery pass. Task review 1/10 has no open P1/P2. Dedicated advertiser enrollment remains W27.4. |
| S08.6 Roll out, monitor and close readiness | `[+]` | Accepted: yixin is the sole node and already has the approved exact canary activation; no second service window is required. Final common/protection key, config/release, schema/grant, NTP, health and preserved rollback bindings pass. The 43.7-minute existing activation/canary window has expected login/recovery/denial audits and no observed failure indicators. Reader preparation stops remain preserved; exact inherited-log/host-selector corrections pass focused checks. Whole-S08 review 2/10 has no open P1/P2. Sanitized readiness handoff supports W27.3 read-only verification and grants no later live authority. |

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


## S08.2 isolated rehearsal preparation

Prepared a private fresh rehearsal kit outside Git for the exact MySQL
8.0.41 image. The baseline is S07's verified encrypted final frozen snapshot,
restored into a network-disabled tmpfs fixture; its complete original inventory
must match before applying the exact S07 retirement inside that fixture.
Current live metadata is inspected read-only, and the retired fixture must
match its tables, columns, indexes, keys, triggers and routines (excluding only
data-dependent AUTO_INCREMENT counters). Matching existing Identity objects
are preserved. The older snapshot is a rehearsal baseline and does not replace
a fresh independently restored frozen backup before S08.3 production changes.

The freshly built existing identity-admin command uses the pinned Go 1.23.5
SDK with fetching disabled. Pzdesign and Genelet worktrees are clean; the CLI
and its relevant dependency source bytes match the selected deployed commits.
Private preparation review 1/10 checks syntax, exact host/image and retired/
Identity-disabled guards, exclusive fresh paths, isolated networking/storage,
source inventory and live schema comparison, runtime audit denials even with
the retention variable, separately gated maintenance deletion, protected
analyst creation/audit, unrelated-data preservation and joined cleanup.
No open P1/P2 within this prepared core rehearsal scope. Whole-milestone S08
review has not started. The estimated fixture run is 2–5 minutes after one
masked GPG backup-key unlock; it is not a browser qualification or full suite.
The run is waiting at that backup unlock. No production account/password,
Identity key, grant, schema, configuration or service has changed.


### S08.2 stopped transfer and focused correction

The owner completed the masked GPG backup unlock successfully. Rehearsal-01
then stopped at static CLI binary transfer: the 9,862,651-byte binary exceeded
the launcher's 30-second SCP timeout after 2,611,200 bytes. No fixture, SQL
rehearsal, canary account or production mutation started. The partial binary
is retained under a distinct failed-transfer filename and is not executable
or eligible for use. Cleanup confirms no fixture or private tmpfs remains;
the temporary protected keyring and GPG unlock session were closed.

The correction begins with only the failed transfer: losslessly compress the
unchanged binary, transfer into a fresh output path with a 180-second bound,
and verify the compressed and recovered binary digests before any execution.
No backup passphrase is requested for that focused transfer. After it passes,
a fresh rehearsal ID must use the already verified uploaded bytes rather than
repeat the upload after unlocking secrets. Earlier evidence remains intact;
whole-milestone review has not started and S08.2 remains incomplete.


### Focused transfer passed; fresh rehearsal-02

The unchanged CLI binary compressed to 5,494,281 bytes and transferred in
12.23 seconds. The server verifies both compressed SHA-256 and recovered
9,862,651-byte binary SHA-256; the earlier partial transfer remains preserved.
The eligible binary is copied into a fresh rehearsal-02 stage on the same
host without another network upload. No backup unlock or fixture was needed
to establish this focused correction.

Private preparation review 2/10 resolves the too-short transfer bound and
moves every static upload and exact binary-hash verification before the
masked GPG unlock. It also applies the existing S03 contract: runtime grants
on api_audit, as on auth_security_audit, are SELECT/INSERT only, with deletion
reserved to separate gated maintenance. The fresh restored-copy rehearsal
checks both denial families alongside the required protected analyst path.
All scripts parse; no open P1/P2 in this prepared scope. Original inputs and
earlier stopped evidence remain intact; S08 whole-milestone review has not
started. The failed run closed its GPG session, so rehearsal-02 needs one
fresh masked backup passphrase entry. No W8M account password is requested.
Production Identity is disabled and no production account, grant, config,
key or schema has changed.


### Rehearsal-02 reached the analyst CLI, then stopped

The new backup unlock passes. The complete restored inventory, retired/live
schema comparison, both audit privilege-denial families, both maintenance
retention gates and unrelated runtime transaction pass before the analyst CLI
returns nonzero. The failure and cleanup are preserved; the fixture and private
tmpfs were removed and the GPG session closed. No production mutation occurred.
The helper captured only its generic wrapper assertion, not the underlying
CLI stderr; this missing diagnostic is explicitly recorded rather than
claimed as detailed evidence.

Source inspection finds that the rehearsal's minimal maintenance configuration
omitted Roles entirely, while NewAccountProtector requires at least one
complete protected password issuer. The next probe will reproduce only that
CLI/config handoff using synthetic keys/data and a minimal isolated database,
retain its exact underlying error, then verify the corrected full role contract.
It needs no GPG unlock or full backup restore. No application bug or production
readiness is concluded from the incomplete rehearsal; S08.2 remains open.


### Focused analyst CLI handoff-03 passed

The fresh minimal network-disabled MySQL 8.0.41 fixture reproduces the exact
underlying error: AccountProtection requires at least one protected password
issuer. The missing-Roles configuration creates no analyst. Adding the full
authoritative five-role issuer contract to the otherwise unchanged synthetic
configuration succeeds: the actual existing identity-admin CLI creates exactly
one protected active analyst and its AnalystCreated success audit event. Exact
CLI stdout/stderr are retained privately before assertions; no actual account
values, backup keys or password inputs are used. The fixture and private
synthetic environment are removed and cleanup verified. No GPG prompt, full
backup restore, browser or broader suite was needed for this failing-stage
correction. No source application bug was found.

The failed rehearsal-02 remains failed. Its reviewed remote script hash matches
the frozen local source; the traceback at its CLI assertion establishes that
the preceding restore/schema/grant and retention checks passed. These stages
and the separate successful focused CLI/config correction are recorded with
their individual scope and source identities, not rewritten as one passing
original attempt. S08.2 acceptance still needs the corrected full-role contract
integrated into the maintained candidate and a complete preparation review.
Production Identity, credentials, grants, schema and accounts remain unchanged.
No additional owner input is needed for this focused diagnostic.


### Authoritative schema verification and S08.2 acceptance

The schema-only source fixture matches all six Identity table definitions,
columns, indexes and keys, including S07's retired analyst adjustment. Its
initial trigger comparison differs because the creation context was not fully
bound. That mismatch remains preserved. A fresh trigger-only fixture binds
the original SQL mode, client charset/connection collation and database
collation; both authoritative audit trigger definitions and all compared
metadata then match production exactly. Only the unresolved trigger context
was rerun; no backup unlock, full restore or browser suite was repeated.
All fixtures are removed. Matching production objects require no migration
DDL or recreation.

The maintained private candidate now includes the corrected complete Roles
issuer contract and retains exact CLI stderr before assertions. Complete
preparation review 4/10 and S08.2 task review 1/10 find no open P1/P2 in this
unit. Evidence is explicitly composed from the passed restored-copy schema/
grant stages, fresh protected-analyst CLI correction and authoritative source
comparison. Failed original attempts remain failed; no fresh passing whole
rehearsal is invented. Source application bytes are unchanged. Documentation
and diff checks pass. S08.2 preparation acceptance passes; its task commit
will preserve the unrelated README edit. S08 whole-milestone review has not
started and no production readiness, canary or enrollment is accepted.

S08.3 next needs a fresh continuously frozen encrypted production backup and
independent restore before reviewed restricted credential/grant provisioning.
The earlier snapshot is only the rehearsal baseline. Existing tables/triggers
are preserved, active runtime configuration stays Identity-disabled until
S08.5, and S08.4 must fix/guard the legacy administrator-reset CLI before
Identity activation. No existing administrator password reset is required by
the selected analyst canary. No further owner input is needed for the focused
CLI correction just completed.


## S08.3 disabled production packet preparation

S08.2 acceptance is committed locally as `fcb5cc2`. The unrelated README
change remains outside the task commit; unpublished layout history is still
local. The next sole owner is S08.3. Prepared a disabled private packet binding
the exact one-node host, release, database image and accepted schema contract.
It specifies a fresh exclusive continuously writer-frozen encrypted snapshot
and independent complete restore/recovery before provisioning separate reviewed
runtime and maintenance credentials. Matching Identity objects require no DDL.
Active HTTP credentials and Identity-disabled configuration remain unchanged
through this stage; later S08.4 owns their reviewed switch and common Identity
key provision. Credential material is generated/escrowed privately, with no
command-argument or log exposure. Unknown mutation outcomes stop without retry
or deleting potentially created principals. Existing protected retired state
and S07 previous-key retention remain mandatory.

No producer, live grant/credential change, account creation or activation has
started. The disabled packet still needs its producer/verifier implementation,
focused readiness/failure checks and bounded preparation review. The owner
needs no further input for the completed S08.2 diagnostic; any later masked
GPG prompt belongs to the fresh S08.3 recovery checkpoint, not account login.


## S08.3 producer/verifier preparation accepted

The existing cross-package goal is resumed under the unchanged task commit
policy and exact S07/S08 host/database/service authorization. S08.3 is the
sole execution owner; S07 and S08.2 remain accepted, and W27.3 remains blocked.
Private one-window producer/verifier sources are prepared and hash-bound outside
Git. They hold the existing exclusive runtime-root lock, stop the recorded
writers, create a fresh encrypted database/runtime/key-ring snapshot, and gate
unused credential provisioning on independently verified complete restore and
credential escrow. Active HTTP credentials/configuration and Identity-disabled
state remain unchanged. Matching objects need no DDL.

Preparation review 1/10 found two P2 preflight defects: MySQL 8 exposes routine
privileges through mysql.procs_priv, and the owner-owned systemd unit is mode
0644 rather than the credential files' 0600. Both are fixed. Review 2/10 found
a P2 freshness gap between escrow verification and CREATE USER; immediate
binlog, connection, schema and config checks now close it. Preparation review
3/10 passes with no open P1/P2; this is separate from the still-unstarted
whole-milestone review.

Six pure grant contract checks pass. A small fresh MySQL 8.0.41 fixture verifies
both principals authenticate, exact table/routine/global/role rights are read
correctly, forbidden audit changes fail, and broad schema rights are rejected.
Its changed grant-reader checks use a fresh second fixture; both are removed.
Focused mocked window checks cover abort before backup, restore/escrow failure,
partial grant failure with no retry/drop, successful writer resume, and drift
immediately before grant consumption. One initial mocked test setup omitted
its inventory file and was corrected; it touched no host/database. Python and
Bash syntax pass; read-only production preflight passes. No broad browser suite
or application source change is involved.

The next exact operation is the fresh frozen backup/recovery checkpoint followed
by unused restricted runtime and maintenance principals. One new masked GPG
unlock precedes writer stopping; cancellation or unlock failure stops without
retry. Task acceptance remains pending until private live receipt, inventory
preservation and resumed readiness are verified. No production grant, account
creation, Identity activation or W27 browser operation is accepted by this note.

The reviewed S08.3 launcher is now live and awaits its masked recovery-key
unlock before stopping writers. Its handle is recorded privately; resume that
same process after input rather than launching another operation. No unlock,
backup or production grant acceptance is yet inferred from launch alone.

Three consecutive resumed goal turns confirm the same pending recovery-key
dialog and live local launcher. The native goal is blocked awaiting that
required owner input; the existing process/dialog remains live. Read-only
production inspection confirms active service, readiness 204 and no S08.3
remote window started. Entering the passphrase resumes this same supervised
launcher; do not start a new producer, re-prompt or reuse a consumed window.
Task and milestone acceptance remain pending.


## S08.3 first window stopped; recovery input accepted

The owner submitted the saved GPG passphrase successfully. The same launcher
then acquired the exclusive lock, froze the recorded writers, and produced
the fresh encrypted database/runtime snapshot. It stopped before starting the
independent restore because its Docker absence check expected the capitalized
`No such object` diagnostic; this host returned the exact lowercase
`error: no such object: <fixture>`. This is a launcher defect, not passphrase
expiry or failed decryption. Original process/session is terminal (exit 1),
and all original evidence remains preserved.

Producer input closure resumed the original services. The private resume/stop
receipts confirm grant_attempted=false, and fresh read-only SSH confirms active
HTTP service, readiness 204 and absence of the grant-started marker. No new
principal, account, Identity activation or browser operation occurred. The
temporary protected keyring and unlock session are closed.

A separate focused candidate corrects only the fixture absence recognizer. It
requires the precise fixture name, exit 1 and empty inspection array, accepts
the host's lowercase or historical uppercase diagnostic, and still rejects
existing fixtures, transport/daemon failures, different names and extra errors.
Three focused cases and a read-only SSH check at the exact original failure
spot pass; syntax passes. The consumed launcher is unchanged and no new live
window or unlock dialog was launched. Subsequent provisioning still needs a
fresh reviewed frozen window and independent recovery; the earlier stopped
snapshot is preserved and is not accepted as continuously frozen grant evidence.
The old native goal's waiting-for-input explanation is now obsolete; input was
accepted, and S08.3 remains incomplete on this diagnosed launcher defect.


## S08.3 corrected fresh window 02

Continued the existing exact backup/database/service authorization after the
known first-window precondition failure and verified service recovery. The
original grant was never attempted and no account submission is repeated.
Prepared fresh private producer/verifier record paths and a new disposable
restore fixture; no consumed path, source or evidence is overwritten. The only
behavioral correction is the exact missing-fixture diagnostic recognizer.
Three focused recognizer cases pass; two restore-checkpoint integration cases
prove the actual lowercase error reaches startup and an SSH failure stops
before startup. Fresh production preflight and new-fixture absence pass.
Private preparation review continues at 4/10, with no open P1/P2. The earlier
passing grant/failure/resume checks retain unchanged affected inputs; no full
browser suite is run. Whole-S08 review remains unstarted.

Window 02 requires one new masked recovery-key unlock because the stopped
window's temporary unlock session was closed. Unlock occurs before service
stopping. It will independently restore a newly frozen snapshot and recover
the complete runtime ring and unused credential escrow before attempting the
reviewed grants. Old failed results remain failed. Live task acceptance is
pending until exact private receipts, preserved inventory and resumed
readiness are verified.

Window 02 launcher and its masked dialog are confirmed live on the ordinary
workstation display, awaiting input before production writer stop. Its private
handle is retained; resume this exact process after entry, never start another
window while it is live. No fresh backup, grants or task acceptance is inferred
from launch alone.

Three consecutive goal turns verify window 02 remains at its required masked
recovery input. The launcher/dialog are live, no unlock checkpoint file exists,
and fresh read-only production inspection confirms active service, readiness
204 and no remote window-02 operation directory. The native goal is blocked
on that exact input; the same dialog/process remains open. No expiry, failed
passphrase or task acceptance is inferred. Resume the existing handle after
entry; do not create another window or dialog.


## S08.3 window 02 accepted — 2026-09-30

The owner submitted the new recovery-key entry successfully. Window 02 completed
the fresh continuously writer-frozen encrypted snapshot, independent exact-image
restore and complete runtime-key recovery before grant consumption. The newly
generated credential recovery archive was independently decrypted and matched
before provisioning the separate unused principals. Exact table/routine/global/
role/proxy privilege checks and production audit denial probes pass. All 96
tables' counts/checksums and schema dump inventory match the source, restored
copy and post-grant production inventory; objects remain 96 tables, 2 routines,
65 triggers and no events. No schema DDL was needed.

The original HTTP credentials, immutable release, runtime configs, Identity-
disabled state and protected retired account schema remain unchanged. Writer
resume and fresh read-only readiness 204 pass. The restore fixture is removed;
private credential files are owner-only; temporary recovery keyring/session is
closed. Task review 1/10 passes with no open P1/P2; whole-S08 review is still
unstarted. This accepts database preparation, not Identity activation, canary
creation, advertiser enrollment or W27 readiness.

Sanitized infrastructure history preserves both
[failed window 01](../../../w8m-infrastructure/history/20260930-s08-window-01-stopped.json)
and [passed window 02](../../../w8m-infrastructure/history/20260930T223843Z-s08-restricted-principals.json).
The former remains failed. No consumed credential/account claim was retried.
S08.4 next owns the reviewed identity-disabled runtime credential/configuration
switch, common Identity key and recovery proof, remaining baseline checks and
legacy admin-reset compatibility guard before canary activation.


## S08.4 readiness preparation started

S08.3 acceptance is committed locally as `3b569e1`; its stopped and passing
operation histories are committed/pushed by infrastructure as `dfcdf59` and
`be2a959`. Unrelated Aofei README and infrastructure layout migration remain
excluded. No Aofei push traverses the unrelated unpublished layout ancestor.
S08.4 is now the sole execution owner. Its first offline change reconciles the
legacy administrator-reset status with retired identifiers and refuses direct
unaudited non-revoking resets when Identity is enabled, serialized with the
existing runtime-root transition lock. No administrator password reset is
needed for the chosen analyst canary and none is invoked. Common Identity key,
reviewed runtime credential/configuration switch and remaining baseline checks
remain preparation items, not completed acceptance.

The offline administrator compatibility patch is prepared in infrastructure.
Status now queries numeric ID/active/bcrypt metadata only and works with the
retired schema. Direct reset checks the owner-only Summer configuration under
the exclusive runtime-root lock and refuses Identity-enabled operation before
account lookup/password input; read-only status uses the shared guard. Focused
fixture cases cover Identity refusal, malformed configuration, transition lock
contention, existing password validation and optimistic concurrent-update guards.
Workstation verification lacks htpasswd; the already installed server tool is
used only inside a supplied synthetic shell/SQL fixture, never a production
reset. Its first fixture attempt correctly rejects group-writable directories
from the server's default umask 0002; a fresh fixture sets umask 077 and passes.
Both preparation attempts and source kits are distinct; no real account is
modified. Syntax/diff checks pass. The code/docs remain an uncommitted S08.4
support patch pending readiness review and task acceptance.


S08.4 private configuration preparation now preserves the existing protected
issuers, account-protection ring, SMTP and deployment settings while binding
both Aofei and Summer to the staged restricted runtime principal. The analyst
role uses the authoritative protected issuer/permission contract with
RequireGrant=true. Identity remains disabled and adv is excluded from required
TOTP. Disabling legacy password upgrades requires a fresh verified bcrypt-only
baseline. Four small synthetic transformation checks pass; no browser suite is
involved.

The private read-only configuration validator builds offline against the
existing Pzdesign/Genelet module graph. An incorrect initial module import was
corrected to the existing declared module; no dependency was fetched or changed.
The build is preparation evidence only: no candidate configuration, Identity
key provisioning or live database/factory validation is accepted yet. Actual
secret delivery uses installed systemd drop-ins, so the next recovery checkpoint
must preserve their effective files as well as the main unit. Fresh read-only
SSH confirms window 02 passed, active service and readiness 204. The previous
owner recovery input succeeded and its temporary unlock session is closed.


## S08.4 deployment-contract finding

Read-only inspection confirms the selected immutable release still declares
six baseline routines, while S07's accepted retired database has two. The
existing generic deployment engine requires exact equality, so a reviewed
production manifest with two routines cannot pass that release preflight.
S08.4 owns this readiness correction before any key/config switch: preserve
immutable old manifests and the six-routine clean baseline; newly built releases
must explicitly declare support for account-identifier retirement. Admission
of the two-routine target also requires protected/retired runtime configuration
and read-only proof of the exact retired account/routine shape. Older bundles
without that capability remain inadmissible for retired-target deployment.
No schema operation or live configuration change is implied. Focused preflight
checks precede release publication/build and the recoverable S08.4 switch.


## S08.4a offline deployment prerequisite accepted

The correction declares `supports_account_identifier_retirement` only in newly
built v2 releases. Old manifests without the field remain readable and
unchanged. Preflight admits only the exact six-to-two routine transition,
with both protection flags true, retained slot procedures, no retired
plaintext account/history identifier columns, and all ten non-null protected
identifier columns across five account roles. Other schema/accounting
contracts stay exact. A partial schema, missing column, wrong procedure,
unsupported count, failed metadata query, absent capability or disabled
protection stops before activation/history mutation.

Ten cases exercise the public checksum/manifest/target preflight path with
synthetic runtime/release fixtures. Older manifest compatibility and explicit
capability restrictions pass; seven cases exercise the shell verifier's actual
predicate. The exact new metadata SQL passes read-only against production.
Affected deployment package/CLI tests pass (about 1.35 seconds); vet, shell
syntax, documentation guard and diff checks pass. No long browser suite is
run. Offline prerequisite preparation review 1/10 passes with no open P1/P2;
whole-S08 review remains unstarted. S08.4a is a subordinate completed source
unit, allowing task-level publication without falsely completing S08.4.

S08.4 remains the sole live-readiness owner: prepare a clean published capable
release, independently recover the complete drop-in/key/config checkpoint,
then apply the authorized Identity-disabled configuration and complete fresh
baseline checks. No release build, service switch, Identity key provisioning,
Identity activation, canary creation or W27 acceptance is inferred here.
Unrelated Aofei README/layout ancestry and infrastructure migration remain
preserved and excluded from source publication.


S08.4a is committed locally as `9967e6b`. Its exact generic source bytes are
published as `0753fd54c505` on the separate
`s08-retired-deployment-20260930` branch based on published main. The branch
excludes the unrelated local layout ancestor and earlier production-ledger
commits; it preserves old-layout current facts with scoped appended source
contracts. Remote identity, identical accepted source bytes and excluded
ancestry are verified privately. The owner checkout remains the single
S08 execution ledger; no main push or history rewrite occurred.
A clean independent source set and capable release build are next S08.4
prerequisites. Production release/configuration and Identity state are unchanged.


## S08.4 capable release preparation

Source-copy preparation 01 stopped before tests: the launcher incorrectly
selected Pzdesign main, while its exact published checkout tracks origin/master.
No release, target operation or key dialog started. The failed preparation and
launcher bytes remain preserved privately. Corrected preparation 02 selects
the verified master branch, creates fresh independent copies of all three
published source commits, verifies no alternates/dirty source, and starts the
normal release builder. It runs the builder's required Go suites and immutable
artifact verification, not W27 browser qualification. Its live handle is
recorded privately; observe the same process, never restart on a polling timeout.
S08.4 acceptance remains pending until the build, recoverable key/config switch
and fresh baseline checks pass. No production mutation is inferred from build
start. The standalone source-copy defect does not change accepted S08.4a bytes
or the whole-S08 review counter.


## S08.4b frozen-build fixture correction

Clean release build 02 terminated with exit 1 after 48.502 seconds at the
Aofei source-test stage. Only TestFcap failed: StartYM was 252 rather than 1,
and StartDHM was 64512 rather than 2048. The fixed input 2025-01-01 00:00 UTC
is 2024-12-31 16:00 in Los Angeles. The production legacy compatibility
prefix intentionally encodes local wall time; the fixture incorrectly
expected UTC calendar bits. The authoritative v2 UTC minute representation
and production implementation are unchanged.

The fixture now constructs midnight in time.Local, preserving all existing
assertions. Fresh exact-test checks pass under UTC, America/Los_Angeles and
Asia/Shanghai; the affected match package passes freshly in 0.101 seconds.
Preparation review 1/10 has no open P1/P2. The failed build's cloned sources,
output, launcher and terminal receipt remain preserved and unchanged; no
release output or production/key operation occurred. Whole-S08 review remains
unstarted. S08.4b is a completed source prerequisite; S08.4 still requires new
clean publication, a fresh passing release build and live readiness acceptance.


S08.4b is committed locally as `c49fdf0`. Its exact fixture correction is
published as `96456a2bf9bb`, fast-forwarding only the separate S08 source branch.
The unrelated layout ancestor remains excluded. The existing owner checkout
retains task/status authority. A fresh third source/build directory is prepared
for the unchanged normal builder after the exact failed test and affected
match-stage checks pass. Preparations 01 and build 02 remain failed; no old
source copy, output or receipt is overwritten. The third build is not accepted
until its own terminal receipt and immutable artifact verification pass.

Fresh build 03 has passed independent source-copy preparation and its recorded
normal builder is confirmed live. Observe that same private handle. Production
remains unchanged, Identity disabled, and no recovery-key dialog is open.


Fresh build 03 passed in 276.203 seconds, producing immutable release
`aofei-96456a2bf9bb_pzdesign-6767db40d0ff_genelet-fdeff9732d9e`. All three
required Go suites, binary/source bindings, asset inventory and release
verification pass; supplied sources remain clean and unchanged. Failed
preparation 01 and build 02 remain failed. No production change or key
provisioning occurred.

The private configuration window holds the existing exclusive runtime-root
guard, but the current thin deployment adapter opens a new shared guard
descriptor. That nested invocation would refuse rather than execute. S08.4
therefore needs a focused supporting correction before live handoff: validate
and reassert a supplied inherited descriptor as exclusive, without downgrading
the outer lock, while retaining the existing default shared/exclusive behavior
for ordinary callers. The target-private manifest/drop-in and administrator
compatibility guard stay within infrastructure ownership.


## S08.4c target-private support accepted

The supporting infrastructure patch reconciles administrator status with
retired identifiers and refuses unaudited, non-revoking direct resets once
Identity is enabled. Both actions serialize with the runtime-root guard. A
supervised configuration window can supply its exact inherited descriptor to
the thin deployment adapter; the helper checks canonical inode/owner/mode and
reasserts exclusive ownership, preserving the outer guard. Wrong, closed,
replaced or peer-writable descriptors fail without a fallback acquisition.
Ordinary callers retain their existing shared/exclusive behavior.

Local and remote real-kernel lock fixtures pass in under a second. The updated
administrator fixture passes with synthetic SQL/password state and the server's
already installed htpasswd; no production account command is invoked. The
newly built generic binary validates the planned target manifest. A fresh
supplied source snapshot and hash-matching offline Go 1.23.5 run full backend
conformance successfully on yixin, including ownership/current/reachable-history
exclusion, focused deployment tests, release verification and synthetic
bootstrap/migration/admin fixture checks. Tested executable/manifest/drop-in
bytes match the owner checkout. No production configuration, service, account,
key or browser operation changes. Preparation review 1/10 has no open P1/P2;
whole-S08 review remains unstarted.

S08.4c accepts source support only. Infrastructure publication must exclude its
unrelated uncommitted layout migration and preserve D00–D05 history. S08.4
still owns the recovered configuration/key checkpoint, clean adapter handoff,
live Identity-disabled installation and fresh baseline acceptance. The private
checkpoint reader now has seven passing synthetic checks for complete recovery,
missing drop-ins, corruption, links, HTTP activation, credential leakage and
wrong key length. It has not generated or recovered a production Identity key.


## S08.4 encrypted runtime checkpoint prepared

Infrastructure support is published independently as `dabdb783d3a0` on
`s08-readiness-support-20260930`; the unrelated layout migration and D-lane
history are excluded. Its clean source checkout joins the three published
backend revisions used by release build 03. Build 03 passed in 276.203 seconds;
that unchanged build and server conformance evidence are reused.

Fresh read-only readiness, exact restricted grants/audit denials, schema,
clock, bcrypt baseline and the unique maintenance-actor checks pass before
private preparation. One new Identity key and candidate runtime/maintenance
configurations are generated outside Git. The actual production Go factories
and restricted runtime database read pass. HTTP Identity stays false; the
maintenance-only configuration carries the separate restricted principal and
Unix-UID attribution. The encrypted thirteen-member checkpoint includes the
original runtime files, actual Gmail/protection drop-ins, both candidate HTTP
configs, new key/drop-in, maintenance config and staged credential provenance.
Preparation takes 5.268 seconds; encryption is digest-verified on transfer.

Preparation review 1/10 has no open P1/P2. Independent owner-key decryption and
in-memory comparison remain required before any installation. A new masked
GPG recovery entry is needed because the accepted S08.3 temporary unlock session
was closed; it is not an account password or TOTP prompt. No service stop,
configuration installation, release activation or Identity enablement occurred.
S08.4 remains in progress and whole-S08 review remains unstarted.


## S08.4 clean target handoff and recovery input

The clean published four-repository source cohort and built release are copied
to a fresh private target kit. Initial release verification stops on directory
modes created by Python's data-filter extraction; the follow-up byte comparison
also detects the four Git index stat caches updated by its own clean-tree check.
Both stopped verification records are preserved. Explicit archive modes,
restored index caches and GIT_OPTIONAL_LOCKS=0 correct only the supplied kit;
all payload bytes then match, all four sources are clean/published, the release
verifies and the target manifest validates. The focused corrected stage passes
in 4.633 seconds, without rebuilding or repeating backend/browser suites.
Transfer and first verification take 135.427 seconds; archive preparation takes
12.322 seconds. No production file/service/release changed.

The maintenance identity-admin client is built offline from the same published
Pzdesign revision as the accepted release, in 3.706 seconds. Its bytes stay in
private preparation storage; no account command or maintenance installation is
performed. Documentation and diff checks pass for the readiness notes.

The independent checkpoint recovery launcher is live as local PID 3928199,
with its single masked dialog PID 3928200 and tool session 60923. Its durable
private handle is checkpoint-01/recovery-live-handle.json. Resume that handle
after owner input; do not launch another dialog or infer success from waiting.
It verifies the thirteen encrypted members in memory and removes the temporary
protected keyring. Until its receipt passes, S08.4 installation remains gated.
Whole-S08 review remains unstarted.


## S08.4 maintenance client target readiness

The newly built published-source maintenance CLI is copied to the private target
kit, its SHA-256 matches, and its help path executes successfully on yixin.
Transfer and executable readiness take 18.611 seconds. This checks the actual
host binary without a config, key, database call or account mutation; it does
not accept the later maintenance factory or canary operation. The existing
independent recovery process and masked dialog remain live (same PID/session),
with no recovery receipt yet. Production installation still waits for that
owner input; no duplicate dialog or operation is launched.


## S08.4 private operator file-window preparation

A private file-window helper prepares atomic owner-only config replacement and
explicit rollback under the existing exclusive guard. It rejects drift, links,
unsafe paths and colliding new destinations; a consumed object cannot replay.
The bounded helper review finds and fixes three P2 preparation issues at
iteration 1 (replay after rollback, atomic no-clobber creation and file-type
checks before rollback reads). Iteration 2 has no open P1/P2. Nine focused local
filesystem checks pass in 0.078 seconds, including post-rename failure,
restoration, collision, drift and FIFO refusal. Earlier checks are retained;
no backend/browser suite is repeated.

The separate private operator wrapper compiles and requires the exact passed
independent recovery receipt, original file/source bindings, fresh readiness
and a new exclusive consumed window. It delegates release selection to the
published infrastructure adapter and generic engine. It waits for any launched
adapter to terminate before deciding on file rollback; unresolved selection or
post-deploy verification stops without a retry. Source freezing, final wrapper
preparation review and actual execution remain pending. The existing recovery
launcher/dialog are still live; no independent recovery receipt, production
installation or Identity activation is inferred. Whole-S08 review stays
unstarted, and S08.4 remains in progress.


## S08.4 assembly ready; owner recovery input still required

Final operator source bindings are frozen and preparation review 1/10 has no
open P1/P2; this is separate from whole-S08 review. The preserved Gmail
assignments parse successfully. A fresh check-only execution of the actual
wrapper on yixin refuses the missing independent recovery receipt before
creating any activation window. All original runtime hashes remain equal,
new live Identity paths remain absent and the service stays active. That
focused gate check takes 3.796 seconds; it does not consume an activation
attempt or authorize Identity/account/browser operations.

The same required masked GPG input has remained unresolved across three
consecutive goal turns. The actual recovery launcher PID 3928199 and dialog
PID 3928200 are confirmed live again; neither a passed nor stopped recovery
receipt exists. Independent preparation is ready. The native goal is blocked
on this owner input, while the existing dialog is retained. Resume its same
session 60923 after input, read the recovery receipt and recheck readiness
before invoking the separately consumed disabled runtime window. Do not open
another prompt, reuse a consumed window or treat the missing receipt as success.
S08.4 remains in progress; production Identity stays disabled, and later S08
and W27 acceptance are pending.


## S08.4 disabled installation and baseline acceptance — 2026-10-01 UTC

The previously blocked owner GPG entry completes the existing recovery process.
Independent recovery passes for all thirteen checkpoint members in 0.472 seconds,
with the common Identity key, original protection ring and separate credential
boundaries proved in memory. The temporary protected keyring is removed; no
second recovery dialog or replay is needed. The earlier missing-receipt refusal
and native blocked-state notes remain historical evidence.

The fresh consumed activation-01 delegates to the normal published
infrastructure adapter. It installs release
`aofei-96456a2bf9bb_pzdesign-6767db40d0ff_genelet-fdeff9732d9e` in 14.605 seconds.
Both HTTP configurations use the reviewed restricted runtime principal; the
recovered Identity environment file, path-only drop-in and separate restricted
maintenance configuration are installed. The running process has the matching
Identity key and unchanged account-protection ring. HTTP Identity stays false,
account protection stays enabled/retired, readiness and health return 204, and
both public origin checks pass. The immediately prior immutable release and
complete recovered config checkpoint remain available for rollback; rollback
must preserve retired protection and database evidence.

The same published Pzdesign maintenance client reaches UID attribution,
restricted Unix-socket DB access and the Identity/common-key factories, then
refuses a deliberately unsupported action before any mutation branch. Startup
passes in 1.753 seconds, all six Identity table counts remain unchanged, and
its private transient container files are removed. No analyst, TOTP enrollment,
password reset, grant or Identity activation occurs in this probe.

The owner confirms fresh retained advertiser sign-in/dashboard/account display,
a fresh recovery email opening at Set New Password without changing the
password, new Publisher registration with email receipt and successful
activation, and normal display of both portal entry pages. These are manual
backend baselines, not W8M automated-registration or count acceptance. The
conditional recovery-code field is existing template behavior, not evidence
that Identity is enabled. No publisher sign-in is claimed from portal display.
Fresh read-only checks prove unchanged installed config/key bindings, reviewed
role permissions, analyst RequireGrant, adv excluded from required TOTP, HTTPS
secure-cookie configuration, NTP synchronization and readiness 204.

The credential-free infrastructure deployment record is independently reviewed
and published as history-only commit `b7c22bd6c860` on the scoped S08 support
branch. Its remote ref matches; the unrelated layout migration is excluded.
The original source cohort and engine-emitted record remain unchanged.

S08.4 task acceptance review 1/10 checks recovered originals/candidates, exact
published release/source bindings, restricted runtime/maintenance separation,
consumed-window and child-resolution behavior, protected-data rollback,
operational history and owner baseline evidence. No open P1/P2 remains. The
unchanged passing build/conformance/focused tests are reused; only the affected
live readiness stages run fresh. Whole-S08 review is unstarted. S08.5 is next;
its exact enabled-canary preparation and readiness remain required before any
activation or account operation. W27 stays deferred and incomplete.


## S08.5 read-only enabled-canary preparation

After S08.4 acceptance/local task commit `0991279`, read-only preparation under
the exclusive guard passes in 0.180 seconds. The private enabled candidate and
exact disabled rollback copy are saved outside Git. The only semantic delta is
Identity.Enabled false to true; the selected release, recovered key/config
bindings, protected account policy, required-role set, NTP and readiness pass.
No live file, account, grant, session, audit or service is mutated.

The [enabled-canary plan](../../docs/w8m-s08-canary-plan.md) records its actual
single-node production effect, one audited analyst creation, one private masked
password input, owner-only enrollment/recovery, denial and controlled expiry
checks, terminal-result/consumed-operation rules and preserved-data rollback.
The unchanged release/conformance/security evidence is reused. The exact S08.5
enabled-canary authorization remains required by this status's stop condition;
the target is not enabled merely because preparation passes. Whole-S08 review
is unstarted, S08.5 is the sole current task, and W27 remains deferred.


## S08.5 exact canary authorization

The owner explicitly approves the prepared enabled-canary plan, including its
single-node production effect, one audited analyst, owner-only enrollment,
recovery/denial checks and controlled timestamp expiry checks scoped to that
canary. This resolves the separate activation authorization gate. Recheck the
private candidate, complete recovery, service/key/schema/grant bindings under
the exclusive guard before execution. One new masked analyst-password dialog
is prepared; do not confuse it with advertiser or GPG input, launch a second
prompt or collect TOTP/recovery material. No mutation is inferred from approval.


## S08.5 enabled service; advertiser baseline pending

The first local analyst-password input fails format validation before any
production action; its stopped evidence remains unchanged. The owner explicitly
requests one fresh dialog. Input 02 passes validation and its receipt records
one new password entry without printing the credential. No third prompt or
TOTP/recovery input is requested.

Private producer preparation review corrects two P2 issues before execution:
the audit verifier uses the authoritative event_name column, and the account
mutation resolves its exact Docker/CLI child through terminal completion rather
than treating a client timeout as a resolved remote action. Earlier producer
versions and review findings are preserved. Preparation review 3 has no open
P1/P2; this does not increment or replace whole-S08 review.

The exact creation-audit SQL passes read-only. Activation preflight passes in
4.859 seconds, proving recovered originals/candidates, exact schema/image,
restricted grants/audit denials, common-key parity, unused Identity state,
NTP and readiness under the exclusive guard. The separately consumed
canary-activation-01 repeats its cheap immediate mutation gates, changes only
Identity.Enabled and restarts the same selected release. Its terminal receipt
passes in 5.462 seconds: enabled HTTP service, readyz/healthz 204, preserved
retired account protection, common-key parity and optional advertiser TOTP.
The exact disabled configuration remains the reviewed rollback; no database,
account, grant or audit row is changed by the feature window.

Fresh owner advertiser sign-in/display without mandatory TOTP is requested
before the one audited analyst creation. No analyst is created yet. The two
password-input records and feature window remain distinct, and their claims
are never replayed. S08.5 remains in progress, S08.6/W27 remain deferred, and
whole-S08 review remains unstarted. No full browser or backend suite is repeated.


## S08.5 audited analyst created; owner enrollment pending

The owner confirms fresh advertiser sign-in/dashboard/account display succeeds
without TOTP after enablement. This closes the immediate post-switch baseline.
The separately consumed canary-create-01 uses accepted input 02 through private
SSH stdin and an in-memory Docker environment descriptor; no password is in
arguments or shared output. The exact frozen published-source CLI runs under
UID 1002 with its restricted Unix-socket configuration. Its terminal result and
specific attributed AnalystCreated audit pass in 1.266 seconds. One active
protected/bcrypt analyst is present with zero permission grants. Temporary
container executable/configuration files are removed and absence is proved.
Its identifier and numeric ID remain private, outside Git and ledger evidence.

A display-only private dialog shows the owner the generated username; it asks
for no password, seed or code. The owner is asked to use an ordinary browser
for initial limited sign-in, TOTP setup/confirmation and private offline storage
of recovery codes. Enrollment, fresh TOTP login, one-use recovery, logout,
denials, controlled expiry and monitoring are not yet accepted. The expiry
producer is prepared but has not modified any session. The service stays
Identity-enabled and healthy under the approved canary window; S08.5 remains
the sole execution owner and whole-S08 review remains unstarted.


## S08.5 private username handoff correction

The owner cannot see the first display-only username dialog. Inspection proves
that child is closed; do not infer that a successful launch delivered its
contents. A fresh display-only dialog uses a retained waiting launcher and a
durable parent/child handle, with private stderr available for focused diagnosis.
The username alone is also written to an owner-only local text file outside
Git as a fallback; it contains no password. The created account and accepted
password are unchanged, no password is requested again, and no account or
TOTP operation is replayed. Owner enrollment remains the pending dependency.


## S08.5 owner TOTP enrollment verified

The owner reports setup, current-code confirmation and private offline recovery
storage all succeeded. The exact analyst aggregate observer confirms one
Enabled MFA row, no Pending row, ten unused recovery codes and one successful
TOTPEnabled audit (plus its enrollment-start audit). No seed, code, secret URI,
recovery-code hash or session cookie is read or retained in shared evidence.
The initial password-only limited session is not claimed as a fresh MFA login.

The owner is asked to use the page's POST Logout, verify a protected page asks
for sign-in, and then sign in once with a new current TOTP step. This avoids
replaying the enrollment confirmation step. Logout/revocation, fresh verified
TOTP login, recovery replay refusal, denials and controlled expiry remain open.
S08.5 is still the sole execution owner; whole-S08 review remains unstarted.


## S08.5 fresh TOTP login and logout owner result

The owner confirms the analyst Logout and fresh current-code sign-in sequence
succeeds and Account security returns Status Enabled. The next aggregate
observer checks revocation, verified active session and successful login audits;
the enrollment confirmation is kept distinct from this fresh sign-in.
The bounded same-browser denial helper is prepared and syntax-reviewed. Its
four fixed requests check missing-CSRF logout, a mismatched account parameter,
analyst campaign mutation against nonexistent id zero, and report access with
no grant. It requires expected application-error markers as well as HTTP
status, keeps cookies/CSRF inside the owner's browser, and stops on the first
unexpected result. No denial probe or session expiry is inferred from preparing
that helper. Required recovery/denial/expiry checks remain pending.


## S08.5 logout and verified TOTP session accepted

The fresh exact-analyst observer confirms one revoked session and its successful
SessionRevoked audit, one valid MFA-verified session, and two successful login
audits (initial limited enrollment sign-in plus the later fresh TOTP sign-in).
MFA remains Enabled; all ten recovery codes are unused. This supports the
owner's logout/fresh-TOTP result without reading codes or cookies.

The owner is asked to run the four fixed same-browser denial checks once. The
script only reports sanitized statuses/application-error matches, retains a
consumed client attempt marker, and stops rather than continuing or retrying
an unexpected result. Its version is frozen by the private preparation review.
No denial acceptance is inferred yet. Controlled idle/absolute expiry will use
separate fresh uniquely bound canary sessions after the current browser checks.
Its reviewed v2 producer preserves the first preparation, resolves the exact
SQL child before proceeding, bounds database lock wait, and rechecks source/
config bindings inside the exclusive guard. No session timestamp is modified
by that preparation. Required recovery/denial/expiry acceptance remains open;
whole-S08 review remains unstarted.


## S08.5 denial checks passed; controlled idle expiry applied

The owner reports all four frozen same-browser checks pass, including their
expected application response markers: missing-CSRF logout 403, mismatched
account-id read 403, analyst campaign mutation with valid own-browser CSRF and
nonexistent campaign id zero 401, and report read without a grant 403. The
exact analyst observer confirms one PermissionDenied audit; the valid verified
session remains present, and no additional session is revoked. MFA stays
Enabled and all ten recovery codes are still unused. Cookies, hidden form
tokens, page bodies and actual account identifiers never leave the owner browser
through this helper. Source group/account guards are checked against the exact
published bytes; the HTTP checks stop before any application object mutation.

The reviewed expiry producer hashes match locally and on yixin. The separately
consumed canary-expiry-idle-01 changes exactly one uniquely bound valid verified
analyst session's last_seen_at to beyond its existing idle limit. The absolute
expiry and global limits remain unchanged. Its exact SQL child terminates,
ROW_COUNT is one and the expired-idle postcondition passes. The receipt is
shortened, with HTTP rejection pending owner confirmation; it is not yet full
idle-expiry acceptance. The owner is asked to refresh once, confirm sign-in is
required, and sign in using one unused recovery code instead of TOTP. The code
remains private for the later reuse-refusal check. No expiry is automatically
retried and no customer/session/global configuration is changed.


## S08.5 recovery-code distinction and missing private list

The owner reports another successful sign-in using an authenticator code, and
clarifies no recovery-code list is available. The observer confirms three
successful logins, one currently valid verified session, one revoked session,
ten unused recovery codes and zero RecoveryCodeUsed audits. This establishes
the new sign-in as TOTP, not recovery-code acceptance. The earlier owner report
that all enrollment/storage steps succeeded remains historical; actual private
storage of the recovery list is now unconfirmed and must not be assumed.

The owner is asked to confirm that browser reload required sign-in, and to
use the existing account-security Rotate recovery codes action with a new TOTP
step, saving the displayed ten separate grouped backup codes privately. This
replaces the unavailable old list; it does not re-enroll or change the TOTP
seed. No code/list is requested by the agent. A separate aggregate-only observer
v2 includes the specific rotation audit without changing earlier evidence.
The consumed idle mutation is not replayed. Recovery-use/reuse-refusal and
absolute-expiry acceptance remain pending; S08.5 remains open.


## S08.5 recovery-code rotation verified

The owner confirms rotating and privately saving all ten recovery codes. The
fresh rotation-01 aggregate observer confirms RecoveryCodesRotated=1, ten unused
codes, zero used codes, Enabled MFA and one valid verified analyst session.
RecoveryCodeUsed remains zero; rotation/storage acceptance does not imply
recovery sign-in acceptance. Only aggregate counts are retained in the private
receipt; no code, seed, cookie or identifier is output or added here.

The next owner step is ordinary Logout and one fresh sign-in using a single
backup code from the saved list, followed by aggregate verification. The used
code remains private for the later one-use refusal check. The consumed idle
expiry is not repeated; explicit confirmation that reload required sign-in
remains outstanding. Absolute expiry, recovery use/refusal and task acceptance
remain pending. No broader test suite or new deployment is launched.


## S08.5 recovery-code sign-in verified; absolute expiry applied

The owner confirms ordinary Logout, recovery-code sign-in and Status: Enabled.
The fresh recovery-login-01 aggregate observer confirms RecoveryCodeUsed=1,
nine unused codes and one used code, a valid verified session and four total
successful logins. The second session revocation and SessionRevoked audit
corroborate the ordinary Logout. MFA remains Enabled. No code is disclosed.

The reviewed v2 expiry source hashes match locally and on yixin. The separately
consumed canary-expiry-absolute-01 changes exactly the one currently valid
verified analyst session's absolute expiry into the past; ROW_COUNT=1 and
the expired postcondition pass. No idle timestamp, global limit, other account
or configuration is changed. The HTTP expiry/refusal result remains pending: the
owner must reload without Logout, then submit the same used recovery code once
if sign-in is required. Stop on an unexpected result; no automatic retry.
The earlier controlled idle expiry still awaits explicit owner confirmation
that reload required sign-in. Broader qualification is not run.


## S08.5 used recovery-code refusal corroborated

The owner reports the expected sign-in-invalid/session-expired message after
the requested once-only reuse check. The fresh recovery-reuse-01 aggregate
observer confirms LoginFailed increases to one, while LoginSucceeded remains
four and RecoveryCodeUsed remains one. Nine recovery codes remain unused,
one remains used, and there is no valid verified analyst session. This
corroborates refusal without creating a successful session or consuming any
additional recovery code. The published second-factor code only consumes
codes whose used_at is NULL. No retry or new expiry mutation is performed.

The owner is asked to restore access with one fresh authenticator sign-in and
to explicitly confirm that both controlled-expiry reloads required sign-in.
Those HTTP confirmations remain distinct from the database timestamp receipts
and generic refusal message. S08.5 acceptance and its task review remain open
until the remaining owner results and final health/rollback checks pass.


## S08.5 authenticator access restored

The owner confirms fresh authenticator sign-in and Status: Enabled. A current
reload remains on the signed-in page, which is expected for the new valid
session and does not test the previously expired sessions. The fresh
restored-totp-01 aggregate observer confirms five successful logins, one valid
verified analyst session, Enabled MFA, one used recovery code and nine unused
codes. LoginFailed remains one and no additional recovery code is consumed.

The requested historical HTTP confirmations are narrowed to the two reloads
immediately following the separate consumed idle/absolute expiry operations.
No extra expiry mutation, browser capture or broader test is launched. Task
acceptance stays pending until those confirmations and final checks resolve.


## S08.5 owner expiry confirmations and task review 1/10

The owner explicitly confirms both earlier reloads immediately following the
controlled idle and absolute expiry required sign-in. Combined with the
separate one-row shortening receipts and subsequent aggregate audit/session
checks, both HTTP expiry results pass. Fresh TOTP sign-in restores Status:
Enabled, and a current reload retains that new valid session as expected.
The recovery-code first use and refusal are accepted separately: the same
used code creates no second successful login; nine unused codes remain.

Task review 1/10 starts across the exact activation/creation sources, owner
HTTP outcomes, audit counts, controlled session predicates, shared-key and
rollback checks. Whole-S08 review remains unstarted. Final read-only health
and rollback verification passes in 0.150 seconds: both endpoints return 204,
NTP is synchronized, source/config and running Identity/protection keys match,
adv stays optional, retired protection stays enabled, and the exact disabled
rollback plus complete independent recovery remain available. No additional
production mutation, source build or native qualification is required.


## S08.5 accepted — task review 1/10 passed

Review finds no open P1/P2 across explicit authority, exact source/config/key
bindings, one-attempt and child-outcome semantics, private credential handling,
ordinary owner browser results, independent aggregate audit/session checks,
controlled timestamp scope, source authorization guards and preserved-data
rollback. The earlier absent recovery list is resolved by recorded rotation
and private owner storage; no historical result is rewritten. All required
canary outcomes pass. Current contracts are reconciled in the same task.

Unchanged S02 security, release build and backend-conformance evidence is
reused. Only affected documentation/diff checks run at this task boundary.
S08.5 is complete; S08.6 monitoring and whole-S08 review are next. The sole
node already has the exact enabled configuration; no second deployment or
additional production node is inferred. W27 remains deferred until S08 closes.

Affected documentation guard and git diff --check pass for the S08.5 task
commit. The unrelated root README change and unpublished layout ancestor are
excluded from publication; private runtime evidence remains outside Git.


## S08.6 single-node closeout started

S08.5 is accepted in local task commit c0458e3. yixin is the owner-confirmed
sole node and already runs the accepted exact enabled source/configuration,
so the all-node step verifies that existing activation instead of creating
a second service window. No additional node, deploy, source pull or feature
mutation is necessary. S08.6 now owns read-only monitoring, readiness handoff
and whole-milestone bounded review. Current fresh canary results are reused;
only post-window indicators and affected documentation checks run fresh.


S08.6's first read-only monitor stops before any target mutation or accepted
monitoring receipt: its journal-only source assumption must be checked against
the service's actual configured destinations. Earlier audit/health results
remain unchanged. Diagnose only this reader boundary and preserve its stopped
result; no deployment or test-suite retry is warranted.


## S08.6 bounded monitoring and whole-S08 review 1/10 started

The reader correction accepts StandardError=inherit only when StandardOutput
is journal, preserving the stopped v1 result. Four focused destination cases
pass; fresh v2 reads the actual journal in 2.615 seconds. The observed window
is 2026-10-01T00:33:03.567684Z through 01:16:47.448954Z (43.7 minutes),
covering activation and owner canary work rather than introducing a new wait.

Whole-node aggregate audits show six successful logins, one expected failed
login, one PermissionDenied, one recovery-code use, one rotation, one analyst
creation, one TOTP enablement and three session revocations. There are zero
Failure audit outcomes. Across 364 service journal records there are no
observed panic/fatal, database-driver/transport, protected-request or explicit
audit-failure markers. General/slow database query logging stays disabled.
These are bounded observed indicators, not a long-term availability guarantee;
no raw messages, account fields or credential material are emitted.

Whole-S08 bounded review starts at iteration 1/10. Review covers all task
source and operational diffs, restore/recovery and one-attempt semantics,
retired-schema compatibility, permissions/audit immutability, actual node
inventory, owner HTTP results, monitoring and W27 handoff. No counter resets
or changes to earlier failed/consumed attempts are permitted.


Whole-S08 review iteration 1 records one P2 in the final private readiness
reader: it assumes both reviewed principals use localhost, whereas the
accepted runtime host selector is wildcard and maintenance is localhost.
Its read-only stop is not production drift. Preserve stopped readiness-01,
use the exact previously reviewed selector by kind and reject other selectors;
then finish the read-only grant comparison and review the full milestone again
at iteration 2. The already fixed journal/inherited-stderr reader assumption
is also reviewed; neither correction changes server permissions or configuration.

Whole-S08 review iteration 2/10 starts after the exact selector correction
and four focused positive/negative selector checks pass. The corrected
read-only schema/grant comparison runs freshly; no denied UPDATE/DELETE probes
or unrelated source/browser suite is repeated.


## S08 complete — whole-milestone review 2/10 passed

Final read-only readiness-02 passes in 2.532 seconds: the complete schema
contract still matches the independently restored rehearsal and both runtime/
maintenance principals have their exact reviewed rights, without role/proxy/
dynamic-global bypasses. No SQL mutation/denial probe is repeated. The review-1
reader finding is resolved by exact per-kind selectors and passing positive/
negative checks; the stopped reader result remains preserved.

Whole review 2 re-examines all milestone source/operational changes: completed
S07 dependency, restored migration and credential recovery, schema preservation,
audit immutability and source compatibility, kernel UID/maintenance boundaries,
exact source build and deployment history, enabled-canary one-attempt semantics,
owner TOTP/recovery/denial/expiry results, bounded monitoring, current contracts
and receiving-owner gates. No open P1/P2 remains. Source/reset security fixtures,
vet, build and conformance results remain valid on unchanged relevant bytes;
only final read-only checks and affected documentation/diff checks run fresh.

All S08 tasks are complete. The [sanitized readiness handoff](../../docs/w8m-s08-readiness.md)
supplies W27.3's independent read-only prerequisite check. W27.4 dedicated
advertiser enrollment and W27.5/W27.7 capture/count remain separately gated.
No production password change, administrator reset, extra deployment, full
native suite, evolution snapshot or unrelated publication is performed.
Retain the earlier account-data key until at least 2026-10-01T19:02:35Z.

The affected documentation guard and git diff --check pass for closure.
Remaining cross-package order is W27.3 -> W27.4 -> W27.5 -> W27.6 -> W27.7
-> W27.8, with the existing separate live gates unchanged. No evolution
direction/ownership boundary changed; no snapshot is created.
