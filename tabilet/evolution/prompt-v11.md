# Prompt V11

M24 should improve reliability and operations for the current middleman
fallback system without changing auction winner selection. Route-cache rollout
needs visibility and a route-only refresh mode, but Summer must not execute
cache refreshes. Downstream `/mid/*` callback forwarding failures need durable
retry, but `/bid` must not write MySQL retry rows or do slow operational work.

M25 is planned separately for `trigger_mode='Always'` and effective-CPM
competition with local bids.
