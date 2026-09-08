# Result V30

S07 changes the account-security direction from the original password-HMAC and
shared-secret proposal to a bcrypt-preserving identifier-protection boundary:

- deterministic HMAC is used only for normalized identifier lookup;
- AES-256-GCM supports authorized identifier display and mail;
- a dedicated current/previous environment key ring isolates lookup,
  encryption, Summer signing, S02 identity, and provider secrets;
- shared login throttling observes the real post-bcrypt outcome and stores only
  expiring pseudonymous Redis state; and
- additive schema/backfill, dual read/write, opaque action tokens, canary, and
  a later separately authorized plaintext retirement form the rollout.

Repository work is in progress and default-off. No production migration,
configuration change, service reload, or plaintext drop is implied.
