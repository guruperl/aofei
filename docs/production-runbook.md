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
| `aofei-unify.service` | `../pzdesign/cmd/unify` | HTTP admin, DSP bid, win/loss, impression, and click endpoints. |
| `aofei-nats-client.service` | `cmd/nats-client` | Consumes NATS log subjects into interval files. |
| `aofei-spread.service` | `cmd/spread` | Persists spread/cache NATS messages to files. |

Scheduled or manual jobs:

| Job | Binary | Purpose |
|---|---|---|
| Ledger interval/daily | `cmd/ledger` | Aggregates win/loss log files into ledger tables. |
| MaxMind refresh | `cmd/maxmind` | Rebuilds MaxMind JSON country/state maps from MySQL. |
| Cache population | `cmd/redis-cache` | Populates Redis and/or spread cache from MySQL. |

Command placement:

| Node role | Run | Do not run |
|---|---|---|
| HTTP/UI/ADX node | `aofei-unify.service`; `aofei-nats-client.service` when this node publishes logs that should be written locally or forwarded by the deployment | `cmd/ledger`; `cmd/redis-cache` timers |
| Cache-maintenance node | `cmd/redis-cache -cache=redis` from cron or a systemd timer; optional `-cache=spread|all` for full spread republish workflows | per-HTTP-node cache refreshers |
| Log aggregation node | `aofei-nats-client.service`; `cmd/ledger` interval and daily timers | HTTP-node-local ledger jobs |
| Spread/local-cache node | `aofei-spread.service` only when this node needs spread disk snapshots | ledger unless it is also the log aggregation node |

If log files are written on every HTTP node, production must either ship/merge
those files to one log aggregation node before ledger runs, or run separate
database-safe aggregation for each shard. The active recommendation is one log
aggregation node for ledger input.

External dependencies:

- MySQL for schema, admin data, campaign data, and ledger tables.
- Redis for mutable DSP state and Redis-mode static cache reads.
- NATS for best-effort log transport and spread/cache snapshot distribution.
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

Run Redis cache population as a singleton cron job or systemd timer on one
dedicated node with `cmd/redis-cache -cache=redis`; do not run one cache
refresher per `unify` node. Run `cmd/ledger` only on the log aggregation node
where the complete `log_winloss/winloss.<stamp>` stream is available.
Mutating `cmd/redis-cache`, `cmd/ledger`, `cmd/mid-callback-retry`, and
`cmd/winloss` executions also acquire Redis singleton locks.

Checked-in `etc/aofei.json` and `etc/summer.example.json` are examples. Generated
`etc/*.local.json` files are local-only artifacts and must not be copied into
production as-is.

Production secrets and live connection values are injected by deployment
tooling or root-owned config/environment files. Do not commit database
passwords, Redis credentials, SMTP credentials, session secrets, OAuth secrets,
tracking secrets, or cloud keys. DSP tracking URLs use `tracking_secret` in
`AOFEI`, or the `TRACKING_SECRET` environment fallback, to sign click redirect
win/loss, middleman callback, and cap-mutation tracker payloads. Set
`tracking_signature_ttl_seconds` to bound callback replay; the default is
86400 seconds.

Middleman fallback is disabled unless `middleman_enabled` is true in `AOFEI`.
When enabled, set `middleman_exchange_domain`, `middleman_timeout_ms`, and
`middleman_max_bidders_per_imp` deliberately. `trigger_mode='Always'` route
fanout is ignored unless `middleman_always_enabled` is also true. Middleman
callback proxying also requires `tracking_secret`, Redis, and a public
`middleman_callback_base_url` that points back to the `cmd/unify` HTTP service; set
`middleman_callback_ttl_seconds` and `middleman_callback_timeout_ms` according
to exchange callback latency expectations. Downstream callback URLs are rejected
when they resolve to loopback, private, link-local, unspecified, multicast, or
rebinding targets. Each active bidder
`credential_ref` names an environment variable visible to `cmd/unify`; its
value must be a JSON object of outbound HTTP headers. Do not put those header
values in MySQL, Redis, or checked-in config files.

Middleman reporting depends on `cmd/ledger` running on the node with the
complete `log_winloss` stream. Advertiser middleman reports use pay-side spend
from `daily_mid`; admin settlement reports use charge, pay, and margin from
`daily_mid`.

Operators manage middleman route groups, route bidders, and traffic targets in
Summer at `/goto/admin/g/midroute?action=topics`. Route edits update MySQL only;
run the singleton `cmd/redis-cache -cache=redis` or `-cache=all` refresh on the
dedicated cache node before expecting `cmd/unify` workers to use the new route
state.

Summer/Genelet CORS is exact-origin only. `ServerURL` is allowed by default, and
additional browser origins must be listed in `CORSOrigins`; other non-empty
`Origin` values receive HTTP 403 before routing.

Direct publisher SSP traffic is separate from Summer/Genelet admin CORS.
`cmd/unify` handles `POST/OPTIONS /pz` with permissive, credentialless endpoint
CORS because preflight cannot carry the direct `site` token body. Serving
authority is the validated `POST /pz` request: after packed token and cache
validation, browser requests must include `Origin` or `Referer` with a host that
exactly matches the cached publisher site host. `Origin: null`, malformed URLs,
missing browser headers, mismatched hosts, and subdomain variants return `403`
before cookies, bidding, or audit publishing. `platform:"sdk"` may omit both
headers, but any supplied `Origin` or `Referer` must also match. The
`aofei_pz_uid` cookie is browser-only; SDK/in-app requests do not read or set it
and use the existing device identity or UA+IP fallback path.

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

Keep `nats-client` separate when ledger files are required, keep `spread`
separate when spread/cache NATS messages should become disk snapshots, keep
`redis-cache` as a singleton scheduled job, and run `ledger` only on the log
aggregator node.

Example `aofei-nats-client.service`, installed separately from `unify`:

```ini
[Unit]
Description=Aofei NATS log file writer
After=network-online.target nats.service
Wants=network-online.target

[Service]
Type=simple
User=aofei
Group=aofei
Environment=AOFEI=/etc/aofei/aofei.json
ExecStart=/opt/aofei/bin/nats-client -interval=10
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Example singleton Redis cache timer on the cache-maintenance node:

```ini
[Unit]
Description=Aofei Redis cache refresh

[Service]
Type=oneshot
User=aofei
Group=aofei
Environment=AOFEI=/etc/aofei/aofei.json
ExecStart=/opt/aofei/bin/redis-cache -cache=redis
NoNewPrivileges=true
PrivateTmp=true
```

```ini
[Unit]
Description=Run Aofei Redis cache refresh every 15 minutes

[Timer]
OnBootSec=2m
OnUnitActiveSec=15m
Unit=aofei-redis-cache.service

[Install]
WantedBy=timers.target
```

Example ledger interval timer on the log aggregation node:

```ini
[Unit]
Description=Aofei interval ledger aggregation

[Service]
Type=oneshot
User=aofei
Group=aofei
Environment=AOFEI=/etc/aofei/aofei.json
ExecStart=/opt/aofei/bin/ledger -interval=10
NoNewPrivileges=true
PrivateTmp=true
```

```ini
[Unit]
Description=Run Aofei interval ledger aggregation every 10 minutes

[Timer]
OnBootSec=12m
OnUnitActiveSec=10m
Unit=aofei-ledger.service

[Install]
WantedBy=timers.target
```

Example daily ledger timer on the log aggregation node:

```ini
[Unit]
Description=Aofei daily ledger aggregation

[Service]
Type=oneshot
User=aofei
Group=aofei
Environment=AOFEI=/etc/aofei/aofei.json
ExecStart=/opt/aofei/bin/ledger -daily
NoNewPrivileges=true
PrivateTmp=true
```

```ini
[Unit]
Description=Run Aofei daily ledger aggregation

[Timer]
OnCalendar=*-*-* 00:20:00
Persistent=true
Unit=aofei-ledger-daily.service

[Install]
WantedBy=timers.target
```

Use the same `User`, `Group`, `Environment`, restart policy, and hardening
pattern for `spread`, changing only `ExecStart`. `redis-cache` and `ledger`
should normally be oneshot timer services, not long-running services.

The current server commands do not document an application-level graceful
shutdown protocol. Operators should use systemd stop timeouts, observe logs, and
avoid deploys during active ledger/cache maintenance windows until signal
handling is hardened.

## Build And Rollout

Build artifacts from a reviewed commit:

```bash
GOWORK=off go test ./...
GOWORK=off go install ./cmd/nats-client ./cmd/spread ./cmd/ledger ./cmd/maxmind ./cmd/redis-cache ./cmd/mid-callback-retry
(cd ../pzdesign && GOWORK=off go test ./... && GOWORK=off go install ./cmd/unify)
```

Copy binaries into a versioned release directory, update the active symlink or
binary paths, then restart services:

```bash
sudo systemctl daemon-reload
sudo systemctl restart aofei-spread.service
sudo systemctl restart aofei-nats-client.service
sudo systemctl restart aofei-unify.service
sudo systemctl enable --now aofei-redis-cache.timer
sudo systemctl enable --now aofei-mid-callback-retry.timer
sudo systemctl enable --now aofei-ledger.timer
sudo systemctl enable --now aofei-ledger-daily.timer
sudo systemctl status aofei-unify.service
```

Smoke checks:

- `cmd/unify` listens on `ServerPort` and serves expected admin/static paths.
- DSP bid endpoint returns the expected local or staging fixture response.
- Redis contains `pubmap`, `audience`, `creative`, `middleman:routes:v2`,
  fallback-only legacy `middleman:routes`, and `slot:<size_id>` cache families
  after cache population when Redis static-cache mode or middleman fallback is
  used.
- `cmd/redis-cache -cache=routes -read` shows current route-cache metadata when
  middleman routes are used.
- The middleman callback retry timer runs on one operations node and can read
  due rows without processing them via `cmd/mid-callback-retry -read`.
- In local/spread static-cache mode, spread files exist and bid nodes have
  loaded a current in-process static generation.
- NATS log subjects are written into the configured log directories.
- Ledger timers run only on the log aggregation node and see complete
  `log_winloss/winloss.<stamp>` files.
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
- Repopulate static cache after flushes, failover, bidder route changes, or
  schema/data changes when Redis static-cache mode or middleman fallback is
  used.
- Keep mutable-state key families such as `bothcap:<user_id>` and
  `upload:<adv_id>:<marker>` protected from accidental static-cache flushes.

NATS requirements:

- Provide the URL in `AOFEI`.
- Monitor service availability, subscription health, and dropped messages.
- Run `nats-client` as a separate service from `unify`; do not embed it in the
  HTTP process.
- Keep `nats-client` ledger-input directories at `0750` and generated log files
  at `0640`; they should not be world-readable or group/world-writable.
- Run `spread` on nodes that use local/spread static-cache mode so static
  snapshots can be persisted and reloaded.
- Restart `nats-client` and `spread` after NATS outages if subscriptions do not
  recover cleanly.

## Logs And Ledger

The unified HTTP service exposes stdlib expvar metrics at `/debug/vars`,
including bid/no-bid counters, audit queue depth/drops and publish errors,
middleman callback forwarding results, local cache reload and freshness status,
direct SSP request results, cap-refresh contention counters, and
`aofei_ssp_policy_rejections_total`. Local cache freshness includes
`aofei_local_cache_loaded_at_unix`, scrape-time
`aofei_local_cache_age_seconds`, and `aofei_local_cache_stale`. Alert on
non-zero sustained `aofei_audit_dropped_total`, rising
`aofei_bothcap_refresh_conflicts_total`, stale `aofei_local_cache_stale`, and
middleman callback retry command output where `stale_processing` stays
non-zero. Use `cmd/mid-callback-retry -json` for alerting automation; its stable
fields are `due`, `stale_processing`, `selected`, `succeeded`, `retrying`, and
`abandoned`.

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
that day are complete. When middleman fallback is enabled, these jobs also fill
`ledger_mid` and `daily_mid` from middleman callback metadata.

Use standalone `cmd/ledger` for targeted replays with explicit `-timestamp`
values.

Run ledger only where the complete win/loss log stream is present. If each HTTP
node writes local `winloss.<stamp>` files, ship or merge those files into the
log aggregation node before the ledger timer runs.

Keep logs on persistent storage with rotation, backup, and retention policies
defined by operations. The repository does not own production log retention.

## Static, Uploads, And Templates

`DocumentRoot` is the static file root. Requests containing parent-path
segments are rejected before file serving.

`UploadDir` must be writable by the service user and should be outside the
static document root unless a separate review approves direct public serving.
Templates should be read-only to the service user. The local template tree is
the sibling `pzdesign/tmpls`; production config may point there or to a
read-only deployed copy. Static UI assets should be served from the matching
`pzdesign/www` tree or an equivalent deployed asset root.

## Auth Compatibility

Current Summer admin and user flows verify stored bcrypt password hashes through
Genelet's `Password_hash` issuer field. Existing SHA1-era credentials must be
reset before production use.

## Historical Material

[legacy-operations.md](legacy-operations.md) and `backup/` are historical-only.
Use them only as context when interpreting old deployments; do not use them as
active setup instructions or as sources for credentials.
