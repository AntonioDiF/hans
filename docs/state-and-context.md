# State and context

Adopt the explicit-state principle of [SKILL.state](research.md#skillstate), not a literal required file format. The design below extends that principle for recoverable evidence, verified effects, and concurrent workers.

## Invocation contract

Each invocation receives the immutable procedure and mandatory instructions, its scoped task, current bounded state, required schemas, and selected observations/evidence. The model proposes an allowed state update and an action. The host validates them before mutation or execution.

Do not replay the conversation by default. Do not require verbose reasoning transcripts as persistent memory. Keep concise decision rationale where useful, and archive evidence outside the prompt.

Historical evidence, current execution state, and reusable procedural advice are different resources. Retrieved history is not automatically current truth, and a procedural graph must not duplicate mutable task facts.

## Input budget

The initial per-call input ceiling is **8,192 tokens**, representing the selected 8k budget. It is configurable per model and applies to all roles, including planning, review, recovery, and later graph guidance/refinement.

Count the complete request: system and repository instructions, selected documentation, task, state, tools, output schemas, observations, retrieved evidence, and chat-template framing. Reserve generated output and any provider-counted reasoning separately. The request and reservation must fit the endpoint's usable context.

Enforcement requires compatible tokenization/templates for the complete request or a verified server input limit that rejects oversized requests without silently truncating them. An advertised context window, character estimate, response-only `max_tokens`, or post-response usage count is insufficient.

Block profiles that cannot meet this contract. Do not label a conservative estimate as a hard ceiling. Record the enforcement mechanism and actual capability evidence in the profile.

When composing input, select only relevant instructions, state fields, tools, and evidence slices. Archive large observations with bounded excerpts and handles. Never automatically include an entire output because it is the latest one.

If mandatory instructions, acceptance criteria, and required facts cannot fit, report a capacity problem and pause or re-scope. Do not silently discard them. Any compacting/recovery call also has to fit its own limit and consume the applicable budget.

Bound strings, collections, hypotheses, evidence selections, and observations. A schema with fixed field names does not establish constant context if its values can grow indefinitely.

## State ownership

Use a common typed envelope and a workflow-specific schema. Schema versions and field limits are explicit.

| State category | Authority and rule |
|---|---|
| Run/worker identity and revisions | Host-owned; models cannot assign or rewind them. |
| Requirements, acceptance and procedure versions | Pinned to the run or explicit authorized revision boundary. |
| Dependencies, phases, budgets, permissions | Host-owned and updated through validated transitions. |
| Workspace and artifact references | Bound to recorded baselines, hashes/versions, and approved scopes. |
| Action intent and outcome | Separate records; only observed outcomes establish effects. |
| Test/verifier verdicts and completion | Host-observed, bound to exact artifacts and verifier versions. |
| Hypotheses and bounded planning detail | Models may propose changes within schema, ownership, and evidence constraints. |
| Observed domain facts | Preserve source/version provenance; distinguish them from hypotheses. |

Each worker maintains private versioned state. The coordinator is the logical writer of shared run state. Scope revisions to the state and dependencies a proposal actually reads, while serializing authoritative shared commits.

Workers submit scoped proposals with expected revisions and evidence references. Do not let agents edit a shared JSON object or free-form summary concurrently. SQLite transactions alone do not resolve stale semantic assumptions.

## State transitions

Define typed patch operations and field ownership. Do not import unrestricted null-deletion or whole-object replacement from a paper example. Reject unknown fields, invalid types, unauthorized deletions, stale revisions, and violations of domain invariants.

Validate the proposal and accompanying action before either takes effect. An invalid proposal must not execute its action. A syntactically valid model assertion still does not prove a fact.

Persist approved action intent before tool execution. Model-proposed hypotheses may be retained as hypotheses, but the proposed action cannot create confirmed effects before its outcome is observed.

Record the tool result and commit only supported outcome transitions. Verification results belong to the exact source, tests, command configuration, and relevant environment that produced them. Later changes invalidate affected verdicts.

A model-written `done` field or successful-looking log line is never sufficient to advance completion.

### Implemented proposal boundary

The first M1 slice is `internal/state`, a pure Go boundary, not a state store or universal workflow framework. Schema version 1 contains a worker identity, host revision, plan text, and a bounded collection of hypotheses. The test-only catalog-review fixture exercises this contract without source-code or command assumptions.

| Surface | Implemented rule |
|---|---|
| Identity and schema | The host supplies the current state and an independent `Scope`. The proposal echoes run/worker identity and schema version; mismatches are rejected, not assigned to the state. Only `CurrentSchemaVersion` (1) is supported. |
| Revision | `ExpectedRevision` must equal the current worker revision. A valid candidate advances it once, including action-only proposals. Zero is a valid initial revision; an exhausted `uint64` revision fails explicitly. |
| Updates | Ordered `SetPlan` and `AppendHypothesis` operations only. Plan updates contain text alone; each appended hypothesis cites one exact host-approved evidence ID/version. Unknown kinds, deletion, whole-state replacement, and writes to facts, permissions, verification, or completion are not supported. |
| Action intent | The optional `InspectEvidence` intent must be allowed by the host scope and target an approved evidence ID/version. It contains no outcome or success field. Matching a reference does not prove a hypothesis or verify evidence contents. |
| Limits | `MaxUpdates`, `MaxPlanBytes`, `MaxHypothesisBytes`, and `MaxHypotheses` are explicit positive integers; zero and negative values are invalid configuration. Exact limits are accepted, excess is rejected without truncation. Text limits count UTF-8 bytes, not context tokens. |

Proposed text must be valid, nonblank UTF-8; empty text cannot serve as a deletion. A host state may start with an empty plan and no hypotheses. Existing state must already satisfy the supplied schema, scope, and limits; validation never repairs or silently drops it.

`Validate(current, scope, limits, proposal)` validates all updates and the accompanying action together. Rejection returns a nil result and an explicit error, leaving every input unchanged, including slice backing storage. Success returns a detached `ValidationResult` containing candidate `Next` state and optional `Action` intent; it does not modify the authoritative state or execute anything. Callers can use `errors.Is` with the exported error sentinels; messages identify the failing field or operation, and stale-revision errors include both revisions.

The host must construct scope independently of model output, serialize authoritative commits, and recheck revisions and authorization when applying a candidate. Facts and their provenance, permissions, verifier results, and completion remain outside the model-writable state. An assertion retained in a hypothesis is still only a hypothesis.

There is no wire decoder: Go field types constrain values, and runtime validation rejects unsupported operation values. Strict model-response decoding, general workflow adapters, dependency revisions, durable action intent, observations, persistence, execution, and recovery remain subsequent work. This candidate-only contract does not yet provide an atomic persisted transition or replay protection.

## Evidence and persistence

Use SQLite for state, event metadata, dependencies, approvals, action lifecycle, and artifact references. Store large tool outputs, test reports, and artifacts in files with stable identifiers and integrity metadata.

Use approved storage roots outside source-controlled inputs so runtime files do not contaminate worktree baselines. Select a maintained SQLite driver with Windows/Linux coverage and document the build/CGO trade-off before implementation.

SQLite cannot atomically commit an external action or ordinary filesystem write. Define an evidence write, flush, and finalize protocol before committing database references. Inject failures around these boundaries in tests.

Missing referenced evidence, corrupt state, failed writes, and incompatible schemas must be explicit errors. Detect orphan evidence files without silently deleting the only recoverable output. Never reinterpret unavailable evidence as an empty successful result.

Retrieval uses scoped identifiers, metadata, and bounded file ranges. Preserve source/version information and whether an excerpt is partial. Retrieval does not require an embedding store in the first release.

Keep secrets out of model context and routine logs. Record usage, actions, and necessary evidence without blindly archiving every reasoning field or sensitive payload.

## Resume and reconciliation

Pin workflow, schema, model profile, policy, acceptance/verifier, and optional graph versions for a run. Requirement changes and incompatible resume configurations create explicit replan or migration boundaries.

Reject competing coordinators for the same run. Persist enough information to resume the coordinator and workers without replaying their full histories.

After interruption, distinguish completed, failed, pending, and uncertain actions. Reconcile uncertain actions before proceeding. A missing outcome record does not prove that execution never happened.

Do not blindly repeat commands or external side effects unless idempotence is established or the user authorizes a retry. Do not claim exactly-once external effects merely because SQLite transactions are used.

Cancellation and late responses must respect run/worker revisions and action identity. Discard stale proposals without misrepresenting any real side effects or consumed inference budget.

Finite storage and retention policies must preserve pending-action evidence and unintegrated work. Pause when safe persistence cannot continue; destructive cleanup requires appropriate approval.
