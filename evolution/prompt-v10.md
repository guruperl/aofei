# Prompt V10

M23 should make middleman route operations manageable from Summer/Genelet.

Operators should no longer need direct SQL edits for `mid_route_group`,
`mid_route_bidder`, or `mid_route_target`, but route cache publication must stay
separate. `cmd/unify` should keep serving UI and ADX traffic from the existing
Redis `middleman:routes` cache. The singleton `cmd/redis-cache` job should
remain the only route-cache refresh path.
