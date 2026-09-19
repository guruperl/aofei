# Result V8

Resulting direction after M21:

- Middleman bids returned upstream use signed Aofei `/mid/*` callback proxy
  URLs instead of exposing downstream notification URLs directly.
- Selected-bid callback context is short-lived Redis runtime state owned by
  `cmd/unify`, separate from the singleton `middleman:routes` cache owned by
  `cmd/redis-cache`.
- BURL is the preferred billable event. Win notification becomes billable only
  when the selected downstream bid did not provide BURL.
- Aofei records charge price, downstream bid price, pay price, margin, callback
  source, and downstream forwarding status in winloss metadata tied to the
  synthetic reporting chain.
- Downstream ad markup remains untouched. Click measurement is cooperative
  through URLs supplied in forwarded request `ext`, not through arbitrary markup
  rewriting.
- Advertiser/operator reporting over middleman metadata remains M22.
