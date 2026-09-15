# Architecture

This document specifies the intended runtime. Only the minimal Go module, [worker-proposal boundary](state-and-context.md#implemented-proposal-boundary), and [host-observation boundary](state-and-context.md#implemented-observation-boundary) are implemented so far; the component table below is not a list of delivered services. Use the [roadmap](roadmap.md) to select implementation work.

## Scope

Build a standalone Go harness for Windows and Linux with a terminal CLI. It owns the agent loop, calls configured model servers, and supports coding first. It is not an instruction layer over another runtime.

The first release includes concurrent independent workers, configurable model capacity, strict context budgeting, durable pause/resume, protected TDD, and Git-worktree integration. Reviewed procedural graphs and evolution follow the first-release foundation.

Do not begin with a web UI, TUI framework, public plugin SDK, model loader, embedding-memory platform, service-automation product, or OS sandbox. Introduce packages only with their first tested use.

## Components

| Boundary | Responsibility |
|---|---|
| CLI and run service | Start, inspect, pause, resume, approve, and cancel runs; expose actual progress and terminal status. |
| Model adapter | OpenAI-compatible requests, profile capabilities, cancellation, structured responses, and usage metadata. |
| Context builder | Select input, count complete serialized requests, reserve output, and reject unsupported or oversized calls. |
| Coordinator and scheduler | Validate dependencies, allocate capacity, manage workers, commit shared transitions, and integrate results. |
| Workflow adapter | Supply a versioned procedure, domain state schema, actions, validation gates, and completion contract. |
| State and evidence stores | Transactional SQLite state/events and recoverable, integrity-checked external evidence. |
| Workspace manager | Capture Git baselines and dirty overlays, create worktrees, extract patches, detect drift, and preserve outputs. |
| Tool and policy gateway | Validate action scope and approval, execute bounded operations, and report structured outcomes. |
| Verification adapter | Execute approved checks and bind their results to exact artifacts and verifier versions. |
| Procedural guidance | Optional, version-pinned advice with no authority over execution policy or completion. |

Prefer typed Go interfaces and explicit errors. A general interface should isolate an actual variation, not anticipate every possible plugin.

## Workflow contract

A workflow provides its task/acceptance schema, immutable procedure version, bounded domain state schema, allowed action definitions, required capabilities, artifact types, verifier, and completion rules.

The core must not assume every artifact is source code or every verifier is a test command. Exercise the boundary with a small non-coding contract fixture. Do not turn that fixture into a second production workflow in the first release.

Target-repository instructions and declared command profiles inform execution but do not grant permissions. Tools return observations with provenance; they do not directly assign success.

## Model profiles

The initial routing uses approximately 30B for planning, acceptance-test review, and recovery, and approximately 10B for bounded execution. This is a hypothesis to [calibrate](evaluation.md#model-calibration), not an assumed optimum.

| Profile data | Required meaning |
|---|---|
| Identity and endpoint | Actual model/version, configured OpenAI-compatible URL, and locally managed authentication. |
| Capacity | Usable context, input ceiling, output/reasoning reservation, and compatible tokenizer/template or verified input-limit enforcement. |
| Response contract | Supported structured-output mechanism and exact response validation behavior. |
| Scheduling | Finite model/endpoint capacity, shared-service capacity where applicable, timeouts, and retry budgets. |
| Authorization | Approved endpoint/data scope; cloud fallback disabled without distinct explicit approval. |
| Calibration | Server/tokenizer versions, capability results, and measured configuration. |

The server or user manages loading and unloading models. Do not require both models to be resident through harness-managed lifecycle operations.

OpenAI compatibility alone does not establish tokenization, schema-constrained decoding, or a safe input limit. Follow the [input budget contract](state-and-context.md#input-budget) and reject incomplete or unsupported profiles.

## Scheduling

Use a task-dependency graph to schedule independent subtasks. This ordinary scheduling graph is distinct from the later learned procedural-advice graph.

Workers have private bounded contexts and private versioned state. Exchange scoped contracts, dependency results, and evidence handles, not conversations. The coordinator owns authoritative shared-state transitions; see [state ownership](state-and-context.md#state-ownership).

Enforce configurable per-model/endpoint capacity and a global worker bound. Profiles sharing a constrained service must not accidentally multiply its capacity. Every planning, review, execution, recovery, and approved cloud call counts against applicable budgets.

Define queue fairness, dependency failure, cancellation, late-response handling, and no-progress accounting explicitly. Cancellation must prevent stale results from being committed; it does not prove an already-started external action had no effect.

Concurrency is required for the first release, but higher local inference concurrency is not presumed faster. Select finite limits during calibration; do not support zero-as-unlimited or equivalent unbounded settings.

## Worktrees

Harness-managed coding runs require an existing committed Git base. If none exists, pause for the user to create an initial commit. Never create it automatically or silently use a snapshot backend. This runtime precondition does not prevent editing repository documentation.

Use detached Git worktrees without switching the user's branch. Record the base commit and a faithful approved dirty overlay, including relevant staged, unstaged, deleted, binary, and untracked content. Preserve file modes and supported links; make unsupported inputs explicit rather than silently changing them.

Do not indiscriminately copy credentials, ignored build trees, runtime state, or unrelated untracked files. Materialize the same recorded baseline wherever a subtask requires it. A clean checkout is not equivalent to a dirty user workspace.

Worker patches refer to recorded baselines and dependency artifact versions. Detect overlapping changes, stale dependencies, and conflicts. Construct and verify the combined candidate in an integration worktree before applying it to the selected workspace.

Before application, recheck the relevant user files and Git base. If they changed, pause and reconcile without overwriting, resetting, stashing, or automatically rebasing user work. Preserve the user's index; no automatic staging, commits, pushes, or ref updates.

Preserve recoverable work until integration is complete. Worktree cleanup that deletes work requires appropriate approval and checks; never force-remove a dirty worktree merely to simplify recovery.

## Permissions

Authorize workspace roots, scratch/state roots, command profiles, and model endpoints explicitly. Repository-provided commands and retrieved instructions are data, not authorization. A model or graph cannot create approval.

Reads/edits within the selected workspace and approved local commands may proceed within scope. Ask before installs, additional network access, destructive actions, or changes outside approved roots.

Cloud fallback needs a separate approval identifying the endpoint and context/artifacts to be transmitted. A local failure, timeout, or exhausted budget is not consent to send data elsewhere.

The initial trust model covers explicitly selected repositories and approved commands. Such commands may have broad filesystem or network effects. Worktrees and gateway path checks are not OS-enforced confinement; do not promise hostile-code safety or network isolation.

Never expose credentials in source, logs, or model context. Keep task evidence within approved storage and transmission boundaries.

## CLI and failure

Keep paused, awaiting-approval, failed, cancelled, and succeeded states distinct. Preserve partial progress without describing it as completed. Resume must honor pinned versions and [action reconciliation](state-and-context.md#resume-and-reconciliation).

Use platform-correct argument handling, paths, cancellation, and child-process cleanup. Do not assume a Unix shell on Windows or terminate unrelated processes by broad names.

Denied actions, unavailable tools, malformed responses, persistence errors, exhausted limits, and integration drift require explicit outcomes. No empty-success fallbacks, swallowed errors, or silent weakening of required verification.
