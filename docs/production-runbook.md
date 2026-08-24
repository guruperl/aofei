# Production Runbook

This is the current production/operator entry point for `aofei` / `winter`.
Local Docker development remains documented separately in
[local-docker-runtime.md](local-docker-runtime.md).

The current lane baseline is summarized in the
[documentation and milestone index](README.md). The original D/P/R/I/S/A/O
sequence through A02 plus D04, D05, and S06 is implemented. P03 is in progress
with its threat contract, default-off v2 locator codec/runtime dual reader, and
default-off SDK/server authentication plus independent browser/App enforcement
complete; S05, O03, R03, and
A03 remain planned, and I02 remains demand-gated.
D03 remains partner activation-gated; I03, S02, S03, and A02 remain
independently disabled by default. Operators must assess open remediation
against any activation scope.
O02 defines an unclaimed objective, not evidence that production has already
achieved 99.9% or provider-backed RPO/RTO.

## Deployment Model

The production target is at least two Linux `cmd/unify` nodes in one region,
behind a health-checking load balancer, plus systemd-managed singleton worker
roles. Docker Compose is not the production contract for this repository.

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
| Middleman callback retry | `cmd/mid-callback-retry` | Processes durable retryable downstream callbacks. |
| Action reconcile/retention | `cmd/action-measurement` | Repairs unattributed actions and prunes expired action facts. |
| Experiment control/retention | `cmd/report-experiment` | Performs audited experiment transitions, bounded prune, and exact subject deletion from an authorized operations/privacy host. |
| Manual accounting | `cmd/accounting` | Creates and reconciles immutable statements, adjustments, approvals, settlements, corrections, and exports from an authorized accounting host. |
| Traffic-quality aggregate/review maintenance | `cmd/traffic-quality` | Ingests trusted bounded signal windows, reports rule health, and prunes expired evidence from a restricted host. |
| Hosted-payment health/event retention | `cmd/hosted-payment` | Reports aggregate funding/payout health and prunes only eligible expired webhook envelopes from a restricted host; it cannot move money. |
| Identity/API administration and retention | `../pzdesign/cmd/identity-admin` | Creates read-only analysts, changes exact grants, resets TOTP, and applies bounded account-security or management-API audit retention from a restricted maintenance host. |

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
/opt/aofei/bin/accounting
/opt/aofei/bin/action-measurement
/opt/aofei/bin/report-experiment
/opt/aofei/bin/maxmind
/opt/aofei/bin/redis-cache
/opt/aofei/bin/mid-callback-retry
/opt/aofei/bin/traffic-quality
/opt/aofei/bin/hosted-payment
/opt/aofei/bin/identity-admin
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

`cmd/nats-client`, `cmd/spread`, `cmd/ledger`, `cmd/action-measurement`,
`cmd/report-experiment`, `cmd/accounting`, `cmd/traffic-quality`,
`cmd/hosted-payment`, `cmd/maxmind`, and `cmd/redis-cache` read `AOFEI`.

Run Redis cache population as a singleton cron job or systemd timer on one
dedicated node with `cmd/redis-cache -cache=redis`; do not run one cache
refresher per `unify` node. Run `cmd/ledger` only on the log aggregation node
where the complete `log_winloss/winloss.<stamp>` stream is available.
Mutating `cmd/redis-cache`, `cmd/ledger`, `cmd/mid-callback-retry`, and
`cmd/winloss` executions also acquire renewable Redis singleton leases and
report an error if renewal is lost. `ledger_log.timely` and `daily_log.daily`
are unique database backstops; no job may rely on a lease as its only
idempotency boundary.
All mutating `cmd/redis-cache` modes share the `aofei:redis-cache` lock; a
partial route or spread run therefore cannot overlap a full generation build.

Checked-in `etc/aofei.json` and `etc/summer.example.json` are examples. Generated
`etc/*.local.json` files are local-only artifacts and must not be copied into
production as-is.

`hosted_payments` is independently disabled by default. Before enabling it,
apply the reviewed six-table/twelve-trigger migration, enable S02 identity and
the exact `payment.*` permissions/recent-MFA policy, provision separate
API/current-webhook/previous-webhook environment references, preserve the raw
body on exact `POST /webhooks/stripe`, disable proxy caching/challenges for that
path, and pass the test-mode matrix. Live mode additionally requires named
legal, finance, tax, risk, privacy, support, and incident approval. See
[hosted-funding-payout.md](hosted-funding-payout.md); deployment of code alone
does not authorize provider/account changes or money movement.

Production secrets and live connection values are injected by deployment
tooling or root-owned config/environment files. Do not commit database
passwords, Redis credentials, SMTP credentials, session secrets, OAuth secrets,
tracking secrets, or cloud keys. DSP tracking URLs use `tracking_secret` in
`AOFEI`, or the `TRACKING_SECRET` environment fallback, to sign click redirect
win/loss, middleman callback, and cap-mutation tracker payloads. Set
`tracking_signature_ttl_seconds` to bound callback replay; the default is
86400 seconds. Set `cap_state_ttl_seconds` to bound idle Redis frequency-cap
state; the default is 7776000 seconds (90 days), and refreshes never shorten a
longer existing key TTL.
When the later P03 tag-integration and rollout work authorizes enabling
`direct_ssp_tokens`, provision each `key_env` as an owner-only deployment
secret containing a base64 or hexadecimal encoding of an exact 32-byte key.
Keep current and previous keys distinct, consistent across accepting nodes, and
out of config, logs, cache payloads, database rows, tags, and publisher-visible
output. The current default-off codec does not by itself authorize production
activation.
The separate default-off `direct_ssp_auth` block controls publisher/App request
credentials. Before a later P03 rollout authorizes it, apply the additive
`pub_request_credential` migration, enable S02 identity with exact
`publisher.credential.read|issue|rotate|revoke` permissions and recent MFA for
mutations, and prove MySQL/Redis availability on every HTTP node. The defaults
allow 300 seconds of request clock skew, refresh the immutable public-key
snapshot every 30 seconds, fail it closed after 120 seconds, and cap requested
rotation overlap at one day. No deployment signing secret is required:
issuance returns the Ed25519 private seed once to the authorized publisher,
while MySQL retains only the public verifier. The caller must keep that private
value in an approved secret manager and generate a new nonce per request; never
place it in configuration samples, logs, Redis, backups, browser tags, or
mobile source. Do not enable this gate until the remaining P03 client-claim,
sample/cache integration, and canary/rollback rows are complete.
Frequency-cap payload version 2 is rolling-compatible: upgrade readers/writers
normally, monitor cap decode/refresh errors, and keep the rollout bounded to the
callback TTL. New workers read legacy-only values and version-2 values; old
workers read the saturated legacy prefix of version 2. Touched legacy entries
upgrade in place, so do not scan or delete `bothcap:*`. During rollback, retain
the version-2 values; the previous worker uses their prefix and a later upgraded
worker recovers the authoritative UTC trailer.
Use `aofei_bothcap_formats_total` to observe legacy, `utc_v2`, and malformed
reads during the rollout. A growing malformed count blocks rollout; it does not
authorize deleting user cap hashes.
Roll the middleman `/mid/*` callback tier consistently when introducing or
removing split `middleman:notify:*` and `middleman:publish:*` ownership. Preserve
notify, publish, bill, callback, and retry state in both directions; deleting
those keys can replay a downstream side effect. Monitor the fixed-key
`aofei_middleman_callback_outcomes_total` for retry, duplicate, and claim-release
changes before widening traffic. Treat `/mid/*` `503` responses as retryable
Redis, local-publication, or unavailable durable-forward dependencies; missing,
expired, malformed, or corrupt callback state remains `400`.
Set `delivery_cache_max_age_seconds` (default 900) as the hard maximum age for
compiled budget/schedule policy, schedule full cache publication at least every
one-third of it, and keep `delivery_reservation_ttl_seconds` long enough for the
tracking TTL plus five-minute skew (default 86700). The default
`delivery_state_ttl_seconds` is 172800; total delivery counters are persistent,
while this value provides the daily-state reconciliation grace period.
An expired reservation token never refunds its persistent total-budget counter.
If missing callbacks cause suspected conservative underdelivery, pause the
affected demand, preserve the Redis value and request/response/measurement
evidence, run the normal ledger jobs, and compare the reconciled
`adv_balance.current_*` floors with the exact `delivery:budget:total:<balance_id>`
keys. Change or delete an exact key only under an approved incident procedure
after accounting owns the discrepancy; never scan-delete the family or infer a
refund solely from reservation expiry.
Alert on increases in `aofei_tracking_replay_redis_errors_total` and
`aofei_tracking_replay_unkeyed_total`; those events are accepted fail-open and
can therefore retain at-least-once measurement behavior until Redis or event
identity is corrected. Alert on
`aofei_tracking_cap_update_fail_open_total` to detect valid measurement events
published while their cap update could not be completed. Replay, cap-event, and
win/loss markers expire at the signed timestamp's validity deadline, which may
be up to the configured TTL plus the accepted five-minute future skew from the
receiving worker's current time.
For budgeted local callbacks, a `503` can also mean that publication completed
but idempotent delivery state is still pending or its completion marker is
uncertain. Preserve the replay and reservation keys: completed retries do not
republish, and processing retries remain retryable until the short lease
resolves.

Public advertiser and publisher registration/password-retrieval actions require
a complete Summer `Blks._gmail` configuration. W8M uses Gmail API
`users.messages.send`: set `Transport` to `gmail-api`, keep only non-secret
optional `From` and `Reply-To` metadata in JSON, and inject
`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REFRESH_TOKEN` through
an owner-only deployment environment. The service verifies that Google accepts
the refresh token before account mutation. Removing `_gmail` is the supported
emergency email-disable control: those submissions fail before account mutation
with a maintenance error, while login and authenticated portals remain
available. Revoke exposed OAuth credentials before considering a replacement;
never preserve the old values beside the active configuration or in Git.

For a user service, place the OAuth assignments in an owner-only file such as
`%h/.config/aofei/gmail.env`, add
`EnvironmentFile=%h/.config/aofei/gmail.env` to the service, and keep the file
at mode `0600`. A production mail block can then contain only:

```json
"_gmail": {
  "Transport": "gmail-api",
  "Reply-To": "support@w8m.com"
}
```

Before enabling the block, exchange the refresh token against Google's token
endpoint and require a successful bearer token response containing
`https://www.googleapis.com/auth/gmail.send` (or a broader Gmail compose/modify
scope). The sibling `~/Workspace/udon` checkout can renew it without printing
the token or writing it to source files:

```bash
install -d -m 0700 "$HOME/.config/aofei"
cd ~/Workspace/udon
GOWORK=off go run ./cmd/udon oauth google login \
  --client-id "$GOOGLE_CLIENT_ID" \
  --scope https://www.googleapis.com/auth/gmail.send \
  --listen 127.0.0.1:8765 \
  --output "$HOME/.config/aofei/google-oauth.hcl"
```

The command uses the loopback callback
`http://127.0.0.1:8765/oauth2/callback` with a Google Desktop OAuth client,
prints only the consent URL, and creates the requested private HCL file with
mode `0600`; it does not print token values. Forward local port `8765` to the
remote loopback address before opening the URL. Move the returned refresh token
directly from that private file into the owner-only service environment, then
securely remove the intermediate file. Do not retain the token in shell
history, command transcripts, JSON, or Git. A missing, wrong-scope, or rejected
credential must leave `_gmail` disabled so registration stays fail-closed.

Production registration and password-recovery mail must also enable S06 public
account abuse protection. Configure one managed Cloudflare Turnstile widget for
`w8m.com` and `www.w8m.com`, inject its site/secret keys plus exact hostname and
trusted-proxy lists through a separate owner-only `0600` service environment,
and set `PUBLIC_ACCOUNT_PROTECTION_ENABLED=true`. `cmd/unify` then fails startup
on partial configuration. Siteverify runs before password hashing, Redis,
Google, database, or mail; one atomic Redis operation applies pseudonymous IP,
email, and global quotas before Gmail/model work. Add and canary the matching
Cloudflare zone rate-limit rule, inspect it after creation, and alert on the
four `aofei_public_account_*` metric families. Complete activation, validation,
rotation, proxy-list maintenance, and safe mail-first rollback steps are in
[public-account-abuse-protection.md](public-account-abuse-protection.md).

Summer identity hardening is opt-in. Before setting `Identity.Enabled=true`,
apply the six-table/two-trigger S02 migration, provision the same base64- or
hex-encoded 32-byte key named by `Identity.KeyEnv` on every `unify` node, review
all role `Permissions`, preserve `RequireGrant=true` for analysts, verify SMTP
recovery, and canary TOTP enrollment/recovery. The key value belongs only in a
deployment secret environment, never in JSON. Enabled identity adds opaque
database sessions, POST+CSRF logout, TOTP/recovery, resource permissions, and
immutable security evidence. Full rollout, command, retention, key-loss, and
rollback procedures are in
[identity-access-security.md](identity-access-security.md).

The external advertiser management API is independently opt-in. Do not enable
`management_api.enabled` until S02 identity is enabled, the I03 four-table and
version-trigger migration is applied, and the same random 32-byte key named by
`management_api.key_env` is present on every `unify` node. Issue its least-
privilege token only from the recently MFA-authenticated advertiser/admin
portal; the token is shown once and never belongs in JSON, logs, tickets, or
shell history. Review the per-credential/account quotas, 256 KiB default body
limit, five-second timeout, and cache-activation deadline independently from
auction traffic. Canary and rollback steps are in
[advertiser-management-api.md](advertiser-management-api.md).
Management API audit retention uses the same restricted binary with
`-action=prune-api-audit`, a named administrator id, a bounded limit, and a
single-line reason. Run it only with the separate maintenance database
configuration described by S02; never grant the HTTP principal audit deletion.

Traffic quality is independently opt-in. Before setting
`traffic_quality.enabled=true`, apply the S03 nine-table/ten-trigger migration,
enable S02 identity, grant the exact `quality.*` permissions, and provision a
unique base64- or hex-encoded key of at least 32 bytes in the environment named
by `traffic_quality.digest_key_env` on every HTTP and maintenance node. Never
store the key value in JSON. The default enforcement refresh is 30 seconds and
the default maximum snapshot age is 120 seconds. Startup is strict when
enabled; missing schema/key or an initial snapshot failure stops the service.
Subsequent refresh errors retain only the last unexpired snapshot, after which
serving fails open. Create rules as Draft, pass through Observe and Canary, and
activate blocking rules only after reviewed complete canary evidence is within
the false-positive limit. Full rollout, retention, billing, incident, and
rollback steps are in
[traffic-quality-anti-fraud.md](traffic-quality-anti-fraud.md).

Middleman fallback is disabled unless `middleman_enabled` is true in `AOFEI`.
When enabled, set `middleman_exchange_domain`, `middleman_timeout_ms`, and
`middleman_max_bidders_per_imp` deliberately. Set
`middleman_route_cache_ttl_ms` for the worker-side decoded route snapshot; the
default is 5000 ms, and refresh errors disable fanout until the short error
cache expires. Shared route loads use `middleman_timeout_ms` independently of
the initiating request, so client cancellation does not cancel refreshes needed
by other workers. `trigger_mode='Always'` route
fanout is ignored unless `middleman_always_enabled` is also true. Middleman
callback proxying also requires `tracking_secret`, Redis, and a public
`middleman_callback_base_url` that points back to the `cmd/unify` HTTP service; set
`middleman_callback_ttl_seconds` and `middleman_callback_timeout_ms` according
to exchange callback latency expectations. Callback TTL must cover the tracking
signature TTL plus the accepted five-minute future skew and the processing
lease; the 24-hour signature default therefore uses an 86,700-second callback
TTL. Callback timeout must be within 1..60,000 ms. Downstream callback URLs are rejected
when they resolve to loopback, private, link-local, unspecified, multicast, or
rebinding targets. Each active bidder
`credential_ref` names an environment variable visible to `cmd/unify`; its
value must be a JSON object of outbound HTTP headers. Do not put those header
values in MySQL, Redis, or checked-in config files.

Approve partner endpoints only as exact OpenRTB 2.5 profiles. Keep encoded and
decoded public request limits explicit with `traffic_default.max_body_bytes`
and `traffic_default.max_decompressed_body_bytes`; partner overrides inherit
either omitted value. Leave `openrtb_debug_enabled=false` normally. For a
time-bounded incident sample, enable it with a rate in `(0,1]` (normally
`0.01`), confirm diagnostics contain only hashed request identity and fixed
metadata, then disable it again. Raw bid-body capture is not supported.

Middleman reporting depends on `cmd/ledger` running on the node with the
complete `log_winloss` stream. Advertiser middleman reports use pay-side spend
from `daily_mid`; admin settlement reports use charge, pay, and margin from
`daily_mid`.

Operators manage middleman route groups, route bidders, and traffic targets in
Summer at `/goto/admin/g/midroute?action=topics`. Route edits update MySQL only;
run the singleton `cmd/redis-cache -cache=redis` or `-cache=all` refresh on the
dedicated cache node before expecting `cmd/unify` workers to use the new route
state.
After publication, run `cmd/redis-cache -validate-middleman
-activation-stage=preflight` from each canary node's exact service environment.
It fails on a stale/legacy route generation, invalid active profile, incomplete
callback/signing config, or missing/unsafe credential header map without
printing values. Fallback and Always activation, evidence, rotation,
disablement, and rollback follow
[middleman-activation.md](middleman-activation.md); do not enable production
traffic from the health page alone.

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
headers, but any supplied `Origin` or `Referer` must also match. When
`direct_ssp_auth` is enabled, SDK omission is accepted only with the App-scoped,
body-bound, fresh proof and shared Redis replay claim; generic `401` denotes
proof rejection and generic `503` denotes unavailable or stale verifier/replay
state. Browser CORS and cookie behavior remain unchanged. The `aofei_pz_uid`
cookie is browser-only; SDK/in-app requests do not read or set it
and never join IP/User-Agent as a fallback identity. `/pz` uses
`RemoteAddr` for OpenRTB `device.ip` unless the peer address is in
`trusted_proxy_cidrs`; only trusted proxies may supply `X-Forwarded-For` or
`X-Real-IP`.

Commercial publisher activation uses the read-only
`cmd/redis-cache -validate-publishers` gate and the cache-first rolling order in
[publisher-activation.md](publisher-activation.md). P01 adds Web/App type and
server-owned USD CPM floor fields to existing publisher payloads. Publish and
inspect a complete new cache generation before rolling P01 HTTP workers; those
workers intentionally reject pre-P01 entries with missing commercial metadata.

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
Description=Run Aofei Redis cache refresh every 5 minutes

[Timer]
OnBootSec=2m
OnUnitActiveSec=5m
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

Example action attribution reconciliation and retention timers on one
operations node:

```ini
[Unit]
Description=Aofei analytical action attribution reconciliation

[Service]
Type=oneshot
User=aofei
Group=aofei
Environment=AOFEI=/etc/aofei/aofei.json
ExecStart=/opt/aofei/bin/action-measurement -action=reconcile -limit=1000
NoNewPrivileges=true
PrivateTmp=true
```

```ini
[Unit]
Description=Run Aofei action reconciliation every 5 minutes

[Timer]
OnBootSec=3m
OnUnitActiveSec=5m
Unit=aofei-action-reconcile.service

[Install]
WantedBy=timers.target
```

```ini
[Unit]
Description=Aofei expired analytical action pruning

[Service]
Type=oneshot
User=aofei
Group=aofei
Environment=AOFEI=/etc/aofei/aofei.json
ExecStart=/opt/aofei/bin/action-measurement -action=prune -limit=10000
NoNewPrivileges=true
PrivateTmp=true
```

```ini
[Unit]
Description=Run Aofei action retention daily

[Timer]
OnCalendar=*-*-* 01:20:00
Persistent=true
Unit=aofei-action-prune.service

[Install]
WantedBy=timers.target
```

Use the same `User`, `Group`, `Environment`, restart policy, and hardening
pattern for `spread`, changing only `ExecStart`. `redis-cache` and `ledger`
should normally be oneshot timer services, not long-running services.

`cmd/unify` handles `SIGINT` and `SIGTERM` through a signal-aware context. It
stops accepting new HTTP work, allows up to 15 seconds for in-flight handlers,
then closes the Aofei controller so the bounded audit publisher drains before
owned NATS, Redis, and MySQL connections close. If HTTP draining exceeds the
deadline, the server force-closes remaining connections and exits with an
error. Set the systemd stop timeout above 15 seconds so the application gets
its full drain window before the service manager escalates.

## Build And Rollout

Build artifacts from a reviewed commit:

```bash
GOWORK=off go test ./...
GOWORK=off go install ./cmd/accounting ./cmd/action-measurement \
  ./cmd/hosted-payment ./cmd/ledger ./cmd/maxmind \
  ./cmd/mid-callback-retry ./cmd/nats-client ./cmd/redis-cache \
  ./cmd/report-experiment ./cmd/spread ./cmd/traffic-quality
(cd ../pzdesign && GOWORK=off go test ./... && GOWORK=off go install ./cmd/unify ./cmd/identity-admin)
(cd ../genelet && GOWORK=off go test ./...)
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
sudo systemctl enable --now aofei-action-reconcile.timer
sudo systemctl enable --now aofei-action-prune.timer
sudo systemctl enable --now aofei-ledger.timer
sudo systemctl enable --now aofei-ledger-daily.timer
sudo systemctl status aofei-unify.service
```

Smoke checks:

- Every HTTP node returns 204 on `/healthz`; `/readyz` returns 204 only after
  initialization and becomes 503 before graceful drain. The load balancer uses
  `/readyz` without caching and keeps `/debug/vars` externally blocked.
- `cmd/unify` listens on `ServerPort` and serves expected admin/static paths.
- DSP bid endpoint returns the expected local or staging fixture response.
- Redis contains `pubmap`, `audience`, `creative`, `middleman:routes:v2`,
  fallback-only legacy `middleman:routes`, and `slot:<size_id>` cache families
  after cache population when Redis static-cache mode or middleman fallback is
  used.
- `cmd/redis-cache -cache=routes -read` shows current route-cache metadata when
  middleman routes are used.
- `cmd/redis-cache -validate-middleman -activation-stage=preflight` passes in
  the canary service environment; `fallback` or `always` passes only for the
  specifically approved stage.
- The middleman callback retry timer runs on one operations node and can read
  due rows without processing them via `cmd/mid-callback-retry -read`.
- In local/spread static-cache mode, spread files exist and bid nodes have
  loaded a current in-process static generation.
- NATS log subjects are written into the configured log directories.
- Ledger timers run only on the log aggregation node and see complete
  `log_winloss/winloss.<stamp>` files.
- No unexpected HTTP 403 appears for configured admin origins.
- When identity is enabled, canary TOTP enrollment/login, POST logout, public
  recovery with a one-time code, analyst delegated-report access, cross-account
  denial, analyst mutation denial, and immutable audit insertion all pass.

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
until the accounting contract/immutability and source counts are verified,
approved deletion cases are reapplied, `cmd/redis-cache` has repopulated runtime
cache, and the HTTP/admin smoke checks pass. Run the repository rehearsal with
`./scripts/aofei-recovery-drill.sh`; it uses only uniquely named disposable
containers and is not a production backup implementation. Production
encryption, RPO/RTO, restore order, topology, dependency semantics, and SLO
evidence are defined in
[single-region-availability.md](single-region-availability.md).

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
- Treat uploaded audience sets as raw advertiser-provided identifiers. The
  unified service installs `privacy_audience_ttl_seconds` on writes; verify TTL
  coverage and use the scoped deletion procedures in
  [privacy-data-governance.md](privacy-data-governance.md).

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

The unified HTTP service exposes stdlib expvar metrics at `/debug/vars`, but
only direct peers in `metrics_allowed_cidrs` are authorized (loopback by
default). The reverse proxy/Cloudflare route must also deny this path. Do not
trust `X-Forwarded-For` for scrape authorization. Full traffic-control, metric,
dependency, alert, and canary rules are in
[production-traffic-observability.md](production-traffic-observability.md).

The expvar surface includes bid/no-bid counters, audit queue depth/drops and publish errors,
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
Also alert on sustained `aofei_action_touch_errors_total`, action signature or
dependency rejection growth, and nonzero exits from the singleton action
reconcile/prune timers. Action failures remain separate from CPM ledger and
settlement health.

Privacy evidence is fixed-cardinality: monitor
`aofei_privacy_decisions_total`, `aofei_privacy_invalid_signals_total`, and
`aofei_privacy_middleman_blocked_total`. A sudden invalid-signal increase or a
decision-mode change is an integration incident; metrics never contain the
signal or identifier itself.

Traffic-quality evidence is also fixed-cardinality. Monitor decision/match and
five action counters, dependency errors, rollback, enforcement snapshot
refresh/error/evaluation, and serving throttle/reject/quarantine. Run the
restricted `cmd/traffic-quality -action=health -actor-admin-id=<id>
-since-hours=24` for per-rule false-positive state instead of adding rule or
account labels. Dependency/snapshot errors are availability incidents, not IVT;
roll a canary back when its reviewed false-positive limit is exceeded.

The proposed 99.9% auction objective is not a claim until a named rolling
30-day production window, independent probes, contracted-load denominator,
latency distributions, exclusions, and error-budget consumption are retained.
Use [single-region-availability.md](single-region-availability.md) for the exact
good/bad event and burn-alert contract.

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

`cmd/nats-client` prunes its four generated subject file families at startup
and rotation using `privacy_log_retention_hours` (default 168). It ignores
unrelated files and symlinks. Keep the files on encrypted persistent storage;
backup generations need a separately approved expiry/deletion policy because
online pruning cannot remove an existing backup.

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
reset before production use. The S02 compatibility switch
`Legacy_password_upgrade` accepts only an exact legacy plaintext match and
atomically replaces it with bcrypt; use it for a measured migration window and
disable it after an audit confirms no plaintext credentials remain. Each direct SQL issuer must also define `OutPars`
in the exact order of its selected columns, including the `passwd` column;
otherwise Genelet cannot scan and verify the stored hash reliably.

With identity enabled, password login is not sufficient by itself: the opaque
database session, required TOTP state, named action permission, resource scope,
and sensitive-action reauthentication are also enforced. Do not use
`Identity.Enabled=false` as a routine incident workaround because it restores
the lower-assurance legacy cookie boundary.

## Historical Material

[legacy-operations.md](legacy-operations.md) is historical-only. `backup/`
contains policy only; old runtime artifacts are intentionally absent from Git.
Use them only as context when interpreting old deployments; do not use them as
active setup instructions or as sources for credentials.
