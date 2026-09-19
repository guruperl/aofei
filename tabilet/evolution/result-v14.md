# Result V14

Resulting direction after the module rename:

- Aofei's Go module path is `github.com/guruperl/aofei`.
- Active Aofei packages import each other through `github.com/guruperl/aofei`.
- `pzdesign` depends on Aofei through `github.com/guruperl/aofei` with the
  local development replace pointing to `../aofei`.
- Historical files under `backup/` remain historical and may still mention the
  retired `github.com/genelet/winter` path.

