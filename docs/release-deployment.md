# Immutable Backend Releases

This guide defines the generic, non-secret build and deployment boundary for
the Aofei/Pzdesign/Genelet HTTP backend. Hostnames, service paths, container
identities, secret-file locations, activation history, and provider-specific
health policy belong to a private infrastructure repository rather than this
source repository.

## Why The Release Is A Bundle

`cmd/unify` is compiled from the sibling `pzdesign` module, imports Aofei and
Genelet through reviewed sibling replacements, loads each registered Summer
`component.json` below `ProjectRoot`, renders the configured template tree, and
serves the configured static tree. A binary copied without those exact assets
is not a complete release.

Each release therefore contains:

```text
RELEASE_ID
manifest.json
checksums.sha256
bin/unify
bin/config-preflight
bin/aofei-deploy
assets/pzdesign/summer/*/component.json
assets/pzdesign/tmpls/
assets/pzdesign/www/
```

New manifests use schema version 2 and kind `aofei-http-backend`. They record
the exact clean, published Aofei, Pzdesign, and Genelet commits; Go toolchain;
database/accounting contract; component count; and artifact count.
`checksums.sha256` covers every executable, runtime asset, `RELEASE_ID`, and the
manifest. Release directories contain no symlinks or group/world-writable
paths. Verification accepts legacy schema-v1 `w8m-http-backend` bundles only as
a read/rollback bridge; the builder never writes that format.

## Build And Verify

All three repositories must be clean and exactly equal to their configured
upstream branches. A local commit, dirty asset, or unpushed sibling revision
fails before tests or release output. The builder runs all three Go test suites
and refuses to overwrite an existing output path.

```bash
release_parent=$(mktemp -d)
./scripts/aofei-release.sh build \
  --output "$release_parent/http-backend"
./scripts/aofei-release.sh verify \
  "$release_parent/http-backend"
```

The output can be transferred to the deployment host only through a channel
that preserves file contents and modes. Verify it again after transfer and
before activation.

## Private Environment Manifest

The private infrastructure repository owns one manifest per deployment
environment. It must contain no credential value, database row, request data,
or customer identifier. It names only:

- canonical and accepted public origins;
- release root and atomic `current` symlink;
- systemd scope, unit, origin health/readiness URLs, and public smoke URLs;
- paths to owner-managed application configs and secret environment files;
- exact dependency names plus immutable image IDs/digests;
- expected database schema and accounting contract;
- rollback and release-retention policy.

The manifest is strict: unknown fields, unsafe or colliding paths, a service
that bypasses the atomic `current` selection, a systemd manager awaiting reload,
loaded `ExecStart`, working directory, `AOFEI`, `SUMMER`, or environment files
that differ from the declared owner-managed values, non-loopback direct probes,
public probes outside accepted HTTPS origins, mutable dependency identities,
and automatic release deletion fail before effects. Target values never become
defaults in the generic command.

Secrets remain in the host's owner-readable environment/configuration files.
The manifest may name those files so a deploy preflight can require their
existence and modes, but it never copies their contents.

## Generic Command

After the release has been independently verified, a private realization calls
the bundled command with its exact target-owned inputs:

```bash
release=/absolute/verified/release
"$release/bin/aofei-deploy" \
  -manifest /absolute/private/environment.json \
  -unit-template /absolute/private/service.unit \
  -history-dir /absolute/private/history \
  preflight "$release"

"$release/bin/aofei-deploy" \
  -manifest /absolute/private/environment.json \
  -unit-template /absolute/private/service.unit \
  -history-dir /absolute/private/history \
  deploy "$release"
```

`validate`, `verify-release`, `status`, and one-time `bootstrap` are also
available. `bootstrap` is invalid once an atomic current selection exists.
`status` and public smoke probes may contact the target and therefore retain
environment-specific authorization.

## Activation State Machine

The environment deployer must serialize activation with an owner-only lock and
perform this order:

1. Verify the release checksums, manifest, source provenance, and runtime asset
   inventory.
2. Require exact dependency identities, application config preflight, database
   schema/accounting contract, and any cache migration gate owned by the
   release.
3. Copy the release to a new immutable directory. Never alter an activated
   release.
4. Atomically replace the `current` symlink and restart the configured service.
5. Require the manager to have loaded the exact release-backed service paths
   and config environment, then require `active`, direct-origin `/healthz`,
   direct-origin `/readyz`, a new process id, and the configured public smoke
   responses.
6. On any failure, atomically restore the prior symlink, restart it, and require
   prior health before returning failure.
7. Create a credential-free `started` deployment record before selection and
   atomically finalize it with release id, manifest digest, old/new targets,
   result, timestamps, process identities, and health outcomes. A crash cannot
   erase the fact that activation began. If successful-record finalization
   fails, restore and verify the prior release (or the complete legacy service
   during bootstrap) before returning failure.

Public edge policy may intentionally hide `/healthz`, `/readyz`, or
`/debug/vars`; direct-origin checks remain mandatory and public checks use only
the environment's explicitly listed URLs/statuses.

## Schema And Cache Boundary

A release bundle is not authorization to run a schema migration, rebuild a
cache generation, enable a default-off feature, or mutate an edge/provider
configuration. The private deployment preflight verifies the state required by
the release. If state is older, activation stops and the separately reviewed
migration/runbook must be authorized and completed first.

## Rollback And Retention

Rollback switches one symlink to the complete prior release; it never rebuilds
an old binary against new assets or copies selected files backward. Retain at
least the selected and immediately prior releases. Removal of older releases is
a separate exact-target retention action and must never follow a failed
activation.

The deploy path verifies the selected prior bundle and installed unit before
changing the symlink. Health probes bypass ambient HTTP proxies, refuse
redirects, and restrict direct checks to loopback URLs from the strict
manifest. Rollback must pass both direct and public probes before it can be
recorded as recovered.
