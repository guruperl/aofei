# Result V28

S06 adds a default-off public-account abuse boundary across Genelet, pzdesign,
and the W8M deployment:

- advertiser/publisher registration and recovery forms render scoped
  Cloudflare Turnstile widgets in Chinese and English;
- the server verifies each single-use token's hostname/action before password
  hashing, Redis, Google OAuth, database mutation, or Gmail;
- one Redis script atomically enforces IP, normalized-email, and global windows
  under expiring HMAC-pseudonymous keys;
- only an explicitly trusted proxy chain can supply the client address used by
  Siteverify, quotas, and account registration; and
- fixed-cardinality metrics and a fail-closed startup/operator contract make
  dependency failure and rollback observable.

Repository implementation and local verification are complete. Production
activation remains open because the discovered Cloudflare API token is invalid;
no widget, edge rule, secret file, or live-protection claim is fabricated.
