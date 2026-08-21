# Tech Stack

## Languages And Module

- Language: Go
- Main module: `github.com/guruperl/aofei`
- Sibling admin/design module: `github.com/guruperl/pzdesign`
- `go.mod`: Go 1.22 with toolchain 1.23.5

Use `GOWORK=off` for local commands from this repository. The parent `go.work`
currently does not include this module path, and plain `go list ./...` fails
from this checkout under that parent workspace.

## Multi-Milestone Goal Execution

[GOAL.md](../GOAL.md) defines the slash-goal execution protocol for ordered
status files. The caller supplies a linear `STATUS_ORDER`; optional
`DOWNSTREAM_IMPACTS` identify pending plans to reconcile after each completed
milestone. Commit and external-mutation authority remain explicit and default
to none. Its bounded review-fix gate persists review iterations, requires a new
full review after every P1/P2-or-higher fix, and stops incomplete after iteration
10 if a blocking finding remains. `memory-bank/suggested.txt`, when present, is
disposable launch input for `$memory-bank:memory-bank-goal`; it must be
reconciled and deleted after launch or when stale. `memory-bank/milestone.md`
and the lane status files remain the roadmap and task sources of truth.

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
`tracking_secret` in the DSP config signs generated `/imp`, `/clk`, `/win`,
`/loss`, and `/mid/*` callback URLs; when omitted, `TRACKING_SECRET` is used as
the fallback. `tracking_signature_ttl_seconds` bounds signed URL replay and
defaults to 86400. `cap_state_ttl_seconds` bounds idle `bothcap:<user_id>`
retention and defaults to 7776000 seconds (90 days); cap refresh does not
shorten a longer existing TTL.
`trusted_proxy_cidrs` is empty by default. `/pz` uses `RemoteAddr` for OpenRTB
`device.ip` unless the peer address matches an explicit proxy IP or CIDR in
that list, in which case it accepts `X-Forwarded-For` or `X-Real-IP`.
Middleman fallback is controlled by `middleman_enabled`,
`middleman_timeout_ms`, `middleman_max_bidders_per_imp`, and
`middleman_exchange_domain`. `middleman_route_cache_ttl_ms` defaults to 5000
and bounds each worker's immutable decoded route snapshot; concurrent misses
share one refresh and expired routes are not used after refresh failure. The
shared load uses an independent deadline derived from `middleman_timeout_ms`;
callers wait with their own contexts and cannot cancel the load for one another.
`trigger_mode='Always'` fanout also requires
`middleman_always_enabled`; the default is false. Middleman callback proxying is
controlled by `middleman_callback_ttl_seconds`, `middleman_callback_timeout_ms`,
and `middleman_callback_base_url`; it requires `tracking_secret` and Redis.
Bidder `credential_ref` values name environment variables containing JSON
outbound header maps for downstream OpenRTB calls.

OpenRTB partner interoperability uses exact version `2.5`. Public auction
traffic accepts only identity or one gzip content coding; traffic policies set
both `max_body_bytes` and `max_decompressed_body_bytes` (each capped at 1 MiB).
Successful JSON response gzip is negotiated through `Accept-Encoding`; 204 is
never compressed. `openrtb_debug_enabled` is false by default and
`openrtb_debug_sample_rate` must be in `(0,1]` when enabled (default `0.01`).
Sampled records contain hashed request identity and fixed metadata only.
Credential-free partner fixtures are under `dsp/testdata/openrtb/`.

R01 action configuration defaults are `action_token_ttl_seconds=2592000`,
`action_click_window_hours=720`, `action_view_window_hours=168`,
`action_max_age_hours=2160`, `action_request_skew_seconds=300`, and
`action_retention_hours=2160`; validation requires retention to cover maximum
accepted age. The HTTP
surface is `POST /action`; MySQL tables are `measurement_touch` and
`measurement_action`; maintenance is `cmd/action-measurement`. Durable insert
failure returns retryable 503, while touch failure remains measurement
fail-open. The contract and exact-body HMAC input are documented in
[docs/conversion-attribution.md](../docs/conversion-attribution.md).

S01 privacy defaults keep all traffic contextual unless an applicable signal
and explicitly configured contract authorize more. `privacy_tcf_vendor_id=0`
disables personalized processing; a configured id also requires the current
service-specific TCF policy version, every `privacy_tcf_purpose_ids` grant, a
disclosed-vendors segment, and `tracking_secret`. Browser identity cookies,
privacy-safe interval logs, and uploaded-audience sets default to 30 days,
168 hours, and 30 days through `privacy_browser_id_ttl_seconds`,
`privacy_log_retention_hours`, and `privacy_audience_ttl_seconds`.
`privacy_contextual_middleman_enabled` is a second, default-false disclosure
gate in addition to `middleman_enabled`; outbound bidder requests are always
independently contextualized. The complete contract is
[docs/privacy-data-governance.md](../docs/privacy-data-governance.md).

S03 `traffic_quality` is disabled by default. When enabled,
`digest_key_env` names a base64/hex deployment key that decodes to at least 32
bytes; the value never appears in JSON. Enforcement refresh/max-age defaults
are 30/120 seconds. The Go domain is `trafficquality`, the maintenance/ingest
surface is `cmd/traffic-quality`, and the Summer review UI is
`../pzdesign/summer/trafficquality`. Full contracts are in
[docs/traffic-quality-anti-fraud.md](../docs/traffic-quality-anti-fraud.md).

O01 auction admission uses `traffic_default` plus up to 256 exact
`traffic_partners` entries (`adx:<domain>` or `ssp`) for QPS, burst,
concurrency, timeout, and body limits. `metrics_allowed_cidrs` defaults to
loopback and matches only the direct peer. Metric names, capacity commands,
dependency probes, alert thresholds, and rollout rules are in
[docs/production-traffic-observability.md](../docs/production-traffic-observability.md).

Summer/Genelet admin tests must use `SUMMER`; the Genelet config format uses
upper-case keys such as `ConnectArray`, `Template`, and `UploadDir`.
The checked-in Summer config includes `admin`, `adv`, `pub`, and `agent` roles.
Middleman bidder endpoints use the existing `adv` role through the `adv_bidder`
module. Summer/Genelet code lives in the sibling `../pzdesign` checkout together
with HTML templates under `../pzdesign/tmpls` and static UI assets under
`../pzdesign/www`. Generated local Summer config points `ProjectRoot`,
`Template`, and `DocumentRoot` at that checkout.

Production defaults are `/etc/aofei/aofei.json` and
`/etc/aofei/summer.json`, passed through `AOFEI` and `SUMMER`. The checked-in
Summer example is `etc/summer.example.json`. The production
runbook is [docs/production-runbook.md](../docs/production-runbook.md).
Summer/Genelet CORS allows the exact `ServerURL` origin plus exact entries in
`CORSOrigins`.
Genelet framework contracts are documented in
`../pzdesign/docs/genelet-manual.md`; Summer admin module and cache-side-effect
conventions are documented in `../pzdesign/docs/summer-ui-structure.md`.
Rendering entrypoints, contextual-escaping rules, the single fixed CSRF
trusted-HTML boundary, local asset policy, and hostile-input checks are
documented in `../pzdesign/docs/rendering-security.md` and
[docs/template-rendering-security.md](../docs/template-rendering-security.md).

## Schema Baseline Commands

`etc/step4_init.sql` is the active schema/reference-catalog contract;
`etc/demand.sql` owns deterministic synthetic account and bid-path fixtures.
The local helper keeps schema stewardship commands under the same Docker
workflow:

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

A01 accounting uses MySQL `DECIMAL(20,6)` and Go integer micro-dollar values
for statement mutations. `cmd/accounting` is the authorized manual operator
surface for statement creation, adjustment, approval, settlement, correction,
reconciliation, and CSV export; its complete contract and populated-system
migration are in [docs/accounting-settlement.md](../docs/accounting-settlement.md).

A02 hosted payments use the standard-library HTTP client through the
`hostedpayment` Stripe adapter; no Stripe SDK or card/bank field enters this
module. `hosted_payments` is disabled by default. It names separate API/current
webhook/previous-webhook environment references, exact Stripe/public HTTPS
origins, pinned Stripe API version `2024-06-20`, bounded body/time/retry policy,
and retention/reconciliation limits. Mode-matched `rk_test_`/`rk_live_`
restricted keys are preferred and accepted alongside secret keys; the
maintenance command never resolves either API or webhook secret values. Real
signed events preserve linked opaque
objects; authorized reconciliation retrieves the exact linked Balance
Transaction because Stripe has no per-transaction availability webhook.
`cmd/unify` owns the signed webhook and Summer UI; `cmd/hosted-payment` exposes
aggregate health and bounded event retention but cannot move money. See
[docs/hosted-funding-payout.md](../docs/hosted-funding-payout.md).

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

Run the DB-only publisher commercial-readiness gate before publication:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -validate-publishers
```

It performs no Redis operation and prints deterministic Web/App, size, USD CPM
floor, and packed-token evidence. P01 cache/binary ordering and rollback are in
`docs/publisher-activation.md`.

Validate middleman activation from a canary node's exact service environment:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -validate-middleman \
  -activation-stage=preflight
```

This read-only mode compares current MySQL routes with the published Redis v2
checksum/high-water, validates bounded profiles and config, and resolves
environment header references without printing their values. `fallback` and
`always` add the corresponding runtime/privacy gate and route requirements;
the staged evidence/rollback contract is in `docs/middleman-activation.md`.

The D01 delivery snapshot default maximum age is 900 seconds. Schedule this
full refresh (or `-cache=all` for spread/local deployments) at least every 300
seconds. `-cache=routes` does not refresh campaign delivery policy.

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

Run Redis cache refresh from one dedicated node only, normally through cron or a
systemd timer:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis
```

Refresh only the Redis middleman route cache:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes
```

Read only the Redis middleman route cache:

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=routes -read
```

Do not run a Redis cache refresher from every `unify` node. When spread mode is
used, keep `cmd/spread` running on nodes whose local disk snapshots should be
updated from NATS messages.

Expected Redis cache families are `pubmap`, additive direct-SSP
`pubmap:by-id`, `audience`, `creative`, `middleman:routes:v2`,
fallback-only legacy `middleman:routes`, and
`slot:<size_id>` hashes keyed by slot id. Expected spread directories are
`.local/spread/pubmap/`, `.local/spread/audience/`,
`.local/spread/creative/`, and `.local/spread/slot/<size_id>/`.
Middleman route caches are Redis-only and are populated by the singleton
`cmd/redis-cache` job, not by `cmd/unify`.
Full Redis refreshes build internal `:next` shadows and install all live
families with one transaction; live key names and payloads are unchanged.
Direct SSP local/static serving derives its by-publisher-id lookup in memory
from `.local/spread/pubmap/`; it does not add a separate spread directory. The
direct cache includes cache-owned Web/App type, slot-size, and USD CPM floor;
`/pz` rejects mismatched platform/type/size/media and uses the greater of the
configured and finite non-negative request floors. Inactive publisher sites
and slots are omitted. P01 workers reject older payloads missing type/floor.
`POST /pz` is served by `dsp.Controller.ServeSSP` through
`../pzdesign/cmd/unify`; valid omitted or `responseFormat:"html"` requests
return `200 application/json` arrays of HTML strings. `responseFormat:"json"`
returns ordered fill/no-fill objects with markup, tracker URLs, price/currency,
ids, dimensions, and parsed native payloads when applicable.
`responseFormat:"openrtb"` returns an OpenRTB `BidResponse`, including `200`
with an empty `seatbid` on all-no-fill. Malformed JSON, unsupported response
formats, invalid direct tokens, missing slots, unsupported media, and cache
validation failures return HTTP errors.
When `middleman_enabled` is true, validated `/pz` auctions use the same
middleman route/runtime controls as ADX after local matching. Local no-fills can
fan out through `Fallback` routes, and local fills can fan out through `Always`
routes only when `middleman_always_enabled` is also true. Middleman SSP fanout
uses the synthesized internal OpenRTB request. Middleman JSON responses omit
local-only `impressionUrl` and `clickUrl`; OpenRTB responses group bids by the
final winner seat.
Every `adUnits[].code` is required, bounded, URL/DOM-safe, and unique. Invalid
codes return `400` before cache mutation, cookies, bidding, middleman fanout, or
audit publication. Each ad unit must declare exactly one matching media type.
`../pzdesign/cmd/unify` also handles `OPTIONS /pz` and applies permissive CORS
headers only on `/pz`: origin `*`, methods `POST, OPTIONS`, and header
`Content-Type`. Publisher slot pages load `../pzdesign/www/js/ads.js`; the
script posts to the origin it was loaded from plus `/pz` unless
`pzLoadAds(payload, {endpoint: "..."})` is used. Filled markup is installed in
an opaque-origin sandboxed `srcdoc` iframe; the target records deterministic
`filled`, `no-fill`, or `error` state and no-fill clears prior content.
`/pz` remains credentialless at the CORS layer. Browser traffic may read or set
`aofei_pz_uid` only after the centralized privacy decision accepts a configured
personalization grant. Missing, denied, opt-out, invalid, COPPA, and currently
unmapped GPP signals do not read or set the cookie and do not use IP+UA as an
identity fallback. `platform:"sdk"` requests are always cookie-free.
After packed token and cache validation, `POST /pz` enforces browser
origin/referrer policy. Browser requests, including missing or empty `platform`,
must include a valid `Origin` or `Referer` whose host exactly matches the cached
site host, and any present `Origin` or `Referer` must match. `platform:"sdk"`
may omit both headers, but supplied headers must still match. Rejections return
`403` before cookies, bidding, or audit publishing and increment
`aofei_ssp_policy_rejections_total`.
SDK/in-app `/pz` requests may include body `app`, `device`, `user`, and `regs`
objects.
They synthesize `BidRequest.App` and leave `BidRequest.Site` nil. The validated
cached site string becomes authoritative app id/bundle/domain, and mismatched
supplied app identity fields return `400`. Body `device` identity fields feed
the existing attribute identity precedence; request IP/UA may inform contextual
request attributes but are never joined as a fallback identity. Body identity is used transiently
for local matching only when the privacy decision is personalized; otherwise
it is removed. Accepted identity is HMAC-pseudonymized before cap keys and
tracking payloads.
SSP request/response audit logs are JSON envelopes with `source:"ssp"` and
`contract:"pz-v1"`. ADX keeps its OpenRTB envelope shape, but both sources are
privacy-scrubbed before NATS. Attribute logs remove identity and precise
demographic/location facts and add `source`, `contract`, `privacy_mode`, and
`privacy_reason`.
`cmd/unify -local` is an explicit override: when omitted, the Aofei config's
`is_local` value is preserved; when enabled, local static snapshots are loaded
before serving requests and reloaded in the background at one-third of the
tightest configured cache-age bound. `local_cache_max_age_seconds` is an alert-only
freshness threshold for local/static mode. Loaded local snapshots publish
`aofei_local_cache_loaded_at_unix`; scrape-time
`aofei_local_cache_age_seconds` and `aofei_local_cache_stale` continue to update
while traffic is idle. General static snapshots do not fail closed by this age
alone, but RAdv delivery policy has a separate enforced
`delivery_cache_max_age_seconds` and stale policy candidates stop bidding.

## Operational Commands

Active local operational command contracts are documented in
`docs/operational-commands.md`.

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/nats-client -interval=10
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/spread
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/ledger -interval=10
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/ledger -daily -timestamp=YYYY-MM-DD
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/report-experiment -action=list
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/traffic-quality -action=health -actor-admin-id=42 -since-hours=24
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/mid-callback-retry -limit=100 -max-attempts=5 -timeout=2s
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/mid-callback-retry -read -json
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/winloss --bid=/bid/default win
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go run ./cmd/maxmind -city=/path/to/GeoLite2-City.mmdb
./scripts/aofei-recovery-drill.sh
./scripts/aofei-reporting-benchmark.sh
```

Run ledger from the node where `cmd/nats-client` aggregates log files. Do not
run ledger on every `unify` node. When middleman callback metadata is present,
ledger also fills `ledger_mid` and `daily_mid`; advertiser reports use pay-side
spend, while admin reports expose charge, pay, and margin.
Run `cmd/mid-callback-retry` as a singleton operations job for retryable
downstream middleman callback forwarding failures. It forwards downstream only
and does not republish win/loss records. Its output includes due and stale
processing backlog counts for operator alerting; use `-json` for stable
automation fields.
Mutating cache, ledger, callback-retry, and simulator commands use renewable
token-owned Redis leases. Lease loss is a failed run, and MySQL uniqueness or
worker idempotency remains required because Redis cannot prove ownership across
every partition. `scripts/aofei-recovery-drill.sh` uses only uniquely named
disposable MySQL/Redis containers to checksum and restore the complete schema,
A01/R01/R02 evidence, immutable triggers, ledger/report identities, experiment
outcome retention/erasure, and derived cache. `scripts/aofei-reporting-benchmark.sh` similarly
uses a disposable MySQL 8.0.41 container and synthetic interval facts. The
2026-08-01 P02-expanded 100,000-row/five-run baseline measured advertiser
100/119 ms, publisher 105/118 ms, and operator 1684/1830 ms median/max on
x86-64 with eight visible CPUs. This is local query evidence, not production p95/p99 or an
availability SLO; OLAP reconsideration uses the production triggers documented
in `docs/marketplace-analytics-experiments.md`.

Generated log directories are `.local/logs/log_request/`,
`.local/logs/log_response/`, `.local/logs/log_attribute/`, and
`.local/logs/log_winloss/`. `cmd/nats-client` creates or tightens these
directories to `0750` and generated interval files to `0640`; ledger input logs
should not be world-readable or group/world-writable. `cmd/maxmind` reads MySQL
country/state tables and atomically writes the configured MaxMind JSON path,
normally `etc/maxmind.json`.

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

Package gate:

```bash
GOWORK=off go test ./...
```

`GOWORK=off go list ./...` should not include `github.com/guruperl/aofei/backup`;
`backup/` is policy-only; historical Go helpers and operational snapshots are
not retained in Git.

Runtime hardening checks:

```bash
GOWORK=off go vet ./...
GOWORK=off staticcheck ./...
GOWORK=off go test -race ./hostedpayment ./trafficquality ./dsp ./match ./internal/cmdboot ./internal/jobs/midcallback ./internal/jobs/cache ./internal/jobs/ledger ./internal/jobs/action ./cmd/spread ./cmd/nats-client ./cmd/action-measurement ./cmd/hosted-payment ./cmd/traffic-quality
GOWORK=off go test ./dsp ./match -run '^$' -bench . -benchmem
./scripts/aofei-doc-check.sh
git diff --check
```

The documentation guard requires `docs/README.md` to index every active
Markdown guide and every zero-padded A/D/I/O/P/R/S status file. It verifies the
completed original lane set, the planned D04/P03/S05/O03/R03/A03 remediation
horizon, demand-gated I02, and the bounded `GOAL.md` review-fix contract; checks
the 94/0/6/55 schema inventory; and rejects attempts to test the removed
`./genelet` package path from inside pzdesign. Historical status and
legacy-operation evidence may retain commands and counts that were accurate at
their recorded closeout.

Integration and smoke checks are explicit command families rather than hidden
package-test requirements:

```bash
./scripts/aofei-cache-smoke.sh
./scripts/aofei-recovery-drill.sh
./scripts/aofei-reporting-benchmark.sh
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" go test ./dsp -run 'Test.*Smoke'
(cd ../pzdesign && GOWORK=off go test ./cmd/unify)
(cd ../pzdesign && GOWORK=off staticcheck -checks=all,-ST1000,-ST1003,-ST1006 ./...)
```

Admin compatibility verification:

```bash
./scripts/aofei-local.sh reset-sample
(cd ../pzdesign && GOWORK=off SUMMER="$PWD/../aofei/etc/summer.local.json" go test ./...)
(cd ../genelet && GOWORK=off go test ./...)
```

S02 identity verification also runs the external Genelet suite and the full
Pzdesign/template gates. `../pzdesign/cmd/identity-admin` is the restricted
operator CLI for analyst creation, exact grant/revoke, TOTP reset, and bounded
security-audit retention. It reads `SUMMER`, the environment key named by
`Identity.KeyEnv`, and new passwords only from `IDENTITY_NEW_PASSWORD`.
Disposable MySQL verification restores the current clean baseline, expects 94
tables, 6 routines, and 55 triggers, proves `auth_security_audit` update/delete fails,
and exercises analyst creation/grant without touching the configured local
stack. Operational details are in
[docs/identity-access-security.md](../docs/identity-access-security.md).

I03 adds `managementapi` and the generated `managementapi/client` without a new
runtime library dependency. Its OpenAPI 3.1 source is
`docs/management-api-openapi.yaml`. Unit gates cover strict JSON/URL/delivery
validation, credential/account quota Lua, account derivation, stable errors,
idempotency replay/conflict, operation deadlines, and generated-client safety
headers. Closeout also restores a uniquely named disposable MySQL 8 baseline
and uses disposable Redis/key material to prove credential rotation,
concurrent writes, version triggers, immutable audit, and cache activation.

S03 focused gates cover closed signal/overflow fixtures, incomplete-evidence
actions and billing, exact canary-review activation, scope authorization,
appeal and maker/checker transitions, immutable snapshots, false-positive
rollback, detached serving refresh, and the strict aggregate command. The
disposable MySQL lifecycle restores the current 94 tables/6 routines/55 triggers and proves
rule/decision immutability, review/appeal, enforcement/rollback, A01 hold, and
fresh-context retention cleanup.

A02 focused gates cover exact-cent provider forms, same-key retries, uncertain
accepted responses, current/previous raw-body signatures, replay/order/event
race behavior, account/MFA/maker-checker authorization, aggregate statement and
refund limits, mandatory approved bindings, immutable payout-country identity,
Held execution denial, immutable opaque mappings, fee/net and exception
reconciliation, stale-submission takeover, the 23-hour replay boundary,
aggregate health, and size-one-pool retention cleanup. Its disposable MySQL
lifecycle uses only a
recorded provider adapter; live Stripe sandbox evidence remains an external
go-live prerequisite.

Schema baseline verification:

```bash
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh diff-schema
```

GitHub Actions runs the package, vet, staticcheck, scoped race, documentation,
and committed-range diff-hygiene gates on pushes and pull requests. Pull
requests check merge-base-to-head and pushes check event `before`-to-`after`,
with an empty-tree fallback for initial history. The local closeout command
remains `git diff --check` so uncommitted whitespace is also covered. The
sibling pzdesign
workflow checks out public Aofei and Genelet beside its own checkout to satisfy
the sibling `replace` directives. Both checkouts are pinned to explicit
reviewed commits and must be bumped intentionally when pzdesign
adopts a new revision. The workflow then runs tests, the `cmd/unify` race test,
vet, staticcheck with the documented legacy style exclusions, and both
template parsers. Both workflows use Go 1.23.5, staticcheck v0.5.1, and Gitleaks
v8.27.2; that Gitleaks pin is the newest reviewed line that still declares Go
1.23 support. Docker smoke,
database-backed admin tests, and schema checks remain explicit local/operator
gates because hosted CI does not start the local service stack.

The pzdesign template parser also enforces rendering source policy: no unsafe
script/data URL schemes, remote executable or embedded dependencies, stored
creative content in fetching/executable elements, assembled query strings, or
raw `html/template` types in application Go code. Genelet tests keep its fixed
CSRF input renderer as the unique trusted-HTML boundary.

D02 uses the standard JSON/XML decoders plus `golang.org/x/net/html`
tokenization to validate middleman Native/VAST/contained markup without
fetching it. The XML walk validates URL-bearing VAST nodes and subjects Native
embedded VAST to the same active-content boundary. Focused auction coverage lives in `match/radv_test.go`,
`match/creative_d02_test.go`, `dsp/creative_validation_test.go`, and
`dsp/auction_d02_test.go`; pzdesign form coverage lives in
`summer/item/filter_test.go` and `summer/creative/filter_test.go`. The
schema/cache smoke must compile the sample's active CPM and URL-backed creatives
before a D02 closeout.

## External Requirements

- Docker CLI and a working Docker daemon.
- Internet access only when pulling Docker images or Go modules.
- No production credentials are required for local development.
