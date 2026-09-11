# Account Identifier Protection

S07 is a default-off, staged migration that prevents interactive account
identifiers from being storage/search authority in plaintext. It covers the
`adv`, `pub`, `admin`, `agent`, and `analyst` roles. Passwords remain bcrypt
credentials verified in Go; there is deliberately no deterministic password
digest.

## Threat Boundary

Identifier HMACs prevent a database-only reader from enumerating normalized
emails or logins without the deployment key. AES-256-GCM ciphertext permits
authorized application display and mail while detecting modification. This
does not protect identifiers from a compromised application host that can read
the key or from an authorized response that intentionally displays an account.

The account-data key is independent from Summer's signing `Secret`, the S02
identity key, provider credentials, and quota keys. A 32-byte value is supplied
only through an environment variable named by the owner-readable Summer
configuration. The repository and private infrastructure manifest may name
that variable but never contain its value.

Genelet derives separate lookup and encryption subkeys. Lookup digests are
HMAC-SHA256 over a versioned, role/field-separated namespace. Ciphertexts use a
random nonce, namespace-bound associated data, and the envelope
`ap1.<key-id>.<base64url>`. Email normalization is exactly lowercase plus trim
followed by strict address parsing; login normalization is lowercase plus trim
with control-character rejection. Both forms are limited to 255 bytes so the
protected value and its rollback `VARCHAR(255)` projection cannot diverge. The
same Genelet functions serve online lookup, writes, backfill, verification,
and rotation.

## Password Contract

The raw password already stays in the Go process on the live bcrypt path.
MySQL returns the stored bcrypt value and Genelet performs the password check.
S07 does not add `passwd_hmac`: such a value would be a fast deterministic
password verifier and would make offline guessing much cheaper after a
database-and-key disclosure. Every password write continues to store only a
bcrypt value.

The bounded legacy-password upgrade locks and reads the row by numeric account
ID, compares the old value in Go, and writes only the new bcrypt hash by ID.
It never sends the legacy plaintext back to MySQL as a comparison bind.

The legacy `proc_adv` and `proc_pub` routines are retired in the clean schema;
calling them fails instead of comparing plaintext passwords. Normal and
protected issuer SQL remain separate during rollout. Enabling protection makes
Genelet send only identifier digests to the protected query and try the current
key before bounded previous-key candidates. The returned cipher attribute is
authenticated and decrypted before session attributes are built.

## Rotation Contract

`AccountProtection.Current` names one write key and `Previous` names at most
three read keys. New digests and ciphertext always use Current. Login can find
rows written with any configured key, and ciphertext carries its key ID. Run
the offline rotation mode to rewrite rows to Current, verify zero previous-key
rows, then remove an old key. Never remove the only key able to decrypt a row.
Keep a previous key available for at least the longest outstanding account
action lifetime after the final write with that key: 24 hours for activation
and one hour for password reset.

Key promotion requires an account-writer stop: first distribute the future key
as Previous everywhere, stop registration and identifier/account-action
writes, make every writer use the same ordered ring and Current, then resume
writes before the row rotation. Lookup preflights reject an identifier still
stored under any configured previous key, while the Current unique index
arbitrates concurrent writers. The writer stop is still necessary during the
Current flip because two instances writing different Current keys cannot be
made mutually unique by separate database indexes.

An emergency rotation of Summer's ordinary signing secret does not affect
account lookup. Loss of every account-data key makes protected identifiers
unrecoverable and is restored through the encrypted infrastructure secret
backup process, not guessed from the database.

## Repository Rollout

The checked-in Summer example and clean schema are default-off and rollback
compatible:

1. Apply `etc/s07_account_identifier_migration.sql` only during an authorized
   writer stop with a verified frozen backup. It adds nullable digest/cipher
   columns, activation/reset token-digest and expiry columns for advertiser and
   publisher accounts, and full unique digest indexes. It never drops
   plaintext or adds a password digest.
2. Provision an independently generated 32-byte account-data key in the
   owner-only environment and add its key ID/environment-variable name to the
   owner-readable Summer config. Do not enable protection yet.
3. Use Pzdesign's `cmd/account-data` with `status`, then explicitly
   `backfill -write`, then `verify`. The command reports only table/count/id
   diagnostics and never prints identifiers, ciphertext, digests, keys, or
   database error values. Before plaintext retirement, verification also
   requires the retained plaintext projection to normalize to the decrypted
   ciphertext, preserving an honest rollback path.
4. Deploy dual-read/write-capable binaries. Enable AccountProtection only
   after every interactive account row verifies and all service instances have
   the same ordered key ring. Enabled startup rejects missing keys or incomplete
   protected issuer contracts.
5. Enabled login uses Redis-backed shared throttling keyed only by a protected
   login-attempt digest. Redis failure fails closed with `503`; five failures in
   five minutes return `429`; success clears the shared state. During key
   rotation, reads and cleanup include bounded previous-key candidates while
   failures write only the current-key candidate. On W8M, the client address
   comes from S06's trusted-proxy chain resolver; forwarding headers from an
   untrusted peer are ignored.
6. Retain the plaintext columns and legacy SQL through the canary and rollback
   window. Repository implementation does not claim production migration or
   activation.

Publisher cache discovery no longer creates an account for an unknown domain:
it has neither an interactive identifier nor access to the account-data key.
Provision the publisher through the authorized Summer/admin account workflow
before discovery adds sites or slots. The local bootstrap-only `etc pub`
command remains a pre-migration convenience and must not run after an S07
backfill without another verification pass.

Example read-only and write commands from the Aofei repository root:

```bash
(
  cd ../pzdesign
  GOWORK=off SUMMER=/owner/readable/summer.json \
    go run ./cmd/account-data -mode=status
  GOWORK=off SUMMER=/owner/readable/summer.json \
    go run ./cmd/account-data -mode=backfill -write -limit=1000
  GOWORK=off SUMMER=/owner/readable/summer.json \
    go run ./cmd/account-data -mode=verify -limit=100000
)
```

`backfill` refuses partially populated digest/cipher pairs. `verify` decrypts
each checked ciphertext, requires its digest to match Current, and, while the
plaintext column exists, proves plaintext/ciphertext parity. `rotate -write`
authenticates the ciphertext, requires its existing digest to match the same
identifier under one configured key, and rewrites both fields under Current in
one transaction.

Read-only `status` validates every retained source identifier. Before
`backfill` writes any table, the command repeats that validation and checks all
five tables for partial digest/cipher pairs. Historical publisher rows that use
a domain or another synthetic value as `pub.email` must be assigned an
operator-approved account email (or have their inventory moved to a real
account) first; the tool reports the numeric row ID and refuses to guess or
print the invalid value.

## Account-Action Proofs

When protection is enabled, advertiser and publisher activation and password
recovery links contain only a numeric account ID and a random 32-byte
base64url token. The database stores a role/purpose-bound keyed digest, never
the raw token. Issuing a token replaces the previous proof for that account
and purpose only if the account's current identifier digest still matches the
address selected for delivery. A concurrent identifier change therefore
prevents a proof from being sent to a stale address. Activation expires after
24 hours; reset expires after one hour. A successful protected email change
writes the new identifier tuple and clears both kinds of outstanding proof in
the same account update, so the former mailbox cannot retain reset authority.
The state change, token consume, and (for recovery) bcrypt password write are
one atomic update. Password recovery also invalidates any pending activation
proof. When the S02 identity service is enabled, recovery-code consumption,
session revocation, and the security audit join that same transaction. Current
and previous account-data keys accept mail already in flight.

Raw proofs exist only in the mail-rendering context. Genelet removes delivery
blocks before rendering an HTTP/JSON response, redacts action tokens and
password/MFA fields from application query logs, and marks pages reached with
an action token `no-store` and `no-referrer`. The front proxy still sees the
incoming URL before the application; its exact access-log redaction or
exclusion must be verified in the private W8M deployment runbook before
activation. Enabling account protection invalidates old identifier-bearing
links, so the private rollout must either wait for their bounded recovery
window or explicitly reissue them.

While protection remains disabled, rollback-mode links carry only the numeric
account ID, email, timestamp, and legacy proof. The verifier compares the email
and reconstructs the proof from the authoritative account-row names rather
than requiring those names in the URL. This also preserves older links when a
mail client discarded a trailing encoded name segment; the digest and recovery
timestamp checks remain mandatory.

Genelet SQL diagnostics log the prepared SQL shape and bind count only, never
bind values. Its response boundary removes password/MFA request values and
storage-only account digests/ciphertexts; JSON also omits action tokens. Model
writes strip client-submitted digest, ciphertext, token-digest, and expiry
fields even while protection is disabled, so a request cannot seed authority
for a later migration. Unexpected driver failures in protected requests render
only a generic application error and are logged by type, preventing a binary
unique-key value embedded in an error from becoming a second disclosure path.

Authorized display paths never return `*_hmac` or `*_cipher`. Generic account
topics/edit responses, middleman and ledger reports, login sessions, mail, and
the management API decrypt only after authorization. Agent topics/edit also
exclude the password hash, and agent password writes always pass through
bcrypt validation.

## Remaining Retirement Gates

Plaintext removal is a distinct, one-way migration and is not part of the
additive migration. Repository read/write, lifecycle, display, reporting,
management, opaque-token, throttle, and administrator-only numeric login-as
paths are prepared. `PlaintextRetired` makes protected writes omit legacy
columns and is intentionally false in the checked-in example.

The drop remains blocked on two external/review gates:

- production backfill, dual-read canary, shared-throttle, restart, rollback,
  proxy-log, outstanding-link, and key-rotation evidence retained privately;
  and
- a separately reviewed one-way migration that verifies parity, removes legacy
  routines and `adv_ip`/`pub_ip` plaintext history, makes protected fields
  non-null, and only then drops plaintext identifier columns/indexes.

Do not mutate the production database, runtime config, Redis, service, or
deployment merely because the repository implementation exists. The exact
`w8m.com` manifest and evidence belong in the private infrastructure repository.
