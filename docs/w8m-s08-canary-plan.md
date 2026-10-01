# W8M S08.5 enabled-canary execution plan

S08.4 is accepted in local commit `0991279`. The immutable release is healthy,
HTTP Identity is disabled, the complete common-key checkpoint is independently
recovered, and the owner baseline sign-in, recovery, registration/activation
and portal checks pass. This preserves the reviewed operation plan;
it does not grant execution authority. The separately approved S08.5 run now
passes its analyst/security checks; final monitoring/review remains S08.6.

## Exact scope and effect

The read-only canary preparation on the sole node yixin passes. The private
candidate changes only `Identity.Enabled` from false to true; key versions,
permissions, database credentials, session durations, account protection and
all other configuration stay equal. Restart the same installed service/release
through the operator-owned feature window. No rebuild, source pull, schema
migration, Cloudflare change or W27 browser operation is needed.

This single-node canary enables Identity for the production service. Existing
roles admin, pub, agent and analyst require TOTP enrollment; adv remains optional.
This is not an account-isolated switch. Create one owner-controlled analyst
through the audited maintenance CLI in its proven Unix-socket namespace,
with its kernel UID attribution and separate restricted principal. Create no
permission grants. Its password is entered once through a private masked input
and supplied only through the CLI environment. Keep the identifier, numeric ID
and credential material outside repositories and shared evidence.

## Sequence and evidence

1. Obtain the exact enabled-canary authorization required by S08.5, then recheck
   release/config/key bindings, protected schema/grants, NTP, readiness and
   complete recovery/rollback availability under the exclusive runtime guard.
2. Consume a new feature window, atomically install the reviewed candidate and
   restart the existing service. Verify enabled state, common-key parity,
   readiness/health and an ordinary advertiser sign-in without mandatory TOTP.
3. Consume one analyst creation through the published audited CLI. Resolve its
   terminal result and the specific creation audit before continuing. Preserve
   any failure or uncertain result; do not automatically retry.
4. The owner uses an ordinary browser for analyst enrollment and saves the seed
   and recovery codes privately. Verify current-code sign-in, POST/CSRF logout,
   one recovery-code sign-in and refusal of that same code on reuse. Collect
   only outcomes; do not read the seed, code, session cookie or browser capture.
5. Verify cross-account and analyst mutation denials using the canary's own
   session and safe requests that must stop at authorization. Confirm audit
   insertion with bounded aggregate checks, without raw account or audit logs.
6. Verify idle and absolute expiry using the existing passing deterministic
   clock tests plus separate canary sessions whose timestamps are shortened
   under an exact account/session guard. These narrowly scoped changes only
   reduce the canary's session validity; never change global durations, customer
   sessions, grants or audit rows. Fresh HTTP requests must reject both expired
   states. Record this as controlled timestamp testing, not natural twelve-hour
   observation. This action must be included in the exact operation approval.
7. Recheck health, audit/database failure indicators and rollback availability.
   Complete S08.5 only after every required result is established. S08.6 retains
   monitoring and whole-milestone review; W27.3 remains deferred until S08 closes.

## Rollback and stop

Restore the exact recovered disabled configuration under the exclusive guard
and restart the same service when a definitive failed activation permits
rollback. Preserve account protection enabled/retired, the common key,
analyst/TOTP/recovery/session rows and immutable audit evidence. Do not delete
canary rows, restore an old populated database, replay a consumed operation or
launch competing service windows. Uncertain outcomes require inspection before
another mutation; no result is inferred from a timeout or a process exit.

## Test selection

Reuse unchanged accepted release/source, backend-conformance and S02 focused
security evidence. No native browser qualification or unrelated full suite is
needed for a private feature toggle. Read-only candidate preparation took
0.180 seconds. The service window should take seconds to a few minutes based
on S08.4's 14.605-second installation; human enrollment and individual browser
checks have owner-dependent duration. Keep each check bounded and preserve its
failure detail. The controlled expiry checks avoid a twelve-hour development
wait while retaining the production predicate and focused clock-test evidence.

See [S08 status](../tabilet/memory-bank/status-S08.md) for acceptance/stop gates
and [Identity contract](identity-access-security.md) for the authoritative
security boundary. Host/deployment inputs remain infrastructure-owned and
private runtime/key/configuration copies remain outside Git.
