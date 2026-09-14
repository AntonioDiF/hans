# hans

A standalone, local-model-first agentic harness designed for bounded context and reliable long-running work.

**Status: guidelines and architecture only.** There is no runtime, executable CLI, or buildable Go module yet. The [roadmap](docs/roadmap.md) separates the first release from later experiments.

## Priorities

1. Use a SKILL.state-inspired execution model: bounded live state, with historical evidence available outside the prompt.
2. Make test-driven development the default for coding, with explicit user waivers rather than silent exceptions.
3. Evaluate reviewed procedural guidance, then bounded between-run graph evolution with user-approved promotion.

These are design choices informed by research, not demonstrated performance claims for this project. See the [research assessment](docs/research.md) for evidence and limitations.

## First-release design

The harness will be written in **Go**, support **Windows and Linux**, and expose a terminal CLI with progress, approval prompts, and resumable runs. Coding is the first workflow; language-neutral tool and verification contracts leave room for other workflows.

| Role | Initial model target |
|---|---|
| Planning, acceptance-test review, recovery | Approximately 30B |
| Bounded execution | Approximately 10B |

Both models use configured OpenAI-compatible endpoints. The server or user manages model loading. Cloud fallback requires explicit approval, including the data that would leave the machine.

Every model call has an initial **8k input ceiling, concretely 8,192 tokens**, configurable per profile, with output reserved separately. Unsupported hard-budget profiles must be blocked rather than described as approximately compliant.

The first release includes concurrent independent workers, configurable per-model capacity, SQLite-backed state, external evidence files, and isolated Git worktrees. The coordinator verifies the combined result before applying it to the selected workspace. No automatic commits are permitted; harness-managed worktree runs require an existing committed base.

Approved repositories and commands are trusted initially. Worktrees prevent edit collisions; they are **not security sandboxes**.

## Documentation map

| Document | Read it for |
|---|---|
| [Agent contract](AGENTS.md) | Always-on instructions for agents developing this repository. |
| [Architecture](docs/architecture.md) | Components, interfaces, concurrency, worktrees, permissions, and extension boundaries. |
| [State and context](docs/state-and-context.md) | Budget enforcement, state ownership, evidence, persistence, and resume. |
| [Coding workflow](docs/coding-workflow.md) | Test authoring, review, red/green gates, protected verification, and recovery. |
| [Procedural graphs](docs/procedural-graphs.md) | Deferred guidance and evolution experiments, authority limits, and promotion. |
| [Evaluation](docs/evaluation.md) | Deterministic contracts, coding fixtures, long-horizon scenarios, and model calibration. |
| [Roadmap](docs/roadmap.md) | Dependency-ordered implementation milestones and release gates. |
| [Research](docs/research.md) | Primary sources, supported findings, adaptations, and open empirical questions. |

Load the current milestone and relevant document sections, not the entire packet. `AGENTS.md` is the short entry contract; detailed policies live in their owning documents.
