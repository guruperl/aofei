# Prompt V28

Protect W8M's public advertiser and publisher registration and password-
recovery pages from automated spam while preserving the existing Gmail API,
account, activation, and authenticated-portal contracts.

- use Cloudflare Turnstile without a permanent checkbox and require exact
  server-side hostname/action validation before expensive or mutating work;
- add atomic application quotas that retain no raw email/IP identity and fail
  closed before external mail or account writes;
- derive client identity only through an explicit trusted-proxy chain;
- keep secrets in owner-only deployment state, never source/config examples;
  and
- treat Cloudflare widget/rate-rule creation and W8M deployment as an explicit
  activation gate with live readback and rollback evidence.
