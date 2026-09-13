# Retired milestone M18 - Summer Template Modernization

**Milestone.** M18
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M18.md
**Source specification.** memory-bank/milestone.md#m18---summer-template-modernization
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Recorded command/result evidence preserved in the retired status record's Verification section.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M18 - Summer Template Modernization `[+]`

Make the sibling `pzdesign` UI tree the active Summer/Genelet template and asset
source, and move rendering to Go `html/template`.

Scope:

- Use `../pzdesign/tmpls` for Summer HTML templates and `../pzdesign/www` for
  static UI assets in generated local config.
- Keep `.g` templates as the primary runtime surface; keep `.e` variants
  parse-clean where practical.
- Convert Genelet HTML, login, error, and mail template rendering to
  `html/template`.
- Add advertiser and admin bidder `.g` pages to `pzdesign/tmpls`.
- Add bidder navigation links to the existing advertiser and admin sidebars.

Acceptance:

- All active `.g` action templates in `../pzdesign/tmpls` parse as
  `html/template`.
- Bidder advertiser/admin pages render in tests against the sibling template
  tree when it is present.
- Existing `.e` variants are parse-clean as best-effort coverage but remain
  secondary to the active `.g` templates.
````

## Status record

````markdown
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

- `[+]` Best-effort `.e` cleanup in `pzdesign`.
  - English template variants now parse with the sibling template checker.
  - `.e` templates remain secondary variants, not the primary runtime surface.

- `[+]` Public asset pruning in `pzdesign`.
  - Removed unused demo/test pages, unused JavaScript/vendor assets, and
    checked-in upload payloads from the public `www/` tree.
  - Added `tools/check-templates.go` in `pzdesign` for `.g` and `.e`
    `html/template` parse checks.

## Carry Forward

- `[+]` Page-by-page escaping audit ownership moved to S04.
  - `html/template` auto-escapes output. S04 audits existing pages that
    intentionally preview stored HTML snippets and defines the typed,
    sanitized safe-HTML boundary.

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
- `[+]` `go run ./tools/check-templates.go -ext=.g` in `../pzdesign`
- `[+]` `go run ./tools/check-templates.go -ext=.e` in `../pzdesign`

Exploratory `GOWORK=off staticcheck ./genelet` still reports the known
pre-existing Genelet findings recorded in earlier milestones; it is not used as
the M18 gate.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| `html/template` renderer. | `[+]` | |
| Active template and asset boundary. | `[+]` | |
| Bidder pages in `pzdesign`. | `[+]` | |
| Best-effort `.e` cleanup in `pzdesign`. | `[+]` | |
| Public asset pruning in `pzdesign`. | `[+]` | |
| Page-by-page escaping audit ownership moved to S04. | `[+]` | |
| `GOWORK=off go test ./genelet ./summer/bidder` | `[+]` | |
| all `../pzdesign/tmpls` `.g` action templates parse as `html/template` | `[+]` | |
| `GOWORK=off go test ./summer/bidder ./summer/registry ./genelet ./cmd/unify` | `[+]` | |
| `GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./summer/bidder ./summer` | `[+]` | |
| `GOWORK=off go test ./...` | `[+]` | |
| `GOWORK=off staticcheck ./summer ./summer/registry ./summer/bidder ./cmd/unify` | `[+]` | |
| `./scripts/aofei-doc-check.sh` | `[+]` | |
| `git diff --check` | `[+]` | |
| `git diff --check` in `../pzdesign` | `[+]` | |
| `go run ./tools/check-templates.go -ext=.g` in `../pzdesign` | `[+]` | |
| `go run ./tools/check-templates.go -ext=.e` in `../pzdesign` | `[+]` | |

````
