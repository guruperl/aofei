# Prompt V12

M25 should implement middleman `trigger_mode='Always'` without disturbing the
M24 reliability boundary. `middleman_enabled` remains required, and a new
`middleman_always_enabled` gate must default false. `Fallback` routes must
remain local-no-bid-only. When enabled, `Always` routes may fan out for eligible
impressions even when local demand can bid, and final winner selection should
compare marked-up middleman bids against local effective CPM.
