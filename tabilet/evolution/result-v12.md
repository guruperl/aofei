# Result V12

Resulting direction after M25:

- `middleman_always_enabled` gates `trigger_mode='Always'` fanout and defaults
  false in checked-in and generated local configs.
- `middleman:routes` entries include `trigger_mode`; old entries without it are
  treated as `Fallback`.
- `Fallback` routes still fan out only after local no-bid.
- When both middleman gates are enabled, `Always` routes can fan out for
  locally filled impressions, and marked-up downstream bids compete with local
  bids on effective CPM.
- Unsafe local price comparison keeps the local winner.
- Middleman callback setup is deferred until after final winner selection, so
  losing middleman bids do not create callback context.
- Spread/local route snapshots and real settlement execution remain future
  work.
