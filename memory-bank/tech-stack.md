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

## Cache Commands

Populate Redis from MySQL:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

Read Redis cache content:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis -read
```

Populate spread files and Redis/NATS-backed paths:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=all
```

## Verification

Current smoke verification:

```bash
GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
git diff --check
```

`go test ./...` is a milestone target, not yet a clean baseline, because
historical Go files under `backup/` still need to be excluded from normal
package discovery or moved under a non-package history layout.

## External Requirements

- Docker CLI and a working Docker daemon.
- Internet access only when pulling Docker images or Go modules.
- No production credentials are required for local development.
