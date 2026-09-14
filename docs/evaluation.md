# Evaluation

These are future acceptance protocols except where an executed, bounded slice is explicitly reported below. Keep deterministic runtime correctness, endpoint capability calibration, and empirical model performance separate.

Routine tests use deterministic model-server and tool fixtures. Live inference is an explicit, finite-budget evaluation, not an implicit network dependency of unit tests.

## Deterministic contracts

| Area | Required cases and observable outcome |
|---|---|
| Input budget | Count the full request. Cover 8,191, 8,192, and 8,193 input tokens under a compatible profile with sufficient output capacity; reject over-budget calls. Include framing, schemas, output reservations, mandatory instructions, oversized observations, and unsupported enforcement. |
| State | Reject invalid types, protected deletions, fabricated outcomes, stale revisions, and conflicting proposals without executing the accompanying action. Preserve fact provenance. |
| Persistence | Inject interruptions around action intent, tool completion, evidence writes/finalization, and SQLite commits. Missing/corrupt evidence must be explicit; no false completion or blind side-effect replay. |
| Resume | Restore pinned versions and worker state; reject competing coordinators; reconcile uncertain actions and late responses. Test incompatible requirements/configuration. |
| Concurrency | Use blocking fixtures to prove at least two workers overlap, rather than inferring concurrency from logs. Assert observed model/endpoint/shared-service/global maxima remain within configured bounds. |
| Scheduling | Cover saturated queues, dependency failure, fairness, cancellation, late output, and exhausted budgets. A new worker must not reset run limits. |
| Worktrees | Cover clean and dirty baselines, staged/unstaged inputs, approved untracked files, deletions/binaries, supported modes/links, conflicts, user edits, unsupported inputs, and no-commit repositories. Preserve the user index and branch. |
| TDD | Demonstrate relevant red before production changes, reviewed/frozen tests, green on the exact artifact, regression checks, and an explicit waiver path. |
| Anti-gaming | Detect protected-test/fixture/config edits, deselection/skipping, missing expected tests, empty success reports, and stale evidence. |
| Integration | Two worker-local successes can yield a failing combined candidate. Withhold success and application until combined verification passes. |
| Policy | Unauthorized commands, outside-root actions, installs, network access, and cloud transmission pause for approval. Model/graph content cannot grant approval. |
| Platforms | Exercise builds, relevant Go checks, paths, process cancellation/cleanup, worktrees, and persistence on Windows and Linux. |
| Extension boundary | A minimal non-coding fixture supplies domain state, actions, artifacts, and verification without coding assumptions leaking into the core. |

Use fixtures with independently specified expected outcomes. Mocks prove contracts, not model intelligence, prompt quality, or production compatibility with a named server.

### First M1 slice evidence

The proposal-validation slice and its metadata-hardening follow-up were checked on 2026-09-14 with Go 1.25.6. The initial boundary was checkpointed at `34af995`; the results below cover the updated validator. Go commands used the local toolchain with `GOPROXY=off` and `GOSUMDB=off`; these checks installed no dependencies and made no network/model calls.

| Check | Observed result |
|---|---|
| Behavioral red | The initial compiling, unimplemented validator failed all 100 executions. For the metadata follow-up, declarations without enforcement produced 145 executions: 122 passed and 23 failed. Failures reproduced accepted oversized identifiers, including 1 MiB evidence IDs, ignored metadata configuration limits, and oversized scope collections. Rollover regressions already passed against existing behavior. |
| Green and refactor | `go test -count=1 -json -cover .\...` on Windows/amd64: 24 top-level tests, 145 executions including subtests, all passed, no skips, 100.0% statement coverage. Frozen test hashes were unchanged after the metadata red; the original assertions remain intact. The fixture supplies the two new required limits, and rejection diagnostics no longer dump an arbitrarily large candidate. |
| Windows checks | `go test -race -count=1 .\...`, `go vet .\...`, `go build .\...` with `CGO_ENABLED=0`, and `gofmt` checks passed. The race build used GCC 13.2.0 targeting `x86_64-w64-mingw32`. |
| Linux check | Cross-compiled the test binary for Linux/amd64 with `CGO_ENABLED=0`, then actually executed it in Arch Linux under WSL2, kernel `6.6.87.2-microsoft-standard-WSL2`. The same 24 top-level tests and 145 executions passed with no skips. This was not a Linux-native Go build, vet run, or race-detector run. |

The frozen `internal/state/proposal_test.go` SHA-256 is `0AC5E0E0134D4CA13AE41E6DD4406F3C86BD535B0DDF18E13EDDD3803A2F99F6`, and `internal/state/metadata_test.go` is `3920B0D4B16FAFF2EE09814928D77A8730026939964227D1B5D234760A5AD6DB`. The checked `internal/state/proposal.go` SHA-256 is `1A3DE663F20DFBCE52D9A8A87BBEEBBAC225F84743624D023E42D6468A795534`. The module is unchanged from the initial checkpoint.

API-absence runs could not compile and were not used as behavioral-red proof. During the initial slice, a WSL invocation failed before collecting tests because PowerShell split unquoted test flags; quoting `-test.v=test2json` and `-test.count=1` corrected the runner invocation without changing tests. The follow-up used those quoted flags.

This evidence covers only the pure proposal boundary and its test-only non-coding fixture. It does not complete M1 or any first-release gate, demonstrate a production workflow, or validate persistence, model calls, token accounting, commands, scheduling, worktrees, a CLI, procedural graphs, or end-to-end reliability.

## Go acceptance fixture

Create a disposable, committed Go repository with two independent, explicitly specified coding changes and a combined integration surface. Keep requirements visible; held-out tests may check those requirements but must not introduce hidden requirements.

The run must plan both subtasks, establish reviewed red tests, dispatch isolated worktrees, respect model limits, integrate patches, and verify combined behavior and regression checks.

Include variants with dirty inputs, a conflicting or integration-failing pair of patches, and pause/resume. Ensure integration failure is observable even when worker-local tests pass.

Tests may create commits in their own disposable fixture repositories. This does not authorize automatic commits in user repositories.

The first-release gate requires concurrent execution, faithful Git baselines, strict input budgets, SQLite-backed resume, and protected TDD together. A serial demo or individually passing workers is not a substitute.

## Long-horizon scenario

Create a 100-step state-and-recovery fixture with known expected state, superseding observations, pending/failed actions, stale proposals, and a pause/resume point. Require exact scripted-state correctness and budget adherence across the full scenario.

Then run model-driven variants and report full-task success separately from step-level scores. Include shorter horizon observations where useful to identify degradation rather than inspecting only the final state.

Use the scenario to test preservation of relevant facts and corrections, not merely that prompts stayed small. External evidence retrieval must also be measured; ignored or lost observations cannot be hidden by a token metric.

Follow synthetic evaluation with real multi-subtask coding tasks. Report the two categories separately; synthetic state scores are not coding completion rates.

## Model calibration

Record actual approximately 10B and approximately 30B models, model/server/tokenizer versions, quantization where applicable, prompt formatting, structured-output capabilities, and usable context.

Verify trustworthy input-limit enforcement before enabling a profile. Match tokenizer/template behavior or verify server rejection without silent truncation. Post-response usage can detect discrepancies but cannot retroactively make an oversized request compliant.

Choose concrete finite concurrency counts, shared-service limits, retry/no-progress counts, timeouts, run/worker budgets, output/reasoning reservations, field limits, and storage limits during calibration. No unlimited values.

Evaluate each model individually and the intended role-routed configuration. Compare bounded state with a clearly specified capped-history baseline under comparable conditions. Include 8k and smaller budgets, including 4k where supported.

Control task selection, tool interfaces, verifier, routing, and concurrency when comparing memory strategies. If a comparison changes more than one factor, state that explicitly rather than attributing all gains to state management.

Define sample counts, datasets/splits, exclusions, quality thresholds, and resource ceilings before the corresponding held-out evaluation. Do not tune promotion thresholds after seeing favorable results.

## Graph evaluation

After first-release evaluation, compare no graph with reviewed compact guidance on the same profiles and tasks. Do not begin by assuming a separate guidance-model call is worthwhile.

For evolved candidates, preserve separate training/validation/held-out sets and paired or repeated comparisons where needed. Record the actual candidate version and protect test/scoring/policy artifacts.

Require no protected-invariant failures and no unacceptable per-category regression. Predeclare a meaningful quality improvement, or a justified non-inferiority threshold with a measured cost benefit. A noisy tie is not automatic evidence of improvement.

Include graph retrieval, guidance generation, refinement, failed candidates, and evaluation runs in cost accounting. Promotion still requires user approval; see [promotion policy](procedural-graphs.md#promotion).

## Reporting

Record task completion, step-level accuracy, state loss/overwrite, invalid proposals, repeated actions, test validity, integration regressions, and human interventions.

Record total input/output tokens, peak input size, retries, model/tool calls, guidance/refinement work, elapsed time, and concurrency. Do not hide review or recovery overhead by reporting only solver steps.

Bind reports to artifact, model, tokenizer, procedure, graph, verifier, dataset, and environment versions. Include exclusions, seeds where relevant, uncertainty, and failed runs.

No local-model success-rate guarantee exists yet. Documentation must distinguish planned gates, executed checks, and measured performance. A single successful demo does not establish general long-horizon reliability.
