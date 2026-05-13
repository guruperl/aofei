# Prompt V7

Middleman bidder runtime should move from schema/UI preparation into DSP core.
Approved advertiser-owned bidders are selected like demand candidates: active
routes choose the coarse bidder pool, and the bidder's synthetic item ACL and
channel rules decide whether the original publisher/site/slot may be forwarded.

The forwarded OpenRTB request should preserve the original payload semantics and
full impression list, while the auction accepts downstream bids only for
impressions that local campaign matching could not fill. It should replace the
upstream exchange domain with Aofei/W8M identity through `ext.request_domain`.
Bidder routes should be cached like campaign data, but the singleton
`cmd/redis-cache` job remains the owner of cache refresh.
