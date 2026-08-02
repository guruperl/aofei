# Prompt V23

Replace W8M's password-plus-stateless-cookie control-plane boundary with an
explicitly deployable, higher-assurance identity and least-privilege contract.

- add database-backed opaque sessions, TOTP enrollment/recovery, revocation,
  and recent reauthentication without breaking the opt-out legacy rollout;
- name server-side action and resource permissions for commercial, report,
  seller, bidder, route, credential-reference, and security operations;
- introduce an exact-grant, read-only analyst role without turning internal
  Summer JSON into a public API;
- retain secrets only as encrypted values or keyed digests and record immutable
  redacted security evidence under a separate retention class; and
- keep recovery, privileged maintenance, key ownership, migration, rollout,
  and rollback explicit operator responsibilities rather than hidden UI
  behavior.
