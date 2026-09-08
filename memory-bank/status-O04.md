# Status O04 — Generic release deployment and bootstrap

State: `[~]` In progress

## Goal

Provide a reusable, strict deployment engine for an Aofei/Pzdesign/Genelet
website realization without embedding W8M host policy in this public source
repository.

## Tasks

| Item | State | Notes |
| --- | ---: | --- |
| Generic release contract | `[+]` | New bundles write generic schema-v2 `aofei-http-backend`, include the deployer, and bind all executable revisions/toolchains; verification reads legacy W8M v1 only for rollback. |
| Strict environment contract | `[+]` | Strict decoding rejects unknown fields, path overlap, service ambiguity, unsafe health/dependency/database/retention policy, and owner/file drift before effects. |
| Deployment state machine | `[+]` | Preflight, status, one-time bootstrap, immutable installation, atomic activation, independent-context rollback, config projection, locking, and bounded history are implemented. |
| Synthetic verification | `[+]` | Fixtures prove activation, cancellation and health rollback, unrecovered-history finalization, mutation-free preflight failure, bootstrap restoration, and malformed manifest/release rejection without host contact. |
| Documentation and private handoff | `[-]` | Public release/runbook/memory contracts are updated; adoption by the private W8M realization and exact-host evidence remain pending. |

## Acceptance

- Generic code and tests contain no W8M hostname, environment name, host path,
  Docker identity, credential path, or deployment-history value.
- New releases serialize `aofei-http-backend` v2 and contain the generic
  deployer; legacy `w8m-http-backend` v1 is read-only compatibility.
- All effect-capable commands validate strict manifest and release provenance;
  deployment validates a usable prior release before changing selection.
- Failed activation restores the exact prior selection, verifies its origin and
  public health, and writes a bounded credential-free rollback record.
- Schema/cache/feature/provider/dependency/retention changes are neither
  implemented nor implied.
- Full Aofei gates and the bounded review-fix gate pass before closeout.

## Review limit

At most ten review/fix iterations. Record each P1/P2 finding before repair and
stop only after a clean pass or the limit is reached.

## Review-Fix Gate

- Iteration 1 (2026-09-08): the implementation review found and resolved the
  following blockers before private adoption:
  - P1: real and legacy release manifests contain the full spaced `go version`
    value, which the first provenance validator rejected. Display metadata now
    allows spaces but rejects controls; new v2 bundles record the pinned build
    toolchain and bind it to every executable, while legacy v1 retains read
    compatibility.
  - P1: activation cancellation reused the canceled caller context for
    rollback. Recovery now always uses an independently bounded context, and a
    cancellation test proves the prior release is restored and rechecked.
  - P2: a checksum-valid prior bundle was treated as rollback-ready without
    requiring the installed unit, active service, PID, direct health/readiness,
    and public smoke. All are now preflight requirements before selection.
  - P2: release-tree root permissions, checksum truncation, dirty binaries,
    hard links, colliding mutable paths, unsafe Docker image arguments, and
    symlink release targets were insufficiently rejected. Both verifier layers
    and fixtures now cover those cases.
  - P2: bootstrap did not preflight the legacy unit, copied state through
    truncating paths, and could stop rollback after its first restore error.
    Unit safety is mandatory, copies use no-follow atomic replacement, and
    rollback attempts every restore step under its independent context.
  - P2: an unsuccessful recovery could leave only a `started` record, and lock
    setup chmodded an inode before validating its owner/link metadata. Failed
    recovery is now atomically finalized as `rollback_failed`, and lock
    metadata is validated before chmod or work.

Focused package tests, vet, pinned staticcheck, the full Aofei package/vet/
staticcheck gates, documentation guard, scoped race suite, target-neutral value
scan, strict parsing of the private manifest, and legacy selected-release
verification pass.

- Iteration 2 (2026-09-08): two P2 findings were found after the first exact
  generic cutover and are resolved before final closeout.
  - The manifest declared owner-managed secret files, but preflight proved only
    their file metadata rather than requiring systemd to have loaded exactly
    that set through its unit/drop-ins. Preflight, bootstrap, status, and every
    post-restart verification now compare the manager's `EnvironmentFiles`
    projection with the strict manifest; mismatch fails without mutation.
  - Engine-owned unit/history inputs were checked against each other and the
    release root but could overlap another manifest-owned mutable state path.
    Construction now rejects overlap with every backend state/config/secret
    path, and a focused fixture covers it.
  - The source-side release verifier now rejects hard-linked inputs before it
    invokes the bundle's generic verifier.

Iteration 3 and final milestone closeout follow a corrected exact release and
private evidence update.
