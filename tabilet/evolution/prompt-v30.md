# Prompt V30

Protect W8M interactive account identifiers at rest and in database lookup
without weakening the existing bcrypt password boundary or coupling account
data to Summer's general signing secret.

- cover advertiser, publisher, administrator, agent, and analyst roles;
- use a dedicated versioned environment key ring with deterministic
  role/field-scoped lookup digests and authenticated reversible ciphertext;
- keep bcrypt as the sole password verifier and add shared post-verification
  login throttling;
- migrate additively through backfill, dual read/write, canary, and rollback
  before any plaintext retirement; and
- replace identifier-bearing account-action links and identifier-based
  impersonation before the one-way drop gate.
