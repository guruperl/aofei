# Lessons

Keep concise, reusable lessons that still affect decisions. Consult relevant
topics before substantial changes; this is not a session log.

Maintain lessons when ordinary work produces reusable, evidence-backed
learning, and consolidate them during milestone closure. Keep applicable
lessons even after their supporting milestone completes.

For each lesson, use a descriptive heading and record when it applies, the
lesson, why it matters, and links to supporting tasks or verification. Keep
product facts in `product.md`, contracts in `architecture.md`, and commands in
`tech-stack.md`. Merge duplicates and revalidate historical evidence before
applying it. Before materially replacing or removing a lesson, preserve its
previous wording in Git history rather than a separate journal.

## Recovery and migration drills need the deployed database version

For a one-way migration, restore an encrypted writer-frozen snapshot into an
isolated instance using the deployed MySQL image. Compare schema, object
inventory and every table's counts/checksums before testing rejection,
rotation and post-migration decryption. A passing different-version fixture
does not establish production-version compatibility. See
[S07's MySQL 8.0.41 drill](status-S07.md#protected-production-canary-and-remaining-stop-gates).

## Check the host owner and executable storage before a private operation

Derive file ownership from the actual SSH UID; local and remote UIDs can differ.
Test binary execution before writer changes. A private tmpfs can be mounted
non-executable: keep secrets there and place only the nonsensitive tool binary
on an executable path. Preserve failed preflights and use fresh evidence for
the correction. See [S07's stopped preflights](status-S07.md#protected-production-canary-and-remaining-stop-gates).

## A new token lifetime does not prove old links expired

During staged account protection, inspect the legacy verifier's actual expiry
conditions. Identity-disabled reset and legacy activation paths can differ
from protected token lifetimes. An observation period or account-row timestamp
cannot prove old mail has drained; establish expiry/reissuance and recovery
continuity before irreversible retirement. See
[the action-proof contract](../../docs/account-identifier-protection.md#account-action-proofs)
and [S07's open link gate](status-S07.md#protected-production-canary-and-remaining-stop-gates).
