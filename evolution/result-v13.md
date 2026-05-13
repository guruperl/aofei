# Result V13

Resulting direction after the Summer/Genelet source move:

- Aofei remains the DSP/schema/runtime module `github.com/guruperl/aofei`.
- The sibling `../pzdesign` checkout is the admin/design module
  `github.com/guruperl/pzdesign`.
- `pzdesign` owns `cmd/unify`, `genelet/`, `summer/`, `tmpls/`, `www/`, and the
  Summer/Genelet docs under `docs/`.
- `cmd/unify` lives in `pzdesign` and imports Aofei's DSP runtime through
  `github.com/guruperl/aofei/dsp`.
- Generated Summer config points `ProjectRoot`, `Template`, and `DocumentRoot`
  at the `pzdesign` checkout, while uploads, logs, DSP config, schema, and
  Docker services remain owned by Aofei.
- `pzdesign` may continue to import Aofei domain packages such as `acl/`,
  `match/`, and `uploaded/` until those boundaries are split further.
