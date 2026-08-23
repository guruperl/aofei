# Public Account Abuse Protection

This is the S06 implementation and operating contract for public advertiser and
publisher registration and password-recovery email requests. It covers both
Chinese (`g`) and English (`e`) Summer pages. It does not change authenticated
portals, login, activation links, or password-reset completion links.

## Request Boundary

When `PUBLIC_ACCOUNT_PROTECTION_ENABLED=true`, the protected submissions are:

| Public form | POST path | Server action |
|---|---|---|
| Advertiser registration | `/goto/web/g/adv`, `/goto/web/e/adv` | `register_adv` |
| Advertiser recovery | `/goto/web/g/adv`, `/goto/web/e/adv` | `recover_adv` |
| Publisher registration | `/goto/web/g/pub`, `/goto/web/e/pub` | `register_pub` |
| Publisher recovery | `/goto/web/g/pub`, `/goto/web/e/pub` | `recover_pub` |

The existing Genelet CSRF token remains mandatory. Turnstile is a separate bot
boundary: the browser obtains a short-lived, single-use token and the server
posts it directly to Cloudflare Siteverify. A successful response must name an
allowed hostname and the exact fixed action rendered by the form. The token is
then removed from form state.

Turnstile is also an external security-processing boundary. Before enablement,
approve Cloudflare for the intended markets and update the public privacy
notice/vendor records as required. W8M sends the one-time token, derived client
IP, and fixed action to Siteverify; it does not retain the token or raw IP in
quota state. Cloudflare's own browser telemetry and retention remain governed
by the operator's Cloudflare terms and notice. See
[privacy-data-governance.md](privacy-data-governance.md).

The order is deliberate:

1. verify Turnstile before password hashing, Redis, Google OAuth preflight,
   database access, account mutation, or Gmail;
2. run the existing form/password validation;
3. atomically consume the application quotas in Redis;
4. verify Gmail credentials, perform the existing model operation, and send
   mail.

Missing or invalid tokens return localized `400`; Turnstile or Redis
availability/configuration failure returns localized `503`; an exceeded quota
returns localized `429`. None of these paths sends email or mutates an account.
Provider error codes and tokens are not returned or logged.

## Application Quotas

One Redis script checks every window before incrementing any window. A denied
request therefore cannot partially consume other counters. Each key gets a TTL,
and all keys share one Redis Cluster hash tag. Normalized email addresses and
client IPs are represented only by deployment-keyed HMAC-SHA256 digests; raw
values never enter a Redis key or metric label.

| Environment override | Default | Window |
|---|---:|---:|
| `PUBLIC_ACCOUNT_IP_10M_LIMIT` | 10 | 10 minutes |
| `PUBLIC_ACCOUNT_IP_DAY_LIMIT` | 50 | 24 hours |
| `PUBLIC_ACCOUNT_EMAIL_HOUR_LIMIT` | 5 | 1 hour |
| `PUBLIC_ACCOUNT_EMAIL_DAY_LIMIT` | 20 | 24 hours |
| `PUBLIC_ACCOUNT_GLOBAL_HOUR_LIMIT` | 200 | 1 hour |
| `PUBLIC_ACCOUNT_GLOBAL_DAY_LIMIT` | 1000 | 24 hours |

Every override must be an integer from 1 through 1,000,000. Redis failure is
fail-closed because these paths can cause external mail and account writes.
Do not scan-delete quota keys during normal deployment; wait for their bounded
expiry. Changing the Summer `Secret` changes future pseudonyms and should occur
only through the existing coordinated secret-rotation process.

## Trusted Client IP

`PUBLIC_ACCOUNT_TRUSTED_PROXY_CIDRS` is mandatory when protection is enabled.
The server parses `X-Forwarded-For` from right to left and removes only peers in
that explicit set. The first untrusted address is the client. If the direct peer
is not trusted, forwarding headers are ignored. A malformed chain from a
trusted peer is rejected. This prevents a direct connection to port 8200 or to
Apache from choosing another quota identity.

For the current W8M topology, include loopback (Apache to `unify`) and the
current Cloudflare IPv4/IPv6 proxy networks. Retrieve the authoritative lists
from `https://www.cloudflare.com/ips-v4/` and
`https://www.cloudflare.com/ips-v6/`, review the change, update every node, and
restart as one coordinated rollout. Do not copy an unreviewed forwarding header
or use `CF-Connecting-IP` merely because it is present.

## Cloudflare Setup

Use a narrowly scoped Cloudflare API token or the dashboard. The token needs
account access to Turnstile Sites Write plus zone read and WAF/rulesets write
for `w8m.com`; store it outside the application service and revoke/rotate it
after automation if continuous management is unnecessary.

1. Create one managed Turnstile widget named `w8m-public-account` with only
   `w8m.com` and `www.w8m.com` in its hostname allowlist. Do not use Cloudflare
   test keys in production.
2. Retain the returned site key and secret only in an owner-readable secret
   handoff. The site key is public; the secret is not.
3. Add a zone rate-limit rule for `POST` requests whose path is one of
   `/goto/web/g/adv`, `/goto/web/e/adv`, `/goto/web/g/pub`, or
   `/goto/web/e/pub`. Use the real client IP as the characteristic, exclude
   verified bots where the plan supports it, and start with a managed challenge
   or block threshold no looser than 10 requests per 10 minutes. Because these
   paths also carry activation/reset completion actions, canary the edge
   threshold and keep the application email quotas authoritative.
4. Keep rate-limit rules at the end of the `http_ratelimit` entry-point rule
   list as required by Cloudflare, and preserve any pre-existing rules.
5. Inspect the resulting widget and entry-point rate-limit rules after creation;
   do not assume an API success response proves the intended rule order or plan
   capability.

The relevant Cloudflare OpenAPI operations are
`POST /accounts/{account_id}/challenges/widgets` and the zone
`http_ratelimit` entry-point ruleset. The OpenUdon/ApiTools catalog contains the
current Cloudflare OpenAPI description, but production mutation still requires
an explicitly authorized, valid credential and exact-resource readback.

## Service Configuration

Create an owner-only environment file such as
`%h/.config/aofei/public-account-protection.env` with mode `0600`:

```bash
PUBLIC_ACCOUNT_PROTECTION_ENABLED=true
TURNSTILE_SITE_KEY=<public site key>
TURNSTILE_SECRET_KEY=<secret key>
TURNSTILE_HOSTNAMES=w8m.com,www.w8m.com
PUBLIC_ACCOUNT_TRUSTED_PROXY_CIDRS=127.0.0.0/8,::1/128,<reviewed Cloudflare CIDRs>
```

Add a `systemctl --user` drop-in containing only:

```ini
[Service]
EnvironmentFile=%h/.config/aofei/public-account-protection.env
```

Run `systemctl --user daemon-reload`, rebuild the reviewed `unify` binary,
restart the service, and require `active` plus a successful `/healthz` before
testing public forms. Startup fails if enabled configuration is partial, a
quota is invalid, Redis is absent, or hostname/proxy lists are empty. Merely
setting the Turnstile keys without the enable gate also fails startup, avoiding
a silent operator mistake.

## Verification And Monitoring

Before production enablement, prove all of the following:

- each of the four Chinese and four English start pages contains one widget,
  the public site key, and its exact action, but never the secret;
- missing, invalid, expired, replayed, wrong-hostname, and wrong-action tokens
  fail before Redis/account/mail work;
- valid advertiser and publisher registration and recovery reach the existing
  Gmail workflow without revealing whether a recovery email exists;
- repeated valid submissions produce `429`, leave every quota counter atomic,
  and expose no raw email/IP in Redis, metrics, or logs;
- a direct origin request cannot spoof `X-Forwarded-For`;
- Cloudflare rule readback, protected metrics, Gmail delivery, and rollback
  evidence are retained with no token or credential values.

Monitor these fixed-cardinality expvar maps on the already protected metrics
surface:

- `aofei_public_account_submissions_total`;
- `aofei_public_account_turnstile_rejections_total`;
- `aofei_public_account_rate_limited_total`;
- `aofei_public_account_dependency_errors_total`.

Alert on sustained dependency errors or a sharp change in rejection/limit
ratios. No action, scope, hostname, account, email, IP, or token supplied by a
request becomes a metric key.

## Rotation And Rollback

Rotate a Turnstile secret through Cloudflare's supported grace-period workflow,
update all nodes' owner-only environment, restart, verify, then invalidate the
old secret. Never keep both values in Git, JSON, shell history, or tickets.

If verification causes an incident, the safe rollback is to disable public
account mail first (remove the `_gmail` block or its credential environment),
then remove both Turnstile key assignments, set
`PUBLIC_ACCOUNT_PROTECTION_ENABLED=false`, restart, and inspect the edge rule.
Do not reopen email-producing forms without either this protection or an
approved equivalent. Login and authenticated portals remain available while
public account mail is disabled.
