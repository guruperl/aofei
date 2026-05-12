# Tech Stack

## Languages And Module

- Language: Go
- Module: `github.com/genelet/winter`
- `go.mod`: Go 1.22 with toolchain 1.23.5

Use `GOWORK=off` for local commands from this repository. The parent `go.work`
currently does not include this module path.

## Core Dependencies

- MySQL driver: `github.com/go-sql-driver/mysql`
- Redis client: `github.com/mediocregopher/radix/v4`
- NATS client: `github.com/nats-io/nats.go`
- OpenRTB: `github.com/mxmCherry/openrtb/v17`
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
```

These files are local artifacts and are ignored by git.

Summer/Genelet admin tests must use `SUMMER`; the Genelet config format uses
upper-case keys such as `ConnectArray`, `Template`, and `UploadDir`.

## Schema Baseline Commands

`etc/step4_init.sql` is the active schema and baseline-data contract. The local
helper keeps schema stewardship commands under the same Docker workflow:

```bash
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh dump-schema
./scripts/aofei-local.sh diff-schema
```

`check-sql` rejects explicit `DEFINER=` clauses and legacy `eightran` auth
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
`.local/logs/log_winloss/`. `cmd/maxmind` is buildable and inventoried in M6;
full external City `.mmdb` validation remains M7 scope.

## Verification

Current smoke verification:

```bash
bash -n scripts/aofei-cache-smoke.sh
GOWORK=off go test ./cmd/redis-cache ./cmd/spread -run '^$'
GOWORK=off go test ./cmd/ledger ./cmd/nats-client ./cmd/winloss ./cmd/spread ./cmd/maxmind
GOWORK=off go test ./dsp -run 'Controller|Win|Loss|^$'
GOWORK=off go test ./cmd/spread -run 'Test'
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
GOWORK=off go test ./match -run 'Test.*Cap|TestFcap'
GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
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

`go test ./...` is a milestone target, not yet a clean baseline, because
historical Go files under `backup/` still need to be excluded from normal
package discovery or moved under a non-package history layout.

## External Requirements

- Docker CLI and a working Docker daemon.
- Internet access only when pulling Docker images or Go modules.
- No production credentials are required for local development.
