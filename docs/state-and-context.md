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
