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
| Action | Block for 10 seconds |

This profile absorbs rapid bursts without bringing DSP endpoints into the
rule. It is deliberately high enough that ordinary page loads should not
trigger the rule, but users behind shared NAT may still be blocked for 10
seconds when their aggregate traffic crosses the threshold. Cloudflare Free is
not entitled to Managed Challenge for rate-limiting rules; Cloudflare's
[sensitive-form guidance](https://developers.cloudflare.com/use-cases/solutions/protect-sensitive-forms-fraud-abuse/)
directs Free zones to use Block.

The profile is not equivalent to a POST-only 10-request/10-minute rule. Do not
claim that Cloudflare enforces the application's sustained quota while the zone
uses the Free plan.

## Active Production State

The Free-plan profile was activated and independently read back on 2026-08-24.
The managed widget is named `w8m-public-account` and allows only `w8m.com` and
`www.w8m.com`. The zone `http_ratelimit` entry point is version 1 with ruleset
ID `0710c013f0c540d18a75421ce35d3990`; it contains exactly one enabled rule,
`7b97765482bf4fd79a8487fc19749bda`, with the expression and limits documented
above. These identifiers are operational metadata, not credentials.

The live burst proof returned `200` for the first ten rapid account-page
requests and `429` for requests eleven and twelve. During that mitigation,
an intentionally invalid `POST /pz` reached the origin and returned its normal
application `400`, rather than an edge `429`; the account page returned `200`
again after the 10-second mitigation expired. This confirms the active rule's
UI-only path boundary without treating an invalid bid request as a successful
ad-serving transaction.

## Layered Enforcement

The production boundary remains layered:

1. Cloudflare applies the path-scoped 10-second burst block.
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
