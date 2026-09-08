# Status S07 - Account Identifier Protection

State: `[~]` In progress

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
| Plaintext retirement migration | `[~]` | Runtime reads/writes and offline verify/rotate support schemas without plaintext identifier columns behind `PlaintextRetired`. The separately reviewed one-way migration and private canary/rollback/rotation evidence remain intentionally pending; no drop is run implicitly. |
| Review and closeout | `[~]` | Four bounded repository review iterations resolved every confirmed P1/P2 and the full three-repository plus local schema/migration gates pass. Milestone closeout remains gated on the separately authorized plaintext-retirement and private activation evidence. |

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
- Full S07 acceptance remains pending on private activation evidence and the
  separately reviewed one-way plaintext-retirement migration.
