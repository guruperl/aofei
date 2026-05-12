# Tech Stack

## Languages And Module

- Language: Go
- Module: `github.com/genelet/winter`
- `go.mod`: Go 1.22 with toolchain 1.23.5

Use `GOWORK=off` for local commands from this repository. The parent `go.work`
currently does not include this module path, and plain `go list ./...` fails
from this checkout under that parent workspace.

## Core Dependencies

- MySQL driver: `github.com/go-sql-driver/mysql`
- Redis client: `github.com/mediocregopher/radix/v4`
- NATS client: `github.com/nats-io/nats.go`
- OpenRTB: `github.com/prebid/openrtb/v20`
- Logging: `go.uber.org/zap`
- Geo/IP and user agent helpers: local `maxmind/`, `github.com/mssola/user_agent`

## Local Services

The supported local runtime uses Docker:

| Service | Image | Default bind |
|---|---|---|
| MySQL | `mysql:8.0.41` | `127.0.0.1:3307` |
| Redis | `redis:7-alpine` | `127.0.0.1:6379` |
| NATS | `nats:2-alpine` | `127.0.0.1:4222` |

Main helper:

```bash
./scripts/aofei-local.sh up
./scripts/aofei-local.sh reset
./scripts/aofei-local.sh load
./scripts/aofei-local.sh sample
./scripts/aofei-local.sh reset-sample
./scripts/aofei-local.sh status
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh dump-schema
./scripts/aofei-local.sh diff-schema
./scripts/aofei-local.sh install
./scripts/aofei-local.sh down
```

## Runtime Config

Generated local configs:

```bash
etc/aofei.local.json
etc/summer.local.json
```

Environment variables:

```bash
AOFEI="$PWD/etc/aofei.local.json"
SUMMER="$PWD/etc/summer.local.json"
TRACKING_SECRET="..."
```

These files are local artifacts and are ignored by git.
`tracking_secret` in the DSP config signs generated `/imp` and `/clk` tracking
URLs; when omitted, `TRACKING_SECRET` is used as the fallback.

Summer/Genelet admin tests must use `SUMMER`; the Genelet config format uses
upper-case keys such as `ConnectArray`, `Template`, and `UploadDir`.

Production defaults are `/etc/aofei/aofei.json` and
`/etc/aofei/summer.json`, passed through `AOFEI` and `SUMMER`. The production
runbook is [docs/production-runbook.md](../docs/production-runbook.md).
Summer/Genelet CORS allows the exact `ServerURL` origin plus exact entries in
`CORSOrigins`.
Genelet framework contracts are documented in
`docs/genelet-manual.md`; Summer admin module and cache-side-effect conventions
are documented in `docs/summer-ui-structure.md`.

## Schema Baseline Commands

`etc/step4_init.sql` is the active schema and baseline-data contract. The local
helper keeps schema stewardship commands under the same Docker workflow:

```bash
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh dump-schema
./scripts/aofei-local.sh diff-schema
```

`check-sql` rejects explicit `DEFINER=` clauses and legacy account-name auth
references in `etc/step4_init.sql`. `dump-schema` writes a normalized current
Docker schema to ignored `.local/schema/aofei.schema.sql`. `diff-schema`
rebuilds a temporary database from `etc/step4_init.sql`, normalizes both dumps,
diffs baseline against the current Docker schema, and drops the temporary
database on exit.

When schema changes are intentional, update `etc/step4_init.sql`, rebuild with
`reset && load`, run `check-sql` and `diff-schema`, and update
`docs/database-baseline.md` plus the memory bank if the inventory or workflow
changed.

## Cache Commands

Run the one-command Docker cache smoke:

```bash
./scripts/aofei-cache-smoke.sh
```

Populate Redis from MySQL:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

Run the bid-path smoke after `reset-sample` and Redis cache population:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go test ./dsp -run 'Test.*Smoke'
```

Read Redis cache content:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis -read
```

Populate spread/NATS cache messages:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=spread
```

Persist spread messages to `.local/spread/` by running the receiver in another
terminal before `-cache=spread` or `-cache=all`:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/spread
```

Populate spread/NATS and Redis together:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=all
```

Expected Redis cache families are `pubmap`, `audience`, `creative`, and
`slot:<size_id>` hashes keyed by slot id. Expected spread directories are
`.local/spread/pubmap/`, `.local/spread/audience/`,
`.local/spread/creative/`, and `.local/spread/slot/<size_id>/`.

## Operational Commands

Active local operational command contracts are documented in
`docs/operational-commands.md`.

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/nats-client -interval=10
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/spread
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/ledger -interval=10
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/ledger -daily -timestamp=YYYY-MM-DD
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/winloss --bid=/bid/exchange.example.test win
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/maxmind -city=/path/to/GeoLite2-City.mmdb
```

Generated log directories are `.local/logs/log_request/`,
`.local/logs/log_response/`, `.local/logs/log_attribute/`, and
`.local/logs/log_winloss/`. `cmd/maxmind` reads MySQL country/state tables and
atomically writes the configured MaxMind JSON path, normally
`etc/maxmind.json`.

## MaxMind Assets

`etc/maxmind.json` is the active geodata config reference. Its `city_file`
currently points to the external GeoLite2 City database at
`/media/GeoLite2-City.mmdb`.

Ignored optional local assets:

```bash
external/GeoLite2-City.mmdb
etc/GeoLite2-City.mmdb
etc/qq-pz.dat
```

Compile and pure-unit tests must pass without those files. Full lookup tests in
`maxmind` and `maxmind/ipsearch` skip with explicit messages when local assets
are absent. `AOFEI_GEOLITE_CITY_FILE` can point `maxmind` tests at a downloaded
City `.mmdb`; otherwise they fall back to `external/GeoLite2-City.mmdb` and then
`etc/GeoLite2-City.mmdb`. Details live in `docs/maxmind-runtime.md`.

## Verification

Canonical package verification:

```bash
GOWORK=off go test ./...
```

`GOWORK=off go list ./...` should not include `github.com/genelet/winter/backup`;
historical Go helpers in `backup/` carry the `ignore` build tag.

Useful non-gating local checks:

```bash
bash -n scripts/aofei-doc-check.sh
./scripts/aofei-doc-check.sh
bash -n scripts/aofei-cache-smoke.sh
GOWORK=off go test ./cmd/redis-cache ./cmd/spread -run '^$'
GOWORK=off go test ./cmd/ledger ./cmd/nats-client ./cmd/winloss ./cmd/spread ./cmd/maxmind
GOWORK=off go test ./maxmind ./maxmind/ipsearch
GOWORK=off go test ./dsp -run 'Controller|Win|Loss|^$'
GOWORK=off go test ./cmd/spread -run 'Test'
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
GOWORK=off go test ./match -run 'Test.*Cap|TestFcap'
GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
GOWORK=off staticcheck -checks=SA* ./...
GOWORK=off staticcheck ./dsp ./match ./acl ./uploaded ./cmd/spread ./cmd/winloss ./cmd/unify ./cmd/redis-cache
git diff --check
```

Admin compatibility verification:

```bash
./scripts/aofei-local.sh reset-sample
GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer ./summer/pub ./summer/slot
GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./genelet
```

Schema baseline verification:

```bash
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh diff-schema
```

Docker smoke/admin/schema checks and staticcheck are documented follow-ups, but
they are not M8 package-gate blockers.

## External Requirements

- Docker CLI and a working Docker daemon.
- Internet access only when pulling Docker images or Go modules.
- No production credentials are required for local development.
