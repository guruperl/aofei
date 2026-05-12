# Local Docker Runtime

The current supported local runtime is managed by
[`scripts/aofei-local.sh`](../scripts/aofei-local.sh).

## Services

| Service | Container | Image | Default bind |
|---|---|---|---|
| MySQL | `aofei-mysql` | `mysql:8.0.41` | `127.0.0.1:3307` |
| Redis | `aofei-redis` | `redis:7-alpine` | `127.0.0.1:6379` |
| NATS | `aofei-nats` | `nats:2-alpine` | `127.0.0.1:4222` |

## Common Commands

Start services and generate local configs:

```bash
./scripts/aofei-local.sh up
```

Reset the database, load the active baseline, and add sample data:

```bash
./scripts/aofei-local.sh reset-sample
```

Show service and database status:

```bash
./scripts/aofei-local.sh status
```

Stop services without deleting Docker volumes:

```bash
./scripts/aofei-local.sh down
```

Install local Go command binaries:

```bash
./scripts/aofei-local.sh install
```

## Generated Configs

The helper writes:

```text
etc/aofei.local.json
etc/summer.local.json
```

These files contain local Docker connection strings and are ignored by git.

Set them explicitly when running commands:

```bash
AOFEI="$PWD/etc/aofei.local.json"
SUMMER="$PWD/etc/summer.local.json"
```

## Redis Cache

Populate Redis from MySQL:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

Inspect Redis cache content through the app reader:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis -read
```

Check Redis directly:

```bash
./scripts/aofei-local.sh redis-status
```

Flush local Redis:

```bash
./scripts/aofei-local.sh redis-flush
```

## NATS

NATS is started by `up` and `reset-sample`. It can also be controlled directly:

```bash
./scripts/aofei-local.sh nats-up
./scripts/aofei-local.sh nats-status
./scripts/aofei-local.sh nats-down
```

The generated DSP config points to:

```text
nats://127.0.0.1:4222
```
