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

### Second M1 slice evidence

The host-observation slice was checked on 2026-09-15 from the merged slice-1 baseline `56495dd`, using Go 1.27.1 on Windows/amd64. The user approved the acceptance contract before implementation. Checks used `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, and `CGO_ENABLED=0`; `GOFLAGS`, `GOEXPERIMENT`, and `GOWORK` were empty. No dependencies were added or installed, and no network/model calls were made.

| Check | Observed result |
|---|---|
| Behavioral red | A compiling, unimplemented `ValidateObservation` stub failed all 274 new executions across 17 top-level tests. All 145 existing executions across 24 top-level tests still passed; nothing was skipped. API-absence compilation failures were not used as behavioral-red proof. |
| Frozen green | `go test -count=1 -json .\internal\state` passed all 419 distinct executions across 41 top-level tests: 274 observation cases and 145 existing cases, with no failures or skips. The final check verified the expected run/outcome counts and all frozen artifact hashes, not just the process exit code. |
| Windows checks | `go vet .\internal\state`, `go build .\...`, and `go test -count=1 -cover .\internal\state` passed, with 100.0% statement coverage. `gofmt` checks on the three new Go files passed. |
| Linux cross-compilation | `go test -c` with `GOOS=linux`, `GOARCH=amd64`, and `CGO_ENABLED=0` produced a Linux test binary; `go version -m` confirmed its target and toolchain. The temporary binary was removed. It was not executed, so this is not Linux runtime or Linux-native toolchain evidence. |

The four test files stayed frozen through implementation. The existing proposal implementation, proposal tests, metadata tests, and module file remain byte-for-byte unchanged in this checkout. Hashes below identify the checked working-tree bytes:

| Artifact | SHA-256 |
|---|---|
| `internal/state/observation_test.go` | `1F8F7679061C27B80329A6152FC82BFCE078F4E284F812479FF7723F21E13E63` |
| `internal/state/observation_metadata_test.go` | `07F56A228021B0BBE4BC191FFEEB669D17FAF28F11E83CD746FE004BE4BD7335` |
| `internal/state/observation.go` | `7722E9DB1593BB9AB53D726C1FBD8ECB630091FCFF4D2A78B1B0059886B5DF6D` |

The race detector was not run: this environment had CGO disabled and no `gcc` on PATH. Only Docker Desktop's WSL distribution was available; Linux execution was not attempted. The first slice's historical race/Linux results do not cover the new observation code.

**Linux re-verification, 2026-09-24.** On the `slice-2` branch at `58aca72` with a clean working tree, a Linux/amd64-native environment with Go 1.27.1 (`go1.27.1-X:nodwarf5`) and GCC 16.2.1 re-ran the checks. `GOTOOLCHAIN=local`, `GOPROXY=off`, and `GOSUMDB=off` were set; `GOFLAGS`, `GOEXPERIMENT`, and `GOWORK` were empty; `CGO_ENABLED=0` applied to the regular checks and `CGO_ENABLED=1` to the race check. No dependencies were added or installed, and no network/model calls were made.

| Check | Observed result |
|---|---|
| Build, vet, formatting | `go build ./...` and `go vet ./...` passed; `gofmt -l internal/` reported no files. |
| Native green | `go test -count=1 -json ./internal/state` passed all 419 executions across 41 top-level tests (378 subtests): 274 observation cases and 145 existing cases, with no failures or skips. `-cover` reported 100.0% statement coverage. |
| Race detector | `go test -count=1 -race ./internal/state` with `CGO_ENABLED=1` passed with no races reported. |
| Windows cross-compilation | `go test -c` with `GOOS=windows`, `GOARCH=amd64`, and `CGO_ENABLED=0` produced a Windows test binary; `go version -m` confirmed its target and toolchain; the temporary binary was removed and not executed. |
| Frozen hashes | The three recorded SHA-256 values above matched the working-tree bytes on this checkout. |

This run closes the two recorded gaps for the second slice: Linux runtime execution and the race detector. It adds no new behavioral coverage; the test set and artifacts are unchanged.

This evidence covers deterministic, bounded metadata validation and the test-only catalog fixture. It does not establish evidence integrity, actual tool effects, lifecycle/replay handling, persistence, verification verdicts, completion, or a runnable harness. M1 and the first-release gates remain incomplete.

### Third M1 slice evidence

The action-lifecycle slice was checked on 2026-09-24 on Linux/amd64-native, from the merged slice-2 baseline `fafeeba`, using Go 1.27.1 (`go1.27.1-X:nodwarf5`) and GCC 16.2.1 for the race check. The user approved the acceptance contract, including fully terminal `unknown` and duplicate no-op semantics, before implementation. No dependencies were added or installed, and no network/model calls were made.

| Check | Observed result |
|---|---|
| Behavioral red | A compiling, unimplemented `ApplyOutcome` stub failed all 211 new subtest executions across the 17 new top-level tests in the red file (647 total runs in the package at red time). All 41 existing top-level tests (274 observation and 145 proposal executions) still passed; nothing was skipped. API-absence compilation failures were not used as behavioral-red proof. |
| Post-red test additions | Three approved-contract cases absent from the red file (zero/negative limits; unsupported lifecycle status; terminal conflict at maximum lifecycle revision) were added after the red run and verified red against the original stub before the final freeze. The final test file differs from the red file only by those additions and gofmt whitespace normalization; no existing assertion changed. |
| Green | `go test -count=1 -json -cover ./internal/state` passed all 655 executions across 59 top-level tests: 236 new lifecycle cases and 419 existing cases, with no failures or skips. `-cover` reported 100.0% statement coverage. |
| Race detector | `go test -count=1 -race ./internal/state` with `CGO_ENABLED=1` (GCC 16.2.1) passed with no races reported. |
| Build, vet, formatting | `go build ./...` and `go vet ./...` passed; `gofmt -l internal/` reported no files. |
| Windows cross-compilation | `go test -c` with `GOOS=windows`, `GOARCH=amd64`, and `CGO_ENABLED=0` produced a Windows test binary; `go version -m` confirmed its target and toolchain; the temporary binary was removed and not executed. |

The user approved Linux-only verification for this slice; Windows is cross-compilation evidence only. Hashes below identify the checked working-tree bytes (SHA-256):

| Artifact | SHA-256 |
|---|---|
| `internal/state/lifecycle_test.go` | `3AA9B16DF4AA34C78C537D988229EBB8552A8B0355B395C04317BC8AD3736F7D` |
| `internal/state/lifecycle.go` | `B19E884F1BD34F385E982F58E65D76CE6211ABC4F8E72C8D0D48E2325332AC48` |
| `internal/state/observation.go` | `9879C2A393A459E8B1FF9BA6523822964DD01B75CE86AF2D90BE3B8C77E4E538` |

`observation.go` changed behavior-neutrally in this slice: `validateEvidenceRecord` now takes its sentinel as a parameter and delegates its producer-consistency checks to an extracted helper, preserving the original check order and diagnostics; the frozen observation and proposal suites, byte-for-byte unchanged, prove the preserved behavior. `proposal_test.go`, `metadata_test.go`, `observation_test.go`, `observation_metadata_test.go`, `proposal.go`, and `go.mod` remain byte-for-byte unchanged from the hashes recorded for the earlier slices.

This evidence covers the deterministic action-lifecycle contract and its test-only catalog fixture. It does not establish persistence, reconciliation, verifier verdicts, completion, scheduling, worktrees, a CLI, or a runnable harness. M1 and the first-release gates remain incomplete.

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
