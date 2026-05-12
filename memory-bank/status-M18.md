# Status M18 - Summer Template Modernization

## Goal

Use the sibling `pzdesign` UI tree as the active Summer/Genelet template and
asset source, render HTML through Go `html/template`, and add bidder portal
pages there.

## Completed

- `[+]` `html/template` renderer.
  - Genelet page, login/error, and mail-template rendering now use
    `html/template`.

- `[+]` Active template and asset boundary.
  - Local Summer config generation points `Template` at `../pzdesign/tmpls`.
  - Local Summer config generation points `DocumentRoot` at `../pzdesign/www`.
  - `AOFEI_PZDESIGN_ROOT` can override the sibling checkout location.

- `[+]` Bidder pages in `pzdesign`.
  - Added advertiser bidder `.g` pages for topics, startnew, edit, insert, and
    update.
  - Added admin bidder `.g` pages for topics, edit, update, and approve.
  - Added bidder navigation links to advertiser and admin sidebars.

## Deferred

- `[ ]` `.e` template cleanup and validation.
  - `.e` templates are intentionally out of scope for M18.
  - Known example: `pub/pub/dashboard.e` references `$item` before declaring it.

- `[ ]` Page-by-page escaping audit.
  - `html/template` auto-escapes output. Existing pages that intentionally
    preview stored HTML snippets may need typed safe-HTML handling in a later
    review.

## Verification

- `[+]` `GOWORK=off go test ./genelet ./summer/bidder`
- `[+]` all `../pzdesign/tmpls` `.g` action templates parse as `html/template`
- `[+]` `GOWORK=off go test ./summer/bidder ./summer/registry ./genelet ./cmd/unify`
- `[+]` `GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer/bidder ./summer`
- `[+]` `GOWORK=off go test ./...`
- `[+]` `GOWORK=off staticcheck ./summer ./summer/registry ./summer/bidder ./cmd/unify`
- `[+]` `./scripts/aofei-doc-check.sh`
- `[+]` `git diff --check`
- `[+]` `git diff --check` in `../pzdesign`

Exploratory `GOWORK=off staticcheck ./genelet` still reports the known
pre-existing Genelet findings recorded in earlier milestones; it is not used as
the M18 gate.
