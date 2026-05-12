# Production Runbook

This is the current production/operator entry point for `aofei` / `winter`.
Local Docker development remains documented separately in
[local-docker-runtime.md](local-docker-runtime.md).

## Deployment Model

The production target is a Linux host running systemd-managed services. Docker
Compose is not the production contract for this repository.

Long-running services:

| Service | Binary | Purpose |
|---|---|---|
| `aofei-unify.service` | `cmd/unify` | HTTP admin, DSP bid, win, and loss endpoints. |
| `aofei-nats-client.service` | `cmd/nats-client` | Consumes NATS log subjects into interval files. |
| `aofei-spread.service` | `cmd/spread` | Persists spread/cache NATS messages to files. |

Scheduled or manual jobs:

| Job | Binary | Purpose |
|---|---|---|
| Ledger interval/daily | `cmd/ledger` | Aggregates win/loss log files into ledger tables. |
| MaxMind refresh | `cmd/maxmind` | Rebuilds MaxMind JSON country/state maps from MySQL. |
| Cache population | `cmd/redis-cache` | Populates Redis and/or spread cache from MySQL. |

External dependencies:

- MySQL for schema, admin data, campaign data, and ledger tables.
- Redis for runtime DSP cache reads.
- NATS for log and spread/cache message transport.
- MaxMind GeoLite2 City `.mmdb` file stored outside the repository.
- Static document root, upload directory, template directory, and log
  directories owned by the deployment.

## Filesystem Layout

Recommended default paths:

```text
/etc/aofei/aofei.json
/etc/aofei/summer.json
/opt/aofei/bin/unify
/opt/aofei/bin/nats-client
/opt/aofei/bin/spread
/opt/aofei/bin/ledger
/opt/aofei/bin/maxmind
/opt/aofei/bin/redis-cache
/var/lib/aofei/uploads
/var/lib/aofei/spread
/var/lib/aofei/maxmind/GeoLite2-City.mmdb
/var/log/aofei/log_request
/var/log/aofei/log_response
/var/log/aofei/log_attribute
/var/log/aofei/log_winloss
```

Create a dedicated service user, for example `aofei`, and give it read access
to config files, execute access to binaries, write access to upload/spread/log
directories, and no repository write access.

## Configuration

`cmd/unify` reads both configs:

```text
AOFEI=/etc/aofei/aofei.json
SUMMER=/etc/aofei/summer.json
```

`cmd/nats-client`, `cmd/spread`, `cmd/ledger`, `cmd/maxmind`, and
`cmd/redis-cache` read `AOFEI`.

Checked-in `etc/aofei.json` and `etc/summer.json` are examples. Generated
`etc/*.local.json` files are local-only artifacts and must not be copied into
production as-is.

Production secrets and live connection values are injected by deployment
tooling or root-owned config/environment files. Do not commit database
passwords, Redis credentials, SMTP credentials, session secrets, OAuth secrets,
or cloud keys.

Summer/Genelet CORS is exact-origin only. `ServerURL` is allowed by default, and
additional browser origins must be listed in `CORSOrigins`; other non-empty
`Origin` values receive HTTP 403 before routing.

## systemd Units

Example `aofei-unify.service`:

```ini
[Unit]
Description=Aofei DSP and admin service
After=network-online.target mysql.service redis.service nats.service
Wants=network-online.target

[Service]
Type=simple
User=aofei
Group=aofei
Environment=AOFEI=/etc/aofei/aofei.json
Environment=SUMMER=/etc/aofei/summer.json
ExecStart=/opt/aofei/bin/unify
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Use the same `User`, `Group`, `Environment`, restart policy, and hardening
pattern for `nats-client` and `spread`, changing only `ExecStart`.

The current server commands do not document an application-level graceful
shutdown protocol. Operators should use systemd stop timeouts, observe logs, and
avoid deploys during active ledger/cache maintenance windows until signal
handling is hardened.

## Build And Rollout

Build artifacts from a reviewed commit:

```bash
GOWORK=off go test ./...
GOWORK=off go install ./cmd/unify ./cmd/nats-client ./cmd/spread ./cmd/ledger ./cmd/maxmind ./cmd/redis-cache
```

Copy binaries into a versioned release directory, update the active symlink or
binary paths, then restart services:

```bash
sudo systemctl daemon-reload
sudo systemctl restart aofei-spread.service
sudo systemctl restart aofei-nats-client.service
sudo systemctl restart aofei-unify.service
sudo systemctl status aofei-unify.service
```

Smoke checks:

- `cmd/unify` listens on `ServerPort` and serves expected admin/static paths.
- DSP bid endpoint returns the expected local or staging fixture response.
- Redis contains `pubmap`, `audience`, `creative`, and `slot:<size_id>` cache
  families after cache population.
- NATS log subjects are written into the configured log directories.
- No unexpected HTTP 403 appears for configured admin origins.

Rollback by restoring the previous binary release and restarting the same
services. Roll back immediately on startup failure, repeated bid-path errors,
cache deserialization errors, or ledger write failures.

## Database Lifecycle

For a new environment, load a reviewed schema/data baseline into MySQL, then run
deployment-specific migrations or data loads. `etc/step4_init.sql` is the local
baseline and schema reference; production migration tooling remains deployment
owned.

Before schema changes:

- Take a MySQL backup.
- Record the binary release and schema migration version.
- Run migrations in a staging database with representative data.
- Confirm cache population and bid/admin smoke checks.

After schema changes:

- Run cache population.
- Check admin login and representative model pages.
- Run ledger smoke on a known interval file when ledger tables changed.
- Compare schema drift against the expected release artifact where the
  deployment provides a drift tool.

Restore drills must be practiced outside production. A restore is not complete
until `cmd/redis-cache` has repopulated runtime cache and the HTTP/admin smoke
checks pass.

## Redis And NATS

Production Redis and NATS are owned by the deployment platform, not by
`scripts/aofei-local.sh`.

Redis requirements:

- Persistence and eviction policy are operational decisions.
- Monitor memory, connected clients, command errors, and key counts.
- Repopulate cache after flushes, failover, or schema/data changes.

NATS requirements:

- Provide the URL in `AOFEI`.
- Monitor service availability, subscription health, and dropped messages.
- Restart `nats-client` and `spread` after NATS outages if subscriptions do not
  recover cleanly.

## Logs And Ledger

`cmd/unify` publishes request, response, attribute, and win/loss events when
NATS is enabled. `cmd/nats-client` writes those subjects to:

```text
log_request/request.<stamp>
log_response/response.<stamp>
log_attribute/attribute.<stamp>
log_winloss/winloss.<stamp>
```

`cmd/ledger` consumes `winloss.<stamp>` files. Missing files are retryable input
errors, not successful empty intervals. Run interval ledger jobs after the
matching log rotation window closes, then run daily jobs after interval jobs for
that day are complete.

Keep logs on persistent storage with rotation, backup, and retention policies
defined by operations. The repository does not own production log retention.

## Static, Uploads, And Templates

`DocumentRoot` is the static file root. Requests containing parent-path
segments are rejected before file serving.

`UploadDir` must be writable by the service user and should be outside the
static document root unless a separate review approves direct public serving.
Templates should be read-only to the service user.

## Auth Compatibility

Current Summer admin and user flows retain SHA1-era password hash compatibility
for the existing schema and stored procedures. That is compatibility behavior,
not the long-term production authentication contract. A future auth migration
must define new hashing, migration, reset, and rollback behavior before changing
stored hashes or login queries.

## Historical Material

[legacy-operations.md](legacy-operations.md) and `backup/` are historical-only.
Use them only as context when interpreting old deployments; do not use them as
active setup instructions or as sources for credentials.
