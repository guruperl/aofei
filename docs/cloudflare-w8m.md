# Cloudflare W8M Free-Plan Boundary

This document records the Cloudflare Free-plan boundary for `w8m.com`, with
particular attention to the S06 public advertiser and publisher account UI.
It complements the application contract in
[public-account-abuse-protection.md](public-account-abuse-protection.md).

## Scope

The Cloudflare rate-limit rule is scoped by exact path to these public Summer
account endpoints:

```text
/goto/web/g/adv
/goto/web/e/adv
/goto/web/g/pub
/goto/web/e/pub
```

Requests to DSP, SSP, measurement, action, and middleman endpoints do not match
that expression. In particular, the rule does not apply to `/bid/*`, `/pz`,
`/imp`, `/clk`, `/win`, `/loss`, `/action`, or `/mid/*`.

The path match must remain exact. Do not replace it with `/goto/*`, a hostname-
wide rule, or another prefix expression that would include authenticated UI or
ad-serving traffic.

## Free-Plan Limitations

Cloudflare Free rate limiting is an edge burst control, not the authoritative
S06 sustained quota:

- The counting period is fixed at 10 seconds. The Free plan cannot express the
  original edge target of 10 requests per 10 minutes.
- A Free-plan rule expression can use Path and Verified Bot status, but not the
  HTTP method. Both `GET` page loads and `POST` submissions on the four exact
  paths therefore count.
- The edge cannot distinguish registration, recovery, login, activation, or
  reset-completion actions multiplexed through the same path.
- IP is the available counting characteristic. NAT-aware identity, headers,
  cookies, email addresses, and form fields are unavailable as Free-plan
  characteristics.
- The zone receives one rate-limiting rule. Custom counting expressions and
  cache exclusion are unavailable.
- Cloudflare counters are maintained per data center, and enforcement can lag
  by several seconds. The rule does not guarantee that an exact number of
  requests reaches the origin and is not a defense against a distributed set
  of client IPs.

Cloudflare documents these limits in its
[rate-limiting availability table](https://developers.cloudflare.com/waf/rate-limiting-rules/#availability)
and
[request-rate calculation guide](https://developers.cloudflare.com/waf/rate-limiting-rules/request-rate/).

## W8M Free-Plan Profile

The constrained edge profile for the four paths is:

| Setting | Value |
|---|---|
| Match | Exact path set above, excluding verified bots |
| Methods | All methods; Free cannot select only `POST` |
| Characteristic | Client IP |
| Threshold | 10 requests per 10 seconds |
| Action | Managed Challenge |

This profile absorbs rapid bursts without bringing DSP endpoints into the
rule. It is deliberately high enough that ordinary page loads should not
trigger a challenge, but users behind shared NAT may still be challenged when
their aggregate traffic crosses the threshold.

The profile is not equivalent to a POST-only 10-request/10-minute rule. Do not
claim that Cloudflare enforces the application's sustained quota while the zone
uses the Free plan.

## Layered Enforcement

The production boundary remains layered:

1. Cloudflare applies the path-scoped 10-second burst challenge.
2. Turnstile verifies the exact registration or recovery action and hostname
   before expensive or mutating work.
3. The application Redis script enforces the authoritative pseudonymous IP,
   normalized-email, and global quotas. Its default IP limit remains 10
   submissions per 10 minutes and 50 per day.

Because the application quotas run only after successful Turnstile validation,
the Free edge rule cannot replace either Turnstile or Redis. Conversely, the
Free limitation does not justify broadening the edge rule to DSP traffic.

## Upgrade Boundary

Cloudflare Business or higher is required to express both `POST` and a
10-minute counting period in this zone-level rule. If the zone is upgraded,
replace the constrained profile with the exact POST-only rule documented in
the S06 contract, inspect the resulting `http_ratelimit` entry-point order, and
update this document and the S06 status evidence. A plan upgrade alone does not
create or enable a rule.
