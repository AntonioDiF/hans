# Coding workflow

TDD is the default for behavioral coding changes, including development of hans, unless the user explicitly waives it. The policy is motivated by [TDFlow](research.md#tdflow), but its correctness depends on trustworthy requirements and verification, not the name of the workflow.

The model roles below describe the future hans runtime. Agents developing hans must follow test-first and evidence rules with available trusted tools; they must not pretend that the harness or an unavailable local-model reviewer already ran.

## Phases

| Phase | Required action and exit gate |
|---|---|
| Contract | Clarify material ambiguity; record requirements, acceptance criteria, baseline, repository instructions, and approved command profile. Approximately 30B planning identifies bounded independent subtasks. |
| Baseline | Run relevant approved checks before changes. Record existing failures separately; do not silently repair unrelated defects or claim the suite is green. |
| Test authoring | Write requirement-grounded candidate tests in a distinct authoring phase. Approximately 10B execution may author them in an isolated workspace. |
| Test review | Approximately 30B review checks expected behavior, assertion meaning, boundary cases, and requirement traceability. Review alone is not a correctness oracle. |
| Red | Observe a failure on the baseline for the intended behavioral reason. Preserve the exact test and failure evidence. |
| Freeze | Pin approved tests, required test selection, verifier settings, and relevant environment/version information for the implementation attempt. Preserve protected regressions. |
| Solve | Approximately 10B execution implements the smallest scoped change using bounded code slices and selected failure evidence. |
| Green and refactor | Run required tests on the exact candidate, verify they actually executed, and refactor without weakening the oracle. |
| Integrate | Combine worker results in an integration worktree and run required integration/regression checks. Individual worker success is insufficient. |
| Complete | Apply only the verified combined result to a still-compatible workspace. Host-observed evidence determines completion. |

Use a language-neutral command/verifier contract with Go as the first fully exercised profile. Commands, working directories, selected tests, and environment assumptions must be explicit rather than guessed from a project's language.

## Trustworthy tests

Every acceptance test must trace to a requirement and assert concrete expected behavior. Prefer existing trusted tests and simple independent expected values; add boundary, property, metamorphic, or selected mutation checks where they materially improve confidence.

An unrelated dependency, fixture, import, runner, or syntax failure does not establish meaningful red. An intentionally absent required API can be a valid red signal when that absence is the requirement being tested; record the reason rather than treating all compilation failures alike.

Avoid tests that merely restate the implementation or compute expected values through the same path. Different prompts, roles, or model families do not guarantee independent reasoning.

Baseline failures remain visible. Do not turn unchanged pre-existing failures into a claim of a fully green suite, or fix them without scope. If they block a required gate, pause for an explicit scoped decision.

## Protected verification

Freeze an acceptance manifest containing test artifacts, versions/hashes, required selection, verifier configuration, and red evidence. The solver cannot revise its own oracle inside the repair loop.

Protect against modifications to assertions, test sources, fixtures, runner configuration, environment/dependency assumptions, and skip/deselection behavior that would weaken accepted verification. Additional authoritative tests pass through authoring, review, red, and freeze rather than appearing as unreviewed proof.

A zero exit status or printed success string is not sufficient. Track expected test execution and relevant outcomes; detect empty runs, skipped required tests, collection changes, and stale reports. Distinguish infrastructure failure from a behavioral test failure.

Bind verdicts to the exact source, tests, command profile, and relevant environment. Changes after verification invalidate affected evidence. Run focused checks during repair, and justified integration/regression checks on the combined candidate.

These controls reduce obvious verifier manipulation; they do not prove semantic correctness or provide hostile-code isolation. Keep held-out evaluation and the [trusted-command limitation](architecture.md#permissions) distinct from the production workflow.

## Recovery and waivers

Use finite repair/no-progress limits. After bounded attempts, permit one approximately 30B recovery attempt, then pause for the user. Do not reset an exhausted budget by creating another worker or requesting cloud fallback.

Keep debugging context focused on a selected failure, relevant artifacts, and concise evidence. Do not accumulate all previous patches and reports in every prompt.

If a frozen test appears wrong or unsatisfiable, leave the implementation loop and escalate with evidence. A user-authorized correction creates a new acceptance version and restarts affected red/green checks. Never quietly weaken or delete the test to obtain success.

If practical automated coverage is unavailable for a behavioral change, ask for a scoped TDD waiver. Record the decision, affected requirement, reason, alternative verification, and limitations. A waiver is not permission to claim checks ran when they did not.

Pure documentation changes do not require invented executable tests. Use applicable documentation checks where available. User-requested workflow changes remain explicit and scoped rather than becoming permanent defaults.
