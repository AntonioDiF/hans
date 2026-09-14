# Roadmap

**Current state:** documentation bootstrap is delivered by this packet. Runtime implementation has not started. This roadmap defines future work; it is not authorization to execute every milestone.

The priority order is bounded state, protected TDD, reviewed procedural guidance, then evaluated evolution. Concurrency and Git worktrees are first-release requirements, not later optimizations.

## Milestones

| ID | Scope | Dependencies | Exit gate |
|---|---|---|---|
| M0: `bootstrap-markdown` | Short agent contract, README, and seven focused design documents. | None | Decisions, evidence limits, ownership, loading routes, and future gates are documented without claiming a runtime exists. |
| M1: `define-runtime-contracts` | Minimal Go module and typed contracts for profiles, context accounting, state/events, actions, workflows, approvals, verifiers, scheduling, and Git baselines/overlays. | M0 | Test-first contracts define success and explicit failure behavior; Windows/Linux and dependency/capability decisions are recorded. |
| M2: `implement-state-evidence` | SQLite state/events, evidence files, revisions, checkpoint/resume, and action reconciliation. | M1 | State, crash consistency, stale-update, and resume cases preserve evidence without false success. |
| M3: `implement-model-budgets` | Configured endpoints, role profiles, complete input accounting/enforcement, reservations, structured outputs, and usage. | M1 | Supported profiles enforce the configured ceiling; unsupported capabilities and invalid responses fail explicitly. |
| M4: `implement-worktree-tools` | Approved commands, detached worktrees, dirty overlays, patches, drift detection, and platform-correct process lifecycle. | M1 | Workspace/policy/platform cases preserve user files, index, branch, and recoverable work; no automatic commits. |
| M5: `implement-concurrent-runtime` | CLI, coordinator, private worker state, dependencies, capacity limits, approvals, cancellation, and integration candidates. | M2, M3, M4 | Independent workers overlap within bounds; shared transitions and integration remain coherent. |
| M6: `implement-tdd-workflow` | Author/review/red/freeze/solve/green/refactor/integrate phases, Go verifier, anti-gaming guards, scoped waivers, and bounded recovery. | M5 | Protected tests cannot be bypassed by a solver; combined artifacts and evidence determine completion. |
| M7: `validate-first-release` | Deterministic suites, Go fixture, 100-step scenario, profile calibration, and budgeted real coding evaluations. | M6 | First-release functional gates hold on Windows/Linux; measured model limitations and finite supported profiles are recorded separately. |
| M8: `evaluate-reviewed-graphs` | Opt-in, bounded, version-pinned advice; compare compact guidance with no graph. | M7 | No policy/verification regressions; measured quality and total cost determine whether guidance is retained. |
| M9: `evaluate-graph-evolution` | Bounded between-run proposals, independent evaluation, provenance, rollback, and manual promotion. | M8 | Candidates meet predeclared gates and receive user approval; no active-run mutation or changes to protected invariants. |

M2, M3, and M4 may be independent workstreams after M1 contracts stabilize. Their concurrency is an implementation choice, not permission to launch work outside the current request.

## Next implementation task

When implementation is requested, begin with a bounded part of M1. Read the relevant [component](architecture.md#components), [workflow](architecture.md#workflow-contract), and [state](state-and-context.md#state-ownership) sections rather than this entire packet.

Specify the first typed boundary and its failure cases, add the relevant failing contract tests, and implement only what those contracts require. Establish deterministic model/tool fixtures before wiring live inference.

Do not create a large speculative package tree or assume a tokenizer/SQLite dependency without documenting compatibility and build constraints. Do not claim the future CLI works until an executable path and its checks exist.

The runtime needs a committed base for worktrees. If a harness-managed run targets an unborn repository, pause for the user to create the initial commit. Creating project files or a Go module does not authorize that commit.

## Working rules

Select the current milestone and a bounded task with explicit acceptance criteria, allowed scope, dependencies, and required document sections. Load only those sections and relevant source/evidence slices.

Use [TDD](coding-workflow.md) for behavioral changes and update the authoritative document when a contract changes. Preserve user work and report unexpected blockers rather than inventing a success-shaped fallback.

Keep stable design decisions in their owning documents. Mutable runtime progress belongs in transactional state, not competing Markdown summaries. Session task tracking should reflect the current request, not automatically activate all product milestones.

At a handoff, record the actual artifact/version, evidence, unresolved limitations, and next scoped task. Do not label a milestone complete because a skeleton compiles or an individual worker passed.

## Release gate

The first release includes M1 through M7. It must demonstrate the concurrent two-change Go fixture, combined-result verification, a 100-step state/recovery fixture, strict complete-input enforcement, durable resume, approved tools, and cross-platform behavior.

Live-model performance remains a separate measured claim. Use the [evaluation protocol](evaluation.md); do not substitute mocked success, compact prompts, or a favorable synthetic score for real coding evidence.

M8 and M9 are later experiments. Graphs may remain disabled if they do not justify their quality/cost trade-off.

## Deferred decisions

Exact model families, tokenizer/template integrations, SQLite driver, detailed CLI syntax, concrete concurrency/retry/output reservations, retention limits, and empirical promotion thresholds are implementation or calibration decisions.

Resolve each before enabling dependent behavior and record evidence in its owning document/profile. Use finite conservative settings; unsupported hard-budget profiles remain blocked. These are intentionally deferred engineering choices, not permission for silent guesses.

Do not expand the initial scope into model loading, a web UI/TUI, a public plugin platform, embedding memory, non-coding service products, OS-enforced sandboxing, autonomous graph promotion, or in-run evolution.
