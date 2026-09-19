# Prompt V9

M22 should consume the M21 middleman callback facts into reporting and
settlement views.

Advertiser-owned bidder accounts should see what their downstream bidder pays,
not Aofei's upstream charge or exchange margin. Operators still need the full
charge/pay/margin view by bidder, route, publisher, and callback health. The
existing local campaign ledger must keep its current charge-side behavior, so
middleman reporting should use additive ledger tables instead of overloading the
legacy `spend` columns.
