# Status S07 - Account Identifier Protection

State: `[+]` Complete; W8M protected plaintext retirement and final owner canary accepted

## Goal

Move interactive account lookup and recoverable display data away from
plaintext database authority without weakening bcrypt passwords, coupling the
account key to unrelated secrets, or forcing an irreversible rollout.

## Dependencies

- S02 identity/session/authorization and S06 public-account abuse protection.
- The shared Redis runtime and the private immutable deployment workflow.

## Tasks

| Item | State | Notes |
|---|---:|---|
| Threat, key, and compatibility contract | `[+]` | Dedicated versioned environment key ring; separately derived lookup/encryption keys; exact normalization; bcrypt-only passwords; legacy SQL retained only while default-off/rollback remains necessary. |
| Genelet protected issuer seam | `[+]` | Optional current/previous digest lookup, namespace-bound AEAD decrypt, fail-closed enabled configuration, and protected numeric-ID login-as query support preserve existing method signatures. |
| Additive schema and offline tooling | `[+]` | Nullable digest/cipher pairs plus full unique digest indexes cover adv/pub/admin/agent/analyst. Migration is preflighted and additive; `cmd/account-data` defaults read-only and supports explicit backfill/verify/rotation without identifier output. |
| Primary dual writes and lifecycle lookups | `[+]` | Public adv/pub and agent writes, protected S02 analyst creation/recovery, administrator-created publishers, authorized topics/edit/report views, and the management API use atomic identifier pairs and ciphertext display. Regression tests cover rollback-mode and plaintext-retired schemas and prevent storage fields/password hashes from entering responses. |
| Shared login throttle and impersonation | `[+]` | Redis state uses expiring pseudonymous keys, spans instances, uses S06's trusted client-IP resolver when present, reads/clears bounded rotation candidates, writes Current, clears on success, and fails closed. Legacy login-as is administrator-only, POST/CSRF-protected, numeric-ID based, has no agent entry point, and remains disabled whenever the S02 identity boundary is enabled. |
| Opaque account-action tokens | `[+]` | Advertiser/publisher activation and reset use random 32-byte, purpose/role-bound, expiring one-use tokens. MySQL stores only current-key digests; current/previous keys validate outstanding mail, authoritative updates consume atomically, app logs redact proofs, sensitive pages are no-store/no-referrer, and mail-only context is scrubbed before responses. |
| Plaintext retirement migration | `[+]` | Guarded one-time production drop passed at 2026-09-30T19:06:08Z after the owner-revised observation gate, complete-ring recovery, key rotation, fresh continuously writer-frozen backup/independent restore and full current-key/plaintext parity. All seven plaintext columns and four legacy credential procedures are absent; ten protected columns are non-null. Retained account/history fields, password hashes, all row counts and unrelated table checksums are unchanged. PlaintextRetired=true, previous key retained, restarted readiness 204. Fresh owner sign-in/display and freshly issued recovery-mail/page canaries passed. The earlier human-verification rejection and expired proof remain recorded failures; no password was changed. |
| Review and closeout | `[+]` | Complete at bounded whole-milestone review 8/10 with no open P1/P2; separate migration review remains 4. Earlier full three-repository and exact-image schema/migration checks remain valid for unchanged inputs. Production rotation, continuously frozen backup/restore, guarded retirement and fresh owner sign-in/recovery-page canaries pass. Current facts and queued S08/W27 dependencies are reconciled; older failed attempts and history remain unchanged. |

## Acceptance Criteria

- Enabled authentication never binds a plaintext login or password to MySQL;
  bcrypt remains the only password verifier.
- Database-only disclosure reveals neither a reversible account identifier nor
  a fast password verifier; application-host compromise remains explicitly out
  of scope for at-rest secrecy.
- Rotation accepts a bounded previous-key ring while all writes use Current,
  and the offline tool proves current-key parity without logging identity.
- Five-role login, account lifecycle, display, mail, and management paths work
  after plaintext removal. Legacy numeric impersonation is administrator-only
  and deliberately unavailable when the S02 identity boundary is enabled.
- Shared throttling spans instances, retains no raw login/IP, expires, clears
  on success, and fails closed when Redis is unavailable.
- No irreversible migration or production-activation claim occurs without
  separately authorized private evidence and a tested rollback window.

## Verification

- Focused Genelet account-protection/authentication, Aofei schema/management,
  and Pzdesign Redis/action-token/model tests pass during implementation.
- Implementation review iteration 1 confirmed and resolved seven blockers:
  the legacy-password upgrade no longer rebinds plaintext to MySQL; canonical
  identifiers share the 255-byte rollback-column limit; offline database
  errors cannot echo values; client storage fields cannot seed future
  authority; action-token issuance is conditioned on an unchanged identifier;
  pre-retirement verification proves plaintext/ciphertext rollback parity; and
  invalid protected identifiers remain authentication failures rather than
  dependency failures.
- Iteration 2 confirmed and resolved the continuation findings: rotation now
  rejects ciphertext/digest provenance mismatches; protected identifier
  changes update the complete tuple and revoke outstanding action proofs;
  unexpected protected-request failures expose neither driver values nor raw
  errors; agent edit works after plaintext retirement; cache discovery cannot
  create an unprotected publisher account; every online create/change path
  checks current and previous lookup digests; key promotion requires a writer
  stop; status and backfill preflight every retained source identifier and
  partial tuple before mutation; and the legacy ledger projection remains
  valid under `ONLY_FULL_GROUP_BY`.
- Iteration 3 confirmed that the Aofei-root operator examples tried to run a
  sibling main package under `GOWORK=off`; the runbook now changes into the
  Pzdesign module before invoking `cmd/account-data`.
- Iteration 4 found no remaining P1/P2 in the repository implementation. Full
  unit, vet, static-analysis, scoped race, template/copy/data, leak, schema,
  cache, and additive migration/rotation drills pass across Aofei, Pzdesign,
  and Genelet.
- The new retirement-migration review found in iteration 1 that its preflight
  did not require all five protected unique indexes. Iteration 2 tightened
  full-index shape checks and added dependency guards. Live metadata review
  found iteration 3's blanket trigger rejection also rejected the three
  existing safe account-update triggers; the guard now requires their exact
  shape and rejects any additional or identifier-referencing trigger.
  The guard and regression assertions were added; iteration 4 found no open
  P1/P2 in the migration,
  documentation, or focused checks. The whole-milestone review remains open
  until the private production gates pass.
- Full S07 acceptance remains pending on private activation evidence and the
  separately reviewed one-way plaintext-retirement migration.

## Production readiness check — 2026-09-30

A read-only check over SSH to the manifest-declared W8M host confirmed the
expected host, selected immutable release, active service, and healthy local
`/healthz` and `/readyz` responses. The selected Summer config has no
`AccountProtection` or `Identity` blocks. The database has the additive S07
columns, but several `adv`, `pub`, and `admin` rows still have a null protected
identifier pair, and the four legacy account procedures remain present. The
`adv_ip` and `pub_ip` history tables are empty. No account values or key
material were queried or recorded.

The manifest-declared account-protection environment file is readable but has
no `W8M_ACCOUNT_DATA_KEY` assignment. A metadata-only search under the runtime
root, local state and `/var/backups` found no MySQL/database backup or restore
proof filenames, and no matching user timer. This does not inspect an external
backup provider, whose evidence remains unverified.

The retirement SQL's metadata preflight was evaluated read-only against the
live schema: source shape, old indexes, protected unique indexes, expected
procedures, exact known account-trigger shape, absence of dependent views, and
absence of `passwd_hmac` all pass; the complete-protected-pair condition fails.
Host inspection found no MySQL backup/restore timer or dedicated local backup
utility; the installed `mysqldump` alone does not prove an encrypted physical
backup or successful restore.

This is not acceptance evidence for plaintext retirement. The separate
one-way migration's focused Go schema assertions and synthetic MySQL 9.0.1
positive/fail-closed runs pass; focused migration review iteration 4 found no
P1/P2. The pinned MySQL 8.0.41 drill remains pending. The first disposable-fixture initialization stopped
before applying the migration because its database was not ready; that
container was removed and the next attempts used fresh containers. The
production-pinned image is not available locally, and no dependency was
fetched. The required private evidence for
verified backup/restore, backfill parity,
dual-read canary, rollback window, proxy-log handling, outstanding-link
drain, and key rotation has not been verified. No production data, secret,
configuration, or service was changed. Keep S07 blocked until those exact
prerequisites and the reviewed migration are available; S08 and W27 remain
queued behind S07.

### Production prerequisite recheck — 2026-09-30

A second read-only SSH check confirmed the same immutable release and active
service, with `/healthz` and `/readyz` both returning 204. The production
container is the manifest-pinned MySQL 8.0.41 image. `AccountProtection` and
`Identity` remain absent from the selected Summer config, and the dedicated
environment file still has no account-data key assignment. Aggregate database
counts show incomplete protected pairs for all 5 advertiser, 2 publisher, and
1 administrator rows; agent and analyst tables have no rows. The legacy
`proc_adv` and `proc_pub` procedures remain. The account-history tables contain
zero rows. No account values or key material were queried.

The active `www.w8m.com` TLS virtual host sends requests to
`w8m-access.log` using Apache's `combined` format, whose `%r` field records the
full request line and target. Account-action proofs are carried in URLs, so
this configuration can place such proofs in proxy access logs. No logs were
read. Token redaction or access-log exclusion is not configured in the
inspected virtual-host directives; this prerequisite is therefore **failed**
until the infrastructure owner implements and verifies a safe format or
exclusion.

The service was active during this check, so there was no account-writer stop.
No private frozen-backup or successful-restore evidence was found in the
inspected host paths or infrastructure repository; an external backup
provider remains unverified. No backfill, canary, key rotation, database,
configuration, or service mutation was performed. S07 remains blocked; S08 and
W27 remain queued.

### Authorized S07 operational sequence — 2026-09-30

The owner authorized logging remediation, encrypted backup/restore setup,
private account-key provisioning, protected backfill/canary, and final
plaintext retirement, including human interactions. Each dependency and stop
condition above still applies. No existing backup process is available. The
owner selected encrypted off-Git snapshots on the local workstation, separate
from `yixin`, with a new passphrase-protected GPG recovery key and separately
retained owner recovery material. Recurring/provider backup coverage is not
claimed by this one-time S07 recovery arrangement.

Apache access logging is remediated on both HTTP and HTTPS virtual hosts using
infrastructure's shared metadata-only snippet. Offline host-selection tests,
one isolated Apache fixture, and the full candidate configuration pass. The
first activation rolled back successfully because its probe incorrectly
expected HTTP 301; the existing redirect returns 302. A fresh checkpoint and
corrected probe passed with HTTPS 200/HTTP 302, dummy URL/header markers absent
from all inspected new access-log suffixes, and origin health/readiness 204.
The installed snippet SHA-256 is
`8e70978aac6729d8e6cab698213c01c06dc33b3a923b5de86ce4edf59ddbe1df`.
Both activation outcomes are retained in infrastructure history. Existing
logs were not rewritten or emitted. The current repository-wide infrastructure
conformance gate fails on a pre-existing preserved browser-history backup ref;
it was not weakened. The focused operational acceptance is separate from that
unresolved repository gate and from complete S07 acceptance.

The new workstation GPG key's private files are passphrase-protected. The
writer-stopped baseline snapshot is encrypted on the server before transfer
to owner-only workstation storage outside Git. Its isolated, network-disabled
MySQL 8.0.41 restore matches all 96 table counts/checksums, the schema digest,
6 routines, 65 triggers, and the accounting contract. Binlog coordinates were
stable during the freeze, and previously active writers resumed with readiness
204. The backup receipt SHA-256 is
`e435cdf643c3cb4828207422b04992460d1009a909851901f763a91e25bce734`.
Recovery used a separate keyring restored from the protected-keyring archive;
the owner confirmed that the passphrase is saved separately and that the
recovery archive is copied to private storage. Recurring backups remain an
unaccepted operational gap, separate from this one-time recovery proof.

A dedicated 32-byte account-data key is privately provisioned, and the four
existing production issuer contracts have their protected fields installed
from the exact deployed template. At this provisioning checkpoint the file
configuration remained default-off and the running service had not adopted
protection. The encrypted key/config
archive's independent recovery check passes. Its receipt SHA-256 is
`7bdbeba2e1d629dfb7dfcdfceeaf3d6f0be8be4395d132606e2795ce5f08db14`.
The first noninteractive recovery check stopped because the key was locked;
that diagnostic is retained and the fresh masked-unlock check passed. The
read-only account-data status validates all retained source identifiers and
finds no partial pairs. No backfill or plaintext drop had run at that checkpoint.
The subsequent restored-copy drill and production activation follow.

### Protected production canary and remaining stop gates

The restored-copy MySQL 8.0.41 drill rejects missing pairs before any durable
mutation and preserves the complete source inventory. Backfill, current-key
and plaintext verification, rotation with the former key retained, retirement,
and post-retirement cryptographic verification pass. The SQL removes all seven
targeted plaintext columns and four old procedures and makes ten protected
columns non-null. The receipt SHA-256 is
`975d672e3bf474a1b39c32ba7cf9e8c484af0c96beab63ccdc6174fe69cfbf1f`.
Its first attempt stopped before backfill because the private tmpfs was
non-executable; that failure is preserved. The fresh attempt kept private
configuration/keys on tmpfs and only the tool binary on an executable path.
The restored container and its private tmpfs were removed, with joined cleanup
confirmed. This is a restored-copy drill, not a production drop or rotation.

Production activation's first preflight stopped before writer/service/database
changes because its temporary script assumed the wrong owner UID. The corrected
fresh checkpoint uses the SSH user's actual UID. Under the exclusive runtime
lock, every writer was stopped and the database inventory still exactly matched
the restore-tested encrypted baseline. Backfill protected all 5 advertiser,
2 publisher and 1 administrator rows; agent and analyst remain empty. Full
verification covers every row in all five tables and proves current-key digest,
ciphertext and retained plaintext parity. The same immutable binary release
restarted successfully with protection enabled and origin readiness 204 at
2026-09-30T17:31:47Z. The owner confirmed a fresh ordinary-browser advertiser
sign-in, dashboard and account display succeeded. No automated browser count,
profile capture, account enrollment or registration was performed.

A focused tool built against the selected production source exercised the
actual shared Redis throttle using one synthetic, non-account attempt: the
fifth failure returns 429, keys are pseudonymous with bounded TTL, a separate
process sees the limit, success clears it, and a closed Redis client denies
access. No real account was throttled. This verifies the production Redis seam;
it does not claim an end-to-end HTTP outage test or five-role live sign-ins.

The service returned to default-off mode and restarted healthy, then full
cryptographic/plaintext parity passed before it restarted protected and healthy
again. Fresh process identities and readiness 204 were verified in both modes.
Default-off login was not independently exercised in this rollback check. The
rollback/restart receipt SHA-256 is
`ea4692c355a1eb219351a0c3e344b9fa2aac8c6975bb371f54b8744e21fa4231`.

The owner cannot confirm whether old activation/reset emails remain and chose
a 24-hour observation period. Its minimum endpoint is
**2026-10-01T17:37:57Z**, measured from the final protected resume after the
rollback check. This is not proof that legacy links expired: timestamp checks
on the legacy reset path depend on Identity, which remains disabled, and legacy
activation does not establish a uniform bounded expiry. Protected mode rejects
old identifier-bearing proofs immediately. Before irreversible retirement,
review protected recovery/mail continuity and the remaining outstanding-link
or reissuance disposition; elapsed time alone cannot satisfy this gate.

Production rotation is queued after the observation/canary gate. Retain the
former key for at least the longest protected action lifetime after its final
write, verify all rows under the promoted Current key, and escrow the complete
ring. Then require a fresh writer-frozen encrypted snapshot and successful
restore, final full parity verification, and the reviewed one-way SQL exactly
once. Keep `PlaintextRetired=false` until that authorized guarded operation;
set it true before resuming writers afterward. S07 remains incomplete and S08
and W27 remain queued. Whole-milestone review remains open at its existing
iteration counter; no accepted retirement or closeout is claimed.

Focused account-identifier schema assertions and `go vet ./etc` pass, and
diff whitespace/link checks pass. At this checkpoint the documentation guard
still referenced pre-migration paths and infrastructure's reachable-history
backup reference still blocked conformance. The following authorized
continuation resolves those check failures without rewriting their history.

### Authorized continuation — recovery continuity and closeout-check repair

The owner authorized all six remaining cross-package steps and human
interactions. The order remains S07 -> S08 -> W27 with task-level commits and
the existing scoped Git/server authority. The observation minimum is unchanged:
2026-10-01T17:37:57Z. This approval does not waive the remaining acceptance,
readiness, writer-stop, backup/restore or one-attempt conditions.

The owner requested one fresh recovery email through the ordinary browser and
confirmed that it arrived and its recovery page opened, without changing the
password. This establishes recovery-email/page continuity for the dedicated
advertiser; it does not claim token consumption, publisher recovery or expiry
of unknown old mail. Legacy links are already rejected in protected mode;
the outstanding-link disposition still needs its final review before the
irreversible drop.

The documentation guard now reads `tabilet/` paths and maintained links resolve
from their new locations. Frozen history/evolution documents retain their bytes
and their original source-path link context. Untracked Markdown is checked as
well. The guard passes. A fresh isolated fixture with its required sibling
documentation proves that broken current, frozen-history and untracked links
each fail; the first fixture's missing-sibling setup failure is preserved.
Frozen history, evolution and GOAL.md bytes remain unchanged. Documentation
maintenance review 1/10 has no open P1/P2; it does not close the S07 milestone
review or reset its counter.

Infrastructure's old backup reference was preserved in an owner-only Git
bundle outside the deployment repository, independently restored and checked
with full strict fsck. Its exact tip and historical blob identities match.
Only the local backup reference was removed; no commits were rewritten. The
bundle SHA-256 is
`c3ee5c303626acb855497579f2b9fa08c3ba94fb90330adc4912f9d12aaff263`.
The boundary now recognizes legacy and tabilet browser-status paths, including
retired histories. Focused fixtures prove rejection in the current tree and
reachable history for all four layouts. The actual repository boundary passes.

The pinned Go 1.23.5 SDK was explicitly prepared separately. A supplied offline
source/SDK kit on yixin runs the complete backend conformance gate with fetching
disabled: policy/history, syntax/JSON, deployment unit fixtures, manifest and
selected-release verification, and the private maintenance fixtures all pass.
The existing release and service remain selected and healthy; this check
activates no backend release or database feature. Infrastructure maintenance
review 1/10 passed for the boundary code. Evidence review 1/10 found a P2:
the receipt initially used a later Aofei worktree HEAD rather than the supplied
archive's actual source commit. The archive identity is now read from its Git
tar header; evidence review 2/10 has no open P1/P2. The earlier check failures
are resolved, while S07 remains incomplete on the observation/link disposition,
production rotation and final retirement gates. S08 and W27 remain queued.

The accepted documentation-maintenance change is committed locally as
`2f360a6`, separately from the pending retirement implementation and S08
planning edits. Its exact staged tree passes the documentation guard. The
pre-existing unpublished layout-migration ancestor remains local; it was not
implicitly published with this maintenance acceptance.

### Documentation-check maintenance — 2026-09-30

The owner authorized repair of the closeout checks under the ongoing S07 ->
S08 -> W27 loop. The documentation guard reads the current tabilet layout,
validates maintained incoming links and includes untracked Markdown. Frozen
history/evolution links retain their original source-path context and their
evidence bytes remain unchanged, as does GOAL.md. The guard passes; fresh
isolated fixtures prove broken current, frozen-history and untracked links
fail. The first fixture's missing sibling-doc setup failure is retained.
Maintenance review 1/10 has no open P1/P2. This is a completed maintenance unit,
not S07 retirement or milestone-review acceptance. Production protection is
now enabled and the owner-confirmed advertiser canary/recovery page pass; the
remaining retirement gates still block S07, S08 and W27. The observation
minimum remains 2026-10-01T17:37:57Z.


### Observation-window continuation preparation — 2026-09-30

The owner explicitly retained the 24-hour observation and requested preparation
now. The minimum remains 2026-10-01T17:37:57Z. It is an observation policy,
not proof of legacy-mail expiry; the final link-continuity/disposition review
remains required. The existing cross-package action authority remains valid.

Prepared a disabled private continuation packet at
`/home/peter/.local/share/w8m-s07-backups/kit/continuation-preparation-01`.
It binds the exact tool, reviewed SQL, successful restored-copy drill and
backup/key/rollback references. A fresh read-only yixin inspection confirms the
same immutable release and MySQL image, active service, health/readiness 204,
protection enabled, plaintext retained, and the original Current/no Previous.
The remote SQL and binary hashes match the prepared and drilled bytes. No
writer, key, configuration, database or service was changed.
The preparation receipt SHA-256 is
`f05da36cf8b2e645dc42c01af7fba55be5f8f544e9da0c0d5ae88672c9150e04`.

The packet specifies complete-ring escrow before promotion, consistent Current
promotion under writer stop, fresh rotation/full verification, and retention
of the old key for at least the longest protected-action lifetime after its
final write. That key-retention period is separate from plaintext retirement.
It identifies the historical v1-only helper and consumed output paths as
ineligible for the new operation. The final fresh snapshot/independent restore,
unchanged inventory/binlog and full parity precede the exact one-time drop with
writers continuously stopped; partial DDL stops without automatic retry or
incompatible writer resume. Protected retired configuration precedes restart.

Preparation review 1/10 has no open P1/P2 within this disabled-packet scope;
it does not reset the existing migration or whole-milestone review counters.
The focused checks are exact byte comparison, the read-only host preflight and
documentation/diff validation. No browser or full qualification was run. This
is preparation evidence only: production rotation, final backup/restore,
plaintext retirement and S07 acceptance remain pending. S08 and W27 remain
queued behind S07.


### Owner revision of observation endpoint — 2026-09-30

The owner explicitly changed the endpoint to September 30, 12:00 noon in
America/Los_Angeles. Verified the zone conversion: PDT (UTC-07:00), hence
**2026-09-30T19:00:00Z**. This supersedes the previously selected 24-hour minimum
and its reaffirmation in the preceding chronological notes. The original
rollback receipt and preparation packet remain unchanged historical evidence.
The current task row and W27 dependency summary use the revised endpoint.

The private `continuation-preparation-02` packet records this policy amendment,
references the exact unchanged predecessor receipt/assets and updates the
procedure. The endpoint is a minimum, not automatic drop authority or proof
that unknown old mail expired. Observation/canary and link-disposition review,
rotation, complete-ring recovery, fresh frozen backup/restore, full parity,
once-only guarded SQL, compatible restart and milestone review remain required.
The previous key's protected-action retention is not shortened. No production
operation occurred as part of this planning amendment; S07 remains incomplete,
and S08/W27 remain queued. Amendment review 1/10 has no open P1/P2; existing
milestone review counters remain unchanged.


### Active continuation at the revised endpoint

The owner confirmed that the goal should continue automatically in the active
turn after September 30 noon Los Angeles time. The agent is keeping this run
active until 2026-09-30T19:00:00Z, then checking the remaining S07 prerequisites
under the existing authorization. No scheduled background run was created.

Prepared fresh private rotation, complete-ring recovery and final frozen-window
launchers in `continuation-preparation-02`. Syntax and the real inherited-lock
guard pass; a focused pre-endpoint invocation is rejected before a rotation
record or key is created. This is preserved as a stopped preflight, not a
consumed production rotation. Prepared code uses fresh exclusive operation
paths, exact immutable target/tool bindings, independent ring recovery before
promotion, and a writer freeze maintained across snapshot/restore/parity/drop.
The retired-schema check also requires unchanged retained account/history
fields and password hashes, all table row counts and unaffected checksums.
The isolated exact-image final restore verifier uses the already unlocked
protected GPG session and normal fixture stop/removal. Preparation reviews
1/10 have no open P1/P2 within their respective helper scopes and do not reset
S07's existing milestone-review counter. No browser suite is involved.


### Production rotation and one-time retirement — 2026-09-30

The active turn continued after the owner-revised noon PDT endpoint. The fresh
observation review found readiness 204, an active service and zero priority-error
journal entries since the protected resume. The owner-confirmed sign-in/display
and new recovery email/page evidence pass. Protected mode continues rejecting
old identifier-bearing links; fresh protected requests are the supported
recovery/activation path. No legacy-mail expiry is claimed or bulk mail sent.

The complete old/future key ring and both configurations were encrypted on the
server, transferred to the protected workstation store and independently
recovered using one masked GPG unlock. No W8M password was requested. Under the
exclusive runtime-root lock and account-writer stop, the new key was promoted
and all five tables rotated and verified completely (5/2/1/0/0 role totals).
Rotation completed at 2026-09-30T19:02:35Z; readiness returned 204. The former
key stays in Previous until at least 2026-10-01T19:02:35Z, independently of the
shortened observation window. Private rotation receipt SHA-256:
`f6678c46b73a39a3bdcffcc85d9260af85919f0b7483f5c148b4378bee130946`.

A second exclusive, continuously held writer stop produced a fresh encrypted
database and complete runtime/key-ring snapshot. The independent restore used
the exact supplied MySQL 8.0.41 image, no network and ephemeral database storage;
source/restored schema, every table count/checksum and object inventory match.
The complete ring/configuration recovery and normal fixture stop/removal pass.
The protected GPG unlock session was closed after recovery checks. Writers
stayed stopped throughout snapshot, independent restore, unchanged binlog and
inventory checks, full cryptographic/plaintext parity and the exact once-only
SQL. The drop was not retried.

Production retirement passed at 2026-09-30T19:06:08Z. All seven plaintext columns
and four legacy credential procedures are absent; ten protected columns are
non-null. Complete current-key cryptographic verification passes. All retained
account/history field projections and password hashes, all table row counts,
and unrelated table checksums are unchanged. PlaintextRetired was set true
before the previously active compatible services/timers resumed; readiness is
204 and the previous key remains configured. The immutable release is unchanged.
Private retirement receipt SHA-256:
`2a8686dbcc131fd2be0786b896db0fa532ab93413fc5b639fbd18ecaf4f4731e`.
Encrypted recovery artifacts and sanitized producer evidence remain outside Git
under `snapshots/rotation-01` and `snapshots/final-retirement-01` in the protected
S07 backup store. This is a one-time verified recovery snapshot, not recurring
provider backup or O02 availability/RPO/RTO acceptance.

The final fresh ordinary-browser owner sign-in/display and reopened existing
recovery page are requested without changing the password. S07 remains
incomplete pending that canary, bounded milestone review and task acceptance/
commit. S08 and W27 remain queued; Identity stays disabled and no TOTP enrollment,
profile capture or live-count claim was armed. Earlier failed preflights,
original observation receipts and review counters remain unchanged.


### Closeout evidence review — iteration 5/10

Reviewed the complete S07 boundary against the accepted repository source,
retirement SQL and exact deployed immutable release. The prior source/unit,
vet/static/race and five-role lifecycle checks remain valid for unchanged
application bytes; the new one-way SQL is byte-identical to the successful
exact-image restored-copy drill and the once-only production operation.
The promoted ring, complete independent key/config recovery, continuously
frozen final snapshot, source/restored object and all-table count/checksum
identity, stable pre-drop binlog, full current-key/plaintext parity, exact
schema postconditions, retained fields/passwords and unrelated-data preservation
are supported by the producer receipts. Source/config secrets and account data
remain outside Git; the temporary GPG unlock session is closed.

No open P1/P2 in implementation or the reviewed operational evidence. Final
owner canary is a required pending acceptance input and is not inferred from
readiness, process exit or the earlier pre-drop sign-in. Whole-milestone review
has advanced from 4 to 5; the separate migration review remains 4 and neither
counter is reset. Documentation guard, JSON and diff checks pass. Earlier
history, failed/pre-endpoint preflights, frozen evidence, GOAL.md and evolution
remain unchanged. No new evolution version is required: production activation
implements the existing direction and ownership contracts.

The sanitized infrastructure records are committed as `ac18744` (rotation)
and `6cc1339` (retirement) and pushed. Their receipt digests, completion times,
readiness and plaintext-state fields match their private producers; they record
the then-pending final canary honestly and contain no raw database output,
configuration or account values. The unrelated infrastructure layout migration
is preserved. S07 is not yet complete or committed as accepted. S08/W27 remain
queued until the final owner canary and closeout acceptance pass.


### Post-retirement owner checks and focused recovery diagnosis

The owner independently reports that a new advertiser registration reached the
verification-email page, mail arrived, activation succeeded and the new account
signed in. This is manual application evidence, not a W8M registration adapter
attempt, and does not replace the retained dedicated account or reopen W8M's
dormant registration scope. The new account was created after the retirement
receipt; its later existence does not rewrite the earlier preserved role totals.

The owner reopened the roughly hour-old recovery mail and saw "We Could Not
Complete This Action", "Review the guidance or return home and choose the
correct account entry", and "The Submitted Information Could Not Be Verified".
This is preserved as a rejected page, not a passing recovery canary. Focused
read-only diagnosis at 2026-09-30T19:15:56Z found one stored advertiser reset
proof, expired, and zero unexpired proofs. The previous account-data key is
still configured. The deployed source explicitly sets accountResetTTL to one
hour and requires reset_token_expires >= the current UTC time. The observed
rejection is consistent with expiry; no code change or broader test suite is
justified by this evidence. No token or account identifier was read or printed.

Requested one fresh retained-dedicated-account sign-in/dashboard/display and
one fresh recovery email/page opened promptly without changing the password.
S07 closeout remains pending on that specific acceptance input; the source and
evidence review stays at 5/10, and S08/W27 remain queued. No password, key,
account, browser profile or claim was changed by the diagnostic check.


### Closeout continuation review — iteration 6/10

Resumed under the owner's instruction to proceed; that instruction grants
continuation but is not a successful fresh-canary result. Reviewed the current
retirement implementation/evidence and maintained closeout diff. Confirmed P2:
although the S07 specification body and current facts record successful
retirement, the roadmap header/index still describe production as blocked and
disabled. They must reflect the accepted operation and distinguish default-off
templates from the enabled, retired W8M runtime. Persisted this finding before
the documentation fix; final owner canary remains pending. Source and historical
operation bytes are unchanged. The next bounded pass will review the correction
and updated evidence without resetting the existing counter.


### Closeout correction review — iteration 7/10

Reviewed the full unchanged S07 implementation and accepted operational evidence
alongside the current maintenance diff and iteration 6's roadmap correction.
The roadmap header/index now distinguish default-off templates from the enabled,
plaintext-retired production runtime, consistently with its specification,
product/architecture/tooling facts and current task row. The P2 is resolved;
no open P1/P2 remain in implementation or recorded evidence. Documentation guard
and diff checks pass. No browser qualification or repeated application suite is
needed for this documentation correction.

Prepared an isolated-index commit preview outside Git, including only S07's
retirement/closeout files and the approved queued S08 dependency specification.
The unrelated README label edit and infrastructure layout migration are excluded;
the ordinary index is unchanged. The preview records canary_pending=true and
acceptance_commit_ready=false: it is not an accepted task commit. No deployment,
configuration, database, account or private browser state changed in this turn.
The fresh dedicated-account sign-in/display and newly requested recovery-page
result remain required before acceptance. The owner was asked for those results,
not new authorization; S08/W27 remain queued. The next review continuation must
retain the whole-milestone counter at 7/10 rather than start again.


### Fresh owner canary and exact recovery-submission diagnosis

The owner confirms fresh dedicated-account sign-in/dashboard/account display
succeeded. The newly requested recovery-mail submission failed after clicking
Send Password Reset Email, showing “Some Information Is Missing or Invalid”.
This is a failed submission, distinct from the earlier expired recovery link;
recovery acceptance remains pending.

Read-only sanitized journal inspection correlates the retrieve action and
the reviewed human-verification error at 2026-09-30T19:27:42.194190Z. The
server's exact approved message is “请完成人机验证后再提交。” (complete human
verification before submitting). Its source branch returns HTTP 400 when the
Turnstile response is empty or exceeds 2048 bytes, before Siteverify, database
lookup, token issuance or email sending. No request body, identifier, token or
raw log was exposed. The journal message alone does not distinguish those two
input conditions or explain why the browser did not supply an acceptable
response. The form currently permits submission before the asynchronous
widget finishes. No retirement-schema defect is established by this failure.

Requested one fresh ordinary-browser recovery flow after human verification
finishes, opening the new mail promptly without changing the password; if
the widget is absent or errors, stop and report that instead. No automated
retry, production change, browser qualification or wider suite was started.
The whole-milestone review remains at 7/10 pending this acceptance input.
S08 and W27 remain queued.


### Fresh recovery-page canary passed

The owner reports Password Reset Email Sent, receipt of the fresh mail and
opening its link to Set New Password with the 12-character password guidance.
The owner separately confirms the requested fresh flow worked. Combined with
the fresh sign-in/dashboard/display success, the final dedicated-account
canary passes. No password change or TOTP enrollment was requested or performed.
The earlier expired-link and missing/oversized Turnstile-response submission
failures remain intact; neither is rewritten as successful. This new ordinary-
browser result establishes recovery-mail/page continuity after retirement.
The encrypted recovery snapshots, old-key retention minimum and rollback
requirements remain unchanged.


### Closeout acceptance review — iteration 8/10

Reviewed the full S07 implementation and diff using the unchanged source
reviewed at iteration 7, the separately reviewed guarded SQL, exact producer
receipts and completed owner canaries. Rotation and retirement receipt hashes
match their committed sanitized infrastructure histories. Complete-ring
recovery, fresh continuously frozen backup/independent exact-image restore,
full parity, retained data/password and unrelated checksum preservation, exact
postconditions and compatible restart remain supported. A fresh read-only
check confirms the unchanged release, protection and PlaintextRetired enabled,
one retained previous key, Identity disabled and readiness 204. The earlier
incorrect config-path inspection failed read-only; the corrected inspection
used the existing runtime-root configuration without exposing values.

The existing focused missing/mis-scoped human-verification test passes
(0.019 seconds test execution after dependency compilation); no application
bytes were changed and no wider browser suite was run. The owner's fresh
sign-in/display and newly issued recovery email/page pass. Earlier failed
submissions, consumed paths, original observation receipts and producer history
remain unchanged. No open P1/P2. Whole-milestone review advances from 7 to 8;
the separate migration review remains 4. Required earlier source/unit, vet,
static/race, five-role lifecycle, schema and migration gates remain valid for
their unchanged inputs. Documentation/diff checks pass. GOAL.md, frozen history
and evolution bytes remain unchanged; activation follows the existing direction
and creates no evolution snapshot. Retirement task acceptance passes.

S08 is the next owner after closeout commit, with the retired schema and
proven encrypted recovery baseline as prerequisites. The former key remains
required until at least 2026-10-01T19:02:35Z; key removal and recurring backups
are not implicitly performed by closure. W27 remains gated on S08's separately
verified Identity rollout, enrollment and exact live authorizations.


### S07 closed and downstream reconciliation

Retirement task acceptance is committed locally as `fbfd144`. Final owner
canary evidence is appended to infrastructure history as `4af90a8`; original
rotation/retirement records remain unchanged. Whole-milestone review 8/10
passes with no open P1/P2. All tasks and acceptance gates are complete; current
product, architecture, roadmap, tooling and operator index now reflect that
result. Earlier pending statements above remain chronological evidence.

The remaining cross-package order is S08 -> W27. S08 starts from the unchanged
identity-disabled release on the protected retired schema; the independently
restored encrypted snapshot and complete ring remain the recovery baseline.
S08's permission/key/canary/rollout requirements remain mandatory. W27.3 is
still blocked until S08 rollout and W8M's own read-only acceptance check.
No Identity activation, TOTP enrollment, authoring capture or count claim was
performed by S07 closeout. The unrelated README label edit and infrastructure
layout migration remain outside the task commits. Aofei's existing unpublished
layout-migration ancestor is preserved locally, so these task commits are not
pushed through that unrelated ancestor.
