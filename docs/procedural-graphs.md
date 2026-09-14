# Procedural graphs

**Deferred capability:** establish bounded state, concurrent coding execution, and protected TDD before introducing graphs. Begin with reviewed advice, then evaluate between-run evolution. This design adapts the [procedural-graph paper](research.md#procedural-graphs); it does not claim to reproduce its results.

## Authority

A procedural graph supplies reusable advice. It does not define permissions, current facts, verifier authority, or completion. The host continues to enforce workflow phases, resource limits, approvals, and acceptance criteria.

The scheduler's task-dependency graph is a different structure. Do not make first-release concurrency depend on implementing learned procedural graphs.

Pin guidance versions for each run. A graph cannot grant a tool, bypass TDD, declare a test passed, modify scoring code, waive an approval, or update itself during active execution.

## Reviewed guidance

Use a small versioned data schema with stable node/tool identifiers and bounded descriptions. Edges express relevant transitions with conditions, guidance, and pitfalls. Conditions are advisory content, not authorization predicates.

Validate node/relation types, tool identifiers, endpoints, size limits, and the declared cycle policy. Avoid executable snippets or arbitrary code loading through graph data.

Select a bounded neighborhood using the current procedure or tool identity and render it compactly. Limit traversal and text before composing the model request; graph content shares the existing input ceiling rather than receiving additional context.

If no node matches, emit an explicit no-guidance result or a required-workflow error. Do not silently inject the full graph. Record omissions or unavailable guidance without pretending a match occurred.

Compare no graph with deterministic compact guidance first. A separate generative-guidance call is a later optional experiment only if quality gains justify total tokens, latency, and complexity.

Keep current observations, task state, tool definitions, permissions, and protected tests in their authoritative stores. Graph advice may point to checks but cannot duplicate their mutable outcomes.

## Between-run evolution

After reviewed graphs and reliable evaluation exist, permit a refiner to propose bounded data-only edits from selected scored evidence. Apply proposals to a candidate version, never to an active run's graph.

Record the parent graph, evidence provenance, proposed edits, actual candidate, evaluation protocol, results, and rejection/promotion decision. Bound candidate size, refinement rounds, inference, evaluation work, and retained prompt material.

Validate meaningful start and success/failure terminals, reachability, known tools, supported relations, and bounded cycles/retries. A generic zero-out-degree node is not automatically a legitimate success terminal.

Reject invalid candidates explicitly. If an authorized repair produces a different candidate, validate and evaluate that actual version; do not promote a silently repaired graph under the original candidate's result.

The refiner cannot modify permissions, mandatory workflow phases, approval rules, acceptance tests, scoring code, or budget enforcement. Rejected candidates may inform later proposals through bounded references, not an ever-growing prompt transcript.

## Promotion

Separate training tasks, validation tasks, and untouched held-out evaluation. Predeclare quality, regression, cost, and uncertainty gates before observing candidate results.

No protected-invariant failure is acceptable in exchange for a higher average score. Do not promote on a candidate's explanation of success or on tied noisy scores against a cached incumbent.

Use paired/repeated runs where stochasticity matters. Evaluate each supported local model profile; do not assume graphs transfer between models, quantizations, or tool interfaces. Count guidance/refinement/evaluation costs, not just solver steps.

The harness may propose and evaluate candidates, but **the user approves promotion initially**. Preserve a known-good version and an explicit rollback path. Promotion affects new runs unless the user authorizes a version migration.

See [graph evaluation](evaluation.md#graph-evaluation) for required reporting. Autonomous promotion and in-run graph mutation are outside the initial scope.
