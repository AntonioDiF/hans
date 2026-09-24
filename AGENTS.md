# Agent contract

This is the always-on guide for developing hans. A minimal Go module implements two bounded M1 slices: worker-proposal validation and host-observed action-outcome/provenance contracts. The rest of the runtime remains planned.

## Work within scope

Start with the user's current request and the relevant roadmap milestone. Select a bounded task; do not treat the whole roadmap as authorization to build every feature. Preserve unrelated work and update directly affected documentation.

Use TDD for behavioral code changes: establish a relevant failure before production edits, implement, then verify and refactor. Protect accepted tests and verifier settings from solver changes. If practical automated testing is unavailable, ask for a scoped waiver and record alternative verification. Documentation-only changes need only applicable documentation checks.

Use real tool evidence. A model's review, generated test, or success statement is not a correctness oracle. Verify the exact changed artifacts and combined result; report failures, uncertainty, exhausted limits, and partial work explicitly.

## Preserve boundaries

- Preserve user files, staged changes, and Git history. Do not automatically stage, commit, push, switch branches, stash, reset, or rebase. Create runtime worktrees only within an authorized task.
- Ask before installs, network access, destructive actions, or outside-root changes. Configured inference endpoints require scoped authorization; cloud fallback additionally requires approval of the transmitted data.
- Never treat worktrees as security sandboxes. Initially, repositories and approved commands are trusted.
- Keep authoritative state, permissions, verification, and completion host-owned. Models propose updates; observations and validated transitions establish outcomes. Graphs cannot override these rules.
- Use existing tools and narrow validation. Do not invent dependencies, claim unavailable checks ran, or silently relax a failing requirement.

## Keep context small

Read only the active task's required sections and source slices. Do not bulk-load documents, transcripts, logs, or parent conversations into worker prompts. The planned runtime enforces a configurable 8,192-token input ceiling, reserves output separately, and blocks profiles without trustworthy enforcement.

| Work area | On-demand reference |
|---|---|
| Current task and dependencies | [Roadmap](docs/roadmap.md) |
| Interfaces, workers, tools, permissions | [Architecture](docs/architecture.md) |
| State, budgets, evidence, recovery | [State and context](docs/state-and-context.md) |
| Coding and protected verification | [Coding workflow](docs/coding-workflow.md) |
| Guidance and evolution | [Procedural graphs](docs/procedural-graphs.md) |
| Acceptance and calibration | [Evaluation](docs/evaluation.md) |
| Evidence behind design choices | [Research](docs/research.md) |

Keep this file a small route map, targeting roughly 500-800 tokens once actual profile tokenizers are available. Detailed rules belong in linked sections. Never remove mandatory instructions merely to fit a prompt.
