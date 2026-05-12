# Prompt V5

The DSP should support a future middleman AdX use case. When an upstream AdX
sends a bid request and no local campaign can bid, Aofei should eventually be
able to fan out to downstream DSPs in parallel under a tight deadline, discard
late responses, mark up the selected downstream bid, and return a normal
OpenRTB response upstream.

Before runtime fanout, the database needs a clear downstream endpoint and
reporting boundary. Downstream bidders should be owned by existing advertiser
accounts so advertiser auth, ledger, and reporting tools can be reused, while
operators control credentials, synthetic reporting IDs, activation, routing, and
margins.
