# Research

This assessment distinguishes published mechanisms, source limitations, and hans design proposals. The combined harness and the available local models have not been evaluated by this project.

## Sources

The full primary papers, including relevant methods, results, limitations, and appendices, were reviewed during planning on September 13, 2026.

| Priority | Paper and reviewed version |
|---|---|
| 1 | [SKILL.state: Scalable Long-Horizon Agent Skills](https://arxiv.org/html/2608.26263v3), v3, September 2, 2026. |
| 2 | [TDFlow: Agentic Workflows for Test Driven Development](https://arxiv.org/html/2510.23761v2), v2, January 22, 2026. |
| 3 | [Procedural Graphs: Self-Evolving Execution Structures for LLM Agents](https://arxiv.org/html/2609.09153v1), v1, September 8, 2026. |

ArXiv metadata reports EMNLP acceptance for SKILL.state and EACL 2026 publication for TDFlow. Proceedings records were not independently verified. No acceptance venue was listed in the inspected procedural-graph metadata; treat it as a recent preprint.

## SKILL.state

**Source mechanism:** each invocation receives an immutable procedure, current structured state, and latest observation. The model returns a state patch and action; the runtime validates updates rather than continually replaying a trajectory. See [Section 3](https://arxiv.org/html/2608.26263v3#S3).

`SKILL.state` is an architectural name, not a portable file standard, a mandatory filename, or an established companion-file relationship with `SKILL.md`. The paper uses domain-authored schemas and demonstrates dictionary patches with null-deletion semantics; hans need not adopt those exact semantics.

The source reports substantial token savings in structured simulations. On the 100-step warehouse task, the strong-model configuration reports a 0.94 score and a 16.2-fold token reduction relative to the history-retaining stateful baseline. These are task-specific results, not general coding completion guarantees.

Small-model evidence is cautionary: at 100 steps, SKILL.state scores are 0.42 for the 31B model and 0.34 for the 8B model despite compact prompts. Premature state overwrites/deletions dominate the reported 31B failure taxonomy. See [Section 5.7](https://arxiv.org/html/2608.26263v3#S5.SS7) and [Tables 7-8](https://arxiv.org/html/2608.26263v3#Ax4.T7).

Compactness therefore does not establish state-maintenance correctness. Constrained decoding can address syntax but cannot by itself prove a deletion or asserted fact is correct.

The paper identifies limitations when the state schema is not known, discarded observations become relevant later, or a task depends on historical trajectory. Concurrent state writing is not established by its single-agent evaluation. Public customer-service results provide some non-coding evidence, but not corresponding small-local-model validation. See [Section 7](https://arxiv.org/html/2608.26263v3#S7).

Prompt characters and tokens are not interchangeable, and parts of the source use inconsistent units. Do not derive hans's input budget from its reported character counts.

**hans adaptation:** bounded schema values, host-owned confirmed outcomes, scoped revisions, recoverable evidence, and concurrent coordination. These additions address practical gaps but require their own evaluation.

## TDFlow

**Source mechanism:** separate test generation, repository exploration, patch revision, and focused debugging roles. The host applies patches and executes checks; selected failure evidence guides repair. See [Section 3](https://arxiv.org/html/2510.23761v2#S3).

The strongest experiments expose trustworthy benchmark reproduction tests and use frontier models. The Verified results table reports 94.3% with human tests and 68.0% with generated tests; 45 of 500 Verified instances were excluded because of execution/debugging requirements. These are not equivalent to standard issue-only benchmark results or small-model performance. See [Section 4](https://arxiv.org/html/2510.23761v2#S4) and [Appendix B](https://arxiv.org/html/2510.23761v2#A2).

The introduction's generated-test percentage differs from the results table; use the table's 68.0% when discussing that experiment. No small local open-weight model was evaluated.

Generated-test quality is a major bottleneck. The paper can retrospectively check whether tests fail before and pass after a known gold patch. A novel user task does not supply that oracle. A test may be red for an irrelevant reason or agree with an implementation about the wrong requirement.

TDFlow protects provided/generated reproduction tests from modification during repair. It also acknowledges incorrect frozen tests, missing early-stop/correction mechanisms, and residual test hacking identified through manual review. Protection of test files alone is not complete automatic anti-gaming enforcement. See [Section 4.4](https://arxiv.org/html/2510.23761v2#S4.SS4), [Section 5](https://arxiv.org/html/2510.23761v2#S5), and [Appendix C](https://arxiv.org/html/2510.23761v2#A3).

Its proposal context includes previous attempted patches and reports, so role separation does not establish history-independent input size.

**hans adaptation:** test-first development as the user's policy; requirement review, meaningful red, protected test/verifier artifacts, bounded repair, explicit test-correction/waiver paths, and combined-result verification. Do not copy the growing patch-history strategy or claim the paper proves universal autonomous TDD reliability.

## Procedural graphs

**Source mechanism:** directed attributed graphs represent procedural knowledge. A guidance model receives a neighborhood around the current procedure and recent trajectory information, then advises a solver that still retains trajectory context. The graph is frozen during the episode. See [Section 3](https://arxiv.org/html/2609.09153v1#S3).

Graph edges carry conditions, guidance, and pitfalls. This is advisory execution knowledge, not a hard state-machine authorization mechanism. The published retrieval can fall back to the full graph when matching fails.

Evolution occurs between evaluated batches: propose edits, structurally validate a candidate, score it, and retain candidates whose validation score equals or exceeds a cached incumbent. The generic terminal/reachability checks and prompt-level tool constraints are not sufficient guarantees of safe execution. See [Appendix B.6](https://arxiv.org/html/2609.09153v1#A2.SS6).

Reported evaluations cover multiple coding-adjacent and non-coding domains, using frontier models rather than small local open-weight models. Solver, guide, and refiner use the same underlying model in each configuration; transfer to a cheaper local guide or a different solver is not established.

Localized guidance is cheaper than full-graph generative guidance, but can still increase total inference tokens relative to no graph. Fewer solver steps do not necessarily mean lower cost. A poorly chosen procedure can also reduce success. See [Section 5.3](https://arxiv.org/html/2609.09153v1#S5.SS3) and [Section 5.5](https://arxiv.org/html/2609.09153v1#S5.SS5).

Tied noisy validation scores and repeated reuse of a validation set do not establish general improvement. Evolution requires meaningful external scorers, held-out tasks, and a budget for candidate evaluation.

**hans adaptation:** small reviewed, version-pinned advice with bounded retrieval and no full-graph fallback. Evaluate deterministic rendering before extra model calls. Add stronger protected-invariant, evaluation, cost, and user-promotion gates for later between-run evolution.

## Combined design

Use explicit state for current work, external evidence for recoverable history, protected verifiers for coding outcomes, and optional graphs for reusable advice.

This combination changes each source's assumptions. In particular, replacing the graph paper's trajectory inputs with bounded state is a new hybrid; adding archived evidence and concurrent coordination goes beyond the state paper's evaluated setup.

The practical order is state and evidence first, TDD and integration second, reviewed guidance third, evaluated evolution last. See the [roadmap](roadmap.md) and [evaluation protocol](evaluation.md).

## Evidence policy

Keep source findings, implementation choices, and hypotheses explicitly labeled. Record versions, datasets, model conditions, exclusions, and measurement units when citing results.

No combined-system experiment, quantized-local-serving result, or hardware-sizing result was established by these sources. Approximately 10B/30B availability does not identify actual capability, context accounting, throughput, or task reliability.

Do not claim replication or deployed correctness from paper results, valid JSON, compact prompts, mocked tests, or a single successful task. Evaluate the actual configured profiles and report uncertainty, failures, and complete inference cost.
