# Retired milestone M00 - Agentic Harness Bootstrap

**Milestone.** M00
**Outcome.** completed
**Retired.** 2026-09-13
**Source status.** memory-bank/status-M00.md
**Source specification.** memory-bank/milestone.md#m00---agentic-harness-bootstrap
**Evidence.** 3eff099173403ea5368b7fe12804c738bcd366e2
**Worktree.** includes uncommitted changes
**Review.** passed
**Review iterations.** 1
**Verification.** Milestone closed with its recorded acceptance evidence preserved in the retired status record.
**Consolidated into.** No current-truth change recorded; current facts remain in memory-bank/product.md, architecture.md, and tech-stack.md.

## Milestone specification

````markdown
## M00 - Agentic Harness Bootstrap `[+]`

Create the project operating layer for future agent work.

Scope:

- Add `AGENTS.md`, `memory-bank/`, `evolution/`, and `docs/`.
- Rewrite the root README as a clean current operator entry point.
- Move or preserve old long-form documentation under `docs/`.
- Record milestone-level direction without inventing detailed task breakdowns.

Acceptance:

- A new agent can read `AGENTS.md` and know where to start.
- Product, architecture, tech stack, milestones, and coarse status are captured.
- Historical docs are not lost, but the root README no longer presents legacy
  manual setup as the active workflow.
````

## Status record

````markdown
# Status M00 - Agentic Harness Bootstrap

Milestone status: `[+]` Completed

Goal: Create the project operating layer for future agent work.

## Tasks

- `[+]` Inventory the sample harness in `../skills` and the reference harness in
  `../tfconfig`.
  - Files checked: `../skills/README.md`, `../skills/AGENTS.md`,
    `../skills/memory-bank/*`, `../skills/evolution/*`, `../tfconfig/AGENTS.md`.
  - Verification: sample structure identified as `AGENTS.md`, `memory-bank/`,
    `evolution/`, and `docs/`.

- `[+]` Move historical root documentation into `docs/`.
  - Files moved: `README.md` -> `docs/legacy-operations.md`,
    `introduction.md` -> `docs/dsp-architecture.zh.md`.
  - Verification: moved files are present under `docs/` and root README is
    available for current operator content.

- `[+]` Create the current operator README.
  - File: `README.md`.
  - Content: Docker MySQL/Redis/NATS quick start, repo map, active config notes,
    and legacy auth warning.
  - Verification: README points to active docs and does not present old systemd
    setup as the active path.

- `[+]` Create the agent bootstrap guide.
  - File: `AGENTS.md`.
  - Content: start order, project boundaries, commands, hard rules, and work
    cadence.
  - Verification: agents are directed to the memory bank and per-milestone
    status files.

- `[+]` Create memory-bank source-of-truth files.
  - Files: `memory-bank/product.md`, `memory-bank/architecture.md`,
    `memory-bank/tech-stack.md`, `memory-bank/milestone.md`.
  - Verification: product scope, runtime architecture, local tooling, and
    milestone roadmap are captured.

- `[+]` Create evolution history for the harness setup.
  - Files: `evolution/prompt-v1.md`, `evolution/result-v1.md`.
  - Verification: initial request and resulting harness decisions are recorded.

- `[+]` Add focused long-form docs for current operations.
  - Files: `docs/local-docker-runtime.md`, `docs/database-baseline.md`.
  - Verification: docs cover Docker services, generated configs, Redis/NATS
    commands, and schema baseline rules.

- `[+]` Run verification.
  - Commands:
    ```bash
    git diff --check
    GOWORK=off go test ./cmd/redis-cache ./cmd/nats-client ./cmd/spread ./etc ./dsp ./acl ./match -run '^$'
    ```
  - Result: both commands passed.

## Retired Task Status

| Item | State | Notes |
|---|---|---|
| Inventory the sample harness in `../skills` and the reference harness in | `[+]` | |
| Move historical root documentation into `docs/`. | `[+]` | |
| Create the current operator README. | `[+]` | |
| Create the agent bootstrap guide. | `[+]` | |
| Create memory-bank source-of-truth files. | `[+]` | |
| Create evolution history for the harness setup. | `[+]` | |
| Add focused long-form docs for current operations. | `[+]` | |
| Run verification. | `[+]` | |

````
