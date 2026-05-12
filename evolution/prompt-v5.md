# Prompt V5

The DSP should support a future middleman AdX use case. When an upstream AdX
sends a bid request and no local campaign can bid, Aofei should eventually be
able to fan out to downstream DSPs in parallel under a tight deadline, discard
late responses, mark up the selected downstream bid, and return a normal
OpenRTB response upstream.

Before runtime fanout, the database needs a clear downstream DSP identity and
schema boundary. Downstream DSPs should have self-service access for endpoint
metadata and reports, while operators control credentials, activation, routing,
and margins.
