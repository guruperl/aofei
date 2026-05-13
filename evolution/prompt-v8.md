# Prompt V8

Middleman bidder runtime should keep Aofei in the OpenRTB callback path after
M20 starts returning downstream bids upstream.

Downstream bidders own their ad markup, impression pixels, and click tags, so
M21 should not rewrite arbitrary `adm`. Aofei still needs callback authority for
audit and price reconciliation: win/loss notifications should be proxied, BURL
should be the preferred billable signal, win should be the billable fallback
when BURL is unavailable, and cooperative click notification should be exposed
for downstream markup to opt in.

The callback flow must preserve separate charge and pay prices. Aofei charges
the upstream clearing price and pays the downstream bidder a net price after
margin, then logs enough metadata for later advertiser/operator reporting.
