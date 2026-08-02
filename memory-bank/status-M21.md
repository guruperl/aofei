# Status M21 - Middleman Callback Proxy And Price Reconciliation

## Goal

Keep Aofei in the callback path for middleman bids so wins, losses, billable
events, cooperative clicks, and charge/pay prices can be audited before M22
reporting.

## Tasks

- `[+]` Callback token storage.
  - Added Redis-backed selected-bid callback context with TTL under
    `middleman:cb:<token>`.
  - Added click mapping keys under `middleman:click:<requestToken>:<impID>` and
    bill idempotency keys under `middleman:bill:<token>`.

- `[+]` Upstream callback URL rewriting.
  - Final middleman winners receive signed `/mid/win`, `/mid/loss`, and, when a
    downstream billing URL exists, `/mid/bill` URLs.
  - Callback context is created only after winner selection, not for every
    downstream response candidate.

- `[+]` Proxy handlers and downstream notification.
  - `cmd/unify` now exposes `GET /mid/win`, `/mid/loss`, `/mid/bill`, and
    `/mid/click`.
  - Proxy handlers validate signatures, load callback context, publish winloss
    records, and forward stored downstream callbacks when present.

- `[+]` Price reconciliation.
  - Middleman callbacks record charge price, downstream bid price, upstream bid
    price, pay price, margin CPM, currency, and forward status in winloss
    metadata.
  - Downstream `${AUCTION_PRICE}` receives the net payable price, not the
    upstream charge price.

- `[+]` Cooperative click notify.
  - Forwarded middleman requests include
    `ext.aofei_middleman.click_notify_urls` keyed by impression ID.
  - Downstream markup remains owned by the downstream bidder; M21 does not
    rewrite arbitrary `adm`.

- `[+]` Deep review and closeout.
  - Run milestone verification and resolve findings before marking M21
    complete.
  - Resolved review findings for billable publish retry, callback price
    clamping, and duplicate downstream win/loss notification forwarding.

## Carry Forward

- `[+]` Advertiser/operator reporting using middleman price metadata completed
  in M22.
- `[+]` Optional spread/local route-snapshot ownership is consolidated in D03;
  middleman routes remain Redis-only until D03 records its decision.
- `[+]` Durable callback retry queues moved to M24 and are implemented for
  retryable downstream post-auction callback forwarding failures.
- `[X]` Arbitrary downstream markup impression/click rewrite remains a
  non-goal unless a future reporting requirement justifies reopening it.

## Verification

- `[+]` `GOWORK=off go test ./dsp ./cmd/unify ./match`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./dsp ./match ./cmd/unify`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
