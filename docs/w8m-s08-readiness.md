# W8M S08 Identity readiness handoff

S08 completes its single-node Identity prerequisite at whole-milestone review
2/10 with no open P1/P2. This is sanitized operational evidence for W27.3,
not authorization for enrollment, capture or a live adapter/count attempt.
The owning [status](../tabilet/memory-bank/status-S08.md) preserves the failed
and consumed attempts, task reviews, source fixes and owner confirmations.

## Exact accepted production state

- yixin is the owner-confirmed sole HTTP node. The already accepted canary
  activation supplies the all-node rollout; no second deployment is needed.
- Selected release: `aofei-96456a2bf9bb_pzdesign-6767db40d0ff_genelet-fdeff9732d9e`.
  Its manifest SHA-256 is
  `0a0acf65277a0df21c703899c6d370df4baa04af2938b5b27e6d4d19991678ae`.
- Identity is enabled. Required TOTP roles are admin, pub, agent and analyst;
  advertiser enrollment stays voluntary. The dedicated W27 advertiser account
  has not been enrolled by this work.
- The common protected Identity key matches HTTP and the separate maintenance
  configuration. The running service also matches the unchanged account-data
  key ring. Account protection remains enabled and plaintext retired.
- Runtime and maintenance principals retain exactly their reviewed restricted
  rights. Final read-only comparison passes in 2.532 seconds, with 96 tables
  and the restored schema contract unchanged; role, proxy and dynamic-global
  privilege bypasses are absent. Previously passing audit-denial probes are
  reused rather than repeated. No DDL or clean-baseline replay was needed.

## Accepted checks and limits

The separately approved analyst canary passes ordinary enrollment, fresh TOTP
sign-in, POST logout and missing-CSRF refusal; mismatched account, analyst
mutation and ungranted report denials; recovery-code first use and refusal on
reuse; and both idle/absolute expiry followed by restored valid TOTP access.
The owner stores recovery material privately. HTTP results are corroborated
by separate aggregate audit/session checks; no credential material is retained
here. Session expiry is controlled timestamp shortening on only the unique
canary session, not a natural twelve-hour wait or global duration change.

Fresh owner advertiser sign-in without mandatory TOTP passes after activation.
The unchanged S08.4 release baseline supplies ordinary recovery-mail/page,
Publisher registration/activation and portal checks. No production password
change or administrator reset is claimed. Passing backend security/reset and
release/conformance evidence remains reusable on the unchanged source inputs.

The observed activation/canary window is 2026-10-01T00:33:03.567684Z through
01:16:47.448954Z, 43.7 minutes. Aggregate audits show six successful logins,
one expected failed login, one PermissionDenied, one recovery use, one recovery
rotation, one analyst creation, one TOTP enablement and three revocations.
Failure outcomes are zero. Across 364 service journal records, no panic/fatal,
database-driver/transport, protected-request or explicit audit-failure markers
are observed. This bounded observation is not a long-term availability claim.
Database general/slow query logging remains disabled; raw logs are not emitted.

Final health and exact rollback/key checks pass in 0.150 seconds: readiness
and health return 204, NTP is synchronized and source/config bindings match.
The complete independently recovered checkpoint and exact disabled original
remain available. Identity rollback preserves account protection and all
account/TOTP/session/audit rows. Retain the previous account-data key until
at least 2026-10-01T19:02:35Z for already issued protected action proofs.

## Receiving owner

W8M W27.3 must perform its separately authorized read-only enabled/optional-adv
check against this accepted state before closing its row. W27.4 enrollment
remains owner-controlled. W27.5 supervised capture and W27.7 live count each
retain exact authorization, fresh readiness and one-attempt gates. This
prerequisite grants no browser operation or runtime adoption.
