# Status M10 - Documentation And Agent Stewardship

Milestone status: `[+]` Completed

Goal: Keep the harness useful as the project changes.

## Tasks

- `[+]` Define memory-bank update rules after per-milestone status split.
  - Files: `AGENTS.md`, `memory-bank/milestone.md`,
    `memory-bank/status-M*.md`.
  - Acceptance: every agent knows to update the matching `status-M*.md` file
    instead of a removed aggregate status file.

- `[+]` Add a documentation consistency check.
  - Files: `scripts/aofei-doc-check.sh` or `scripts/aofei-verify.sh`.
  - Command to support:
    ```bash
    ./scripts/aofei-doc-check.sh
    ```
  - Acceptance: script fails on links to the removed aggregate status file and
    active docs that reference retired config-directory workflows.

- `[+]` Verify root README stays operator-focused.
  - Files: `README.md`, `docs/*`, `memory-bank/*`.
  - Command:
    ```bash
    sed -n '1,180p' README.md
    ```
  - Acceptance: root README remains short and links to detailed docs instead of
    becoming the full runbook.

- `[+]` Define evolution-file trigger criteria.
  - Files: `evolution/README.md` or `memory-bank/milestone.md`.
  - Acceptance: agents can distinguish normal task progress from direction
    changes that need `evolution/prompt-vN.md` and `evolution/result-vN.md`.

- `[+]` Add a milestone closeout checklist.
  - Files: `memory-bank/milestone.md`, `AGENTS.md`.
  - Acceptance: each milestone has a consistent closeout path: verify commands,
    update docs, update status file, review evolution need.

- `[+]` Check all memory-bank links.
  - Files: `AGENTS.md`, `README.md`, `memory-bank/*.md`, `docs/*.md`.
  - Command:
    ```bash
    rg -n 'memory-bank/status\\.md|status-M[0-9]+\\.md|docs/' README.md AGENTS.md memory-bank docs
    ```
  - Acceptance: links point to existing files and no removed aggregate status
    file is referenced.

- `[+]` Keep per-milestone statuses current during implementation.
  - Files: `memory-bank/status-M*.md`.
  - Acceptance: task statuses reflect actual verification, not intent.

- `[+]` Run M10 verification.
  - Command:
    ```bash
    bash -n scripts/aofei-doc-check.sh
    ./scripts/aofei-doc-check.sh
    git diff --check
    test ! -e memory-bank/status.md
    GOWORK=off go test ./...
    ```
  - Acceptance: whitespace is clean and the removed aggregate status file does
    not exist.

## Review Findings

- `[+]` Add a doc check for stale harness references. It should fail on links to
  the removed aggregate `memory-bank/status.md` and on active docs that restore
  retired config-directory workflows.

- `[+]` Add a guard for legacy credential strings in active docs and memory-bank
  files, with allowlisted historical references only where they are explicitly
  labeled.

- `[+]` Add a guard for stale root SQL-baseline references so future database
  baseline work keeps `etc/step4_init.sql` as the source of truth. The helper
  no longer has an implicit root SQL fallback.

- `[+]` Define how review findings move from status files into milestone
  closeout notes, so open architecture issues do not remain untriaged after a
  milestone is otherwise completed.

### Second Review Pass - 2026-05-12

- `[+]` Refresh `.gitignore` for the current command layout. It still references
  retired config-directory and command paths and misses current command binary
  paths for the NATS client, Redis cache, spread, and MaxMind commands.

- `[+]` Add documentation checks for stale command paths and active config
  examples that reintroduce non-Docker database auth.

- `[+]` Date future review-finding batches and define a closeout convention so
  repeated deep reviews can distinguish new findings, carried findings, and
  resolved findings.

## Result

- Added `scripts/aofei-doc-check.sh` as the fast documentation guard.
- Refreshed `.gitignore` for current installed command binary paths.
- Added milestone closeout rules to `AGENTS.md` and `memory-bank/milestone.md`.
- Added `evolution/README.md` to define when new evolution prompt/result pairs
  are warranted.
- Kept the root README operator-focused; no new runbook content was added.

## Verification

Passed on 2026-05-12:

```bash
bash -n scripts/aofei-doc-check.sh
./scripts/aofei-doc-check.sh
sed -n '1,180p' README.md
git diff --check
test ! -e memory-bank/status.md
GOWORK=off go test ./...
```
