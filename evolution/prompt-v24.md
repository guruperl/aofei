# Prompt V24

Expose advertiser campaign management as a stable external contract without
publishing internal Summer/Genelet routes or browser identity.

- add an independently gated `/api/v1` boundary for advertiser-owned campaign,
  ad-group, creative, targeting, activation-operation, and delivery-report
  resources;
- authenticate with revocable, expiring, least-privilege service credentials
  whose raw bearer values are never stored, while retaining recent-MFA human
  control over credential lifecycle;
- make writes strict, account-scoped, idempotent, optimistically versioned, and
  immutably audited without granting accounting, publisher, bidder, route, or
  approval authority;
- distinguish durable configuration acceptance from a bounded cache
  publication acknowledgement, including concurrent mutations; and
- publish OpenAPI, typed-client, compatibility, quota, migration, retention,
  rollout, and rollback contracts before activation.
