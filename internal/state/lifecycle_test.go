package state_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/AntonioDiF/hans/internal/state"
)

type lifecycleFixture struct {
	action  state.ActionState
	outcome state.ActionOutcome
	limits  state.ObservationLimits
}

func newLifecycleFixture() lifecycleFixture {
	accepted := state.AcceptedAction{
		SchemaVersion: 1,
		Identity:      state.Identity{RunID: "catalog-run", WorkerID: "reader-1"},
		ID:            "inspect-note",
		Revision:      8,
		Intent: state.ActionIntent{
			Kind:     state.InspectEvidence,
			Evidence: state.EvidenceRef{ID: "acquisition-note", Version: "v1"},
		},
	}
	action := state.ActionState{
		SchemaVersion:    1,
		Identity:         accepted.Identity,
		ActionID:         accepted.ID,
		SnapshotRevision: accepted.Revision,
		Status:           state.ActionPending,
		Revision:         0,
	}
	outcome := state.ActionOutcome{
		Action: accepted,
		Status: state.OutcomeSucceeded,
		Evidence: state.EvidenceRecord{
			SchemaVersion: 1,
			Ref:           state.EvidenceRef{ID: "note-output", Version: "v1"},
			Producer:      accepted,
		},
		Detail: "Read the acquisition note.",
	}
	return lifecycleFixture{
		action:  action,
		outcome: outcome,
		limits:  state.ObservationLimits{MaxIdentifierBytes: 64, MaxDetailBytes: 128},
	}
}

// terminalState derives the state a pending fixture reaches after its
// outcome's status was first applied, so tests can exercise terminal rules.
func terminalState(f lifecycleFixture, status state.ActionStatus) state.ActionState {
	s := f.action
	s.Status = status
	s.LastEvidence = f.outcome.Evidence.Ref
	s.Revision = 1
	return s
}

func checkApply(t *testing.T, f lifecycleFixture, wantErr error, want state.ActionState, haveWant bool, paths ...string) *state.ActionState {
	t.Helper()
	before := f
	got, err := state.ApplyOutcome(f.action, f.outcome, f.limits)
	if !reflect.DeepEqual(f, before) {
		t.Error("ApplyOutcome mutated its inputs")
	}
	if wantErr != nil {
		if err == nil {
			t.Fatalf("ApplyOutcome succeeded, want %v", wantErr)
		}
		if !errors.Is(err, wantErr) {
			t.Errorf("error = %v, want errors.Is(_, %v)", err, wantErr)
		}
		if len(paths) > 0 {
			matched := false
			for _, path := range paths {
				if strings.Contains(err.Error(), path) {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("error %q does not identify any of %q", err, paths)
			}
		}
		if got != nil {
			t.Error("rejected outcome returned a partial action state")
		}
		return nil
	}
	if err != nil {
		t.Fatalf("valid outcome rejected: %v", err)
	}
	if got == nil {
		t.Fatal("valid outcome returned no action state")
	}
	if haveWant && *got != want {
		t.Fatalf("action state = %#v, want %#v", *got, want)
	}
	return got
}

func TestApplyOutcomeRejectsInvalidLimits(t *testing.T) {
	for _, field := range []struct {
		name string
		set  func(*state.ObservationLimits, int)
	}{
		{"max_identifier_bytes", func(l *state.ObservationLimits, n int) { l.MaxIdentifierBytes = n }},
		{"max_detail_bytes", func(l *state.ObservationLimits, n int) { l.MaxDetailBytes = n }},
	} {
		for _, value := range []struct {
			name string
			n    int
		}{{"zero", 0}, {"negative", -1}} {
			t.Run(field.name+"/"+value.name, func(t *testing.T) {
				f := newLifecycleFixture()
				field.set(&f.limits, value.n)
				checkApply(t, f, state.ErrInvalidLimits, state.ActionState{}, false, "limits."+field.name)
			})
		}
	}
}

func TestApplyOutcomePendingTransitions(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status state.OutcomeStatus
		next   state.ActionStatus
		detail string
	}{
		{"succeeded", state.OutcomeSucceeded, state.ActionSucceeded, "Read the acquisition note."},
		{"failed", state.OutcomeFailed, state.ActionFailed, "The evidence source could not be read."},
		{"unknown", state.OutcomeUnknown, state.ActionUnknown, "The host lost contact before observing the result."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture()
			f.outcome.Status = tt.status
			f.outcome.Detail = tt.detail
			before := f.action
			want := f.action
			want.Status = tt.next
			want.LastEvidence = f.outcome.Evidence.Ref
			want.Revision = 1
			got := checkApply(t, f, nil, want, true)
			got.Status = state.ActionPending
			got.LastEvidence = state.EvidenceRef{ID: "changed", Version: "changed"}
			got.Revision = 0
			if f.action != before {
				t.Fatal("mutating the returned state changed the input action state")
			}
		})
	}
}

func TestApplyOutcomeDuplicateIsNoOp(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status state.ActionStatus
		obs    state.OutcomeStatus
	}{
		{"succeeded", state.ActionSucceeded, state.OutcomeSucceeded},
		{"failed", state.ActionFailed, state.OutcomeFailed},
		{"unknown", state.ActionUnknown, state.OutcomeUnknown},
	} {
		for _, revision := range []uint64{1, 5} {
			t.Run(fmt.Sprintf("%s/revision-%d", tt.name, revision), func(t *testing.T) {
				f := newLifecycleFixture()
				f.action = terminalState(f, tt.status)
				f.action.Revision = revision
				f.outcome.Status = tt.obs
				got := checkApply(t, f, nil, f.action, true)
				if got.Revision != revision {
					t.Fatalf("duplicate advanced the lifecycle revision to %d", got.Revision)
				}
			})
		}
	}
}

func TestApplyOutcomeRejectsConflictingObservations(t *testing.T) {
	pairs := []struct {
		terminal state.ActionStatus
		observed state.OutcomeStatus
	}{
		{state.ActionSucceeded, state.OutcomeSucceeded},
		{state.ActionSucceeded, state.OutcomeFailed},
		{state.ActionSucceeded, state.OutcomeUnknown},
		{state.ActionFailed, state.OutcomeSucceeded},
		{state.ActionFailed, state.OutcomeFailed},
		{state.ActionFailed, state.OutcomeUnknown},
		{state.ActionUnknown, state.OutcomeSucceeded},
		{state.ActionUnknown, state.OutcomeFailed},
		{state.ActionUnknown, state.OutcomeUnknown},
	}
	for _, pair := range pairs {
		for _, ref := range []string{"same", "different"} {
			if ref == "same" && string(pair.terminal) == string(pair.observed) {
				continue // exact duplicate, covered by TestApplyOutcomeDuplicateIsNoOp
			}
			name := fmt.Sprintf("%s-to-%s/%s-ref", pair.terminal, pair.observed, ref)
			t.Run(name, func(t *testing.T) {
				f := newLifecycleFixture()
				f.action = terminalState(f, pair.terminal)
				f.outcome.Status = pair.observed
				if ref == "different" {
					f.outcome.Evidence.Ref.Version = "v2"
				}
				checkApply(t, f, state.ErrConflictingObservation, state.ActionState{}, false, "conflicting observation")
			})
		}
	}
}

func TestApplyOutcomeConflictErrorDetails(t *testing.T) {
	f := newLifecycleFixture()
	f.action = terminalState(f, state.ActionUnknown)
	f.outcome.Status = state.OutcomeSucceeded
	f.outcome.Evidence.Ref.Version = "v2"
	_, err := state.ApplyOutcome(f.action, f.outcome, f.limits)
	if !errors.Is(err, state.ErrConflictingObservation) {
		t.Fatalf("error = %v, want conflicting observation", err)
	}
	for _, detail := range []string{"succeeded", "unknown", "note-output", "v1", "v2"} {
		if !strings.Contains(err.Error(), detail) {
			t.Errorf("error %q lacks conflict detail %q", err, detail)
		}
	}
}

func TestApplyOutcomeRejectsMismatchedBinding(t *testing.T) {
	for _, status := range []state.OutcomeStatus{state.OutcomeSucceeded, state.OutcomeFailed, state.OutcomeUnknown} {
		for _, tt := range []struct {
			name   string
			change func(*lifecycleFixture)
			err    error
			path   string
		}{
			{"wrong run", func(f *lifecycleFixture) { f.outcome.Action.Identity.RunID = "other-run" }, state.ErrScopeViolation, "outcome.action.identity"},
			{"run case mismatch", func(f *lifecycleFixture) { f.outcome.Action.Identity.RunID = "Catalog-run" }, state.ErrScopeViolation, "outcome.action.identity"},
			{"wrong worker", func(f *lifecycleFixture) { f.outcome.Action.Identity.WorkerID = "other-worker" }, state.ErrScopeViolation, "outcome.action.identity"},
			{"wrong action", func(f *lifecycleFixture) { f.outcome.Action.ID = "other-action" }, state.ErrScopeViolation, "outcome.action.id"},
			{"action case mismatch", func(f *lifecycleFixture) { f.outcome.Action.ID = "Inspect-note" }, state.ErrScopeViolation, "outcome.action.id"},
			{"stale snapshot revision", func(f *lifecycleFixture) { f.outcome.Action.Revision = 7 }, state.ErrStaleRevision, "outcome.action.revision"},
			{"future snapshot revision", func(f *lifecycleFixture) { f.outcome.Action.Revision = 9 }, state.ErrStaleRevision, "outcome.action.revision"},
			{"unrelated recorded run", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.Identity.RunID = "other-run" }, state.ErrScopeViolation, "evidence_record.producer.identity"},
			{"unrelated recorded action", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.ID = "other-action" }, state.ErrScopeViolation, "evidence_record.producer.id"},
			{"stale recorded action", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.Revision = 7 }, state.ErrStaleRevision, "evidence_record.producer.revision"},
			{"future recorded action", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.Revision = 9 }, state.ErrStaleRevision, "evidence_record.producer.revision"},
			{"different inspected source", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.Intent.Evidence.ID = "other-note" }, state.ErrScopeViolation, "evidence_record.producer.intent"},
		} {
			t.Run(string(status)+"/"+tt.name, func(t *testing.T) {
				f := newLifecycleFixture()
				f.outcome.Status = status
				tt.change(&f)
				checkApply(t, f, tt.err, state.ActionState{}, false, tt.path)
			})
		}
	}
}

func TestApplyOutcomeStaleRevisionErrorDetails(t *testing.T) {
	f := newLifecycleFixture()
	f.outcome.Action.Revision = 7
	_, err := state.ApplyOutcome(f.action, f.outcome, f.limits)
	if !errors.Is(err, state.ErrStaleRevision) {
		t.Fatalf("error = %v, want stale revision", err)
	}
	for _, detail := range []string{"outcome.action.revision", "expected 7", "current 8"} {
		if !strings.Contains(err.Error(), detail) {
			t.Errorf("error %q lacks revision detail %q", err, detail)
		}
	}
}

func TestApplyOutcomeRejectsUnsupportedSchemasAndStatuses(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*lifecycleFixture)
		err    error
		path   string
	}{
		{"missing state schema", func(f *lifecycleFixture) { f.action.SchemaVersion = 0 }, state.ErrInvalidActionState, "action_state.schema_version"},
		{"unsupported state schema", func(f *lifecycleFixture) { f.action.SchemaVersion = 2 }, state.ErrInvalidActionState, "action_state.schema_version"},
		{"missing state status", func(f *lifecycleFixture) { f.action.Status = "" }, state.ErrInvalidActionState, "action_state.status"},
		{"unsupported state status", func(f *lifecycleFixture) { f.action.Status = "done" }, state.ErrInvalidActionState, "action_state.status"},
		{"missing accepted schema", func(f *lifecycleFixture) { f.outcome.Action.SchemaVersion = 0 }, state.ErrInvalidObservation, "outcome.action.schema_version"},
		{"unsupported accepted schema", func(f *lifecycleFixture) { f.outcome.Action.SchemaVersion = 2 }, state.ErrInvalidObservation, "outcome.action.schema_version"},
		{"missing record schema", func(f *lifecycleFixture) { f.outcome.Evidence.SchemaVersion = 0 }, state.ErrInvalidObservation, "evidence_record.schema_version"},
		{"unsupported record schema", func(f *lifecycleFixture) { f.outcome.Evidence.SchemaVersion = 2 }, state.ErrInvalidObservation, "evidence_record.schema_version"},
		{"missing producer schema", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.SchemaVersion = 0 }, state.ErrInvalidObservation, "evidence_record.producer.schema_version"},
		{"unsupported producer schema", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.SchemaVersion = 2 }, state.ErrInvalidObservation, "evidence_record.producer.schema_version"},
		{"missing accepted kind", func(f *lifecycleFixture) { f.outcome.Action.Intent.Kind = "" }, state.ErrInvalidObservation, "outcome.action.intent.kind"},
		{"unsupported accepted kind", func(f *lifecycleFixture) { f.outcome.Action.Intent.Kind = "execute_command" }, state.ErrInvalidObservation, "outcome.action.intent.kind"},
		{"missing producer kind", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.Intent.Kind = "" }, state.ErrInvalidObservation, "evidence_record.producer.intent.kind"},
		{"unsupported producer kind", func(f *lifecycleFixture) { f.outcome.Evidence.Producer.Intent.Kind = "execute_command" }, state.ErrInvalidObservation, "evidence_record.producer.intent.kind"},
		{"missing status", func(f *lifecycleFixture) { f.outcome.Status = "" }, state.ErrInvalidObservation, "outcome.status"},
		{"case-mismatched status", func(f *lifecycleFixture) { f.outcome.Status = "Succeeded" }, state.ErrInvalidObservation, "outcome.status"},
		{"pending is not an outcome", func(f *lifecycleFixture) { f.outcome.Status = "pending" }, state.ErrInvalidObservation, "outcome.status"},
		{"completion is not an outcome", func(f *lifecycleFixture) { f.outcome.Status = "complete_worker" }, state.ErrInvalidObservation, "outcome.status"},
		{"verification is not an outcome", func(f *lifecycleFixture) { f.outcome.Status = "verified" }, state.ErrInvalidObservation, "outcome.status"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture()
			tt.change(&f)
			checkApply(t, f, tt.err, state.ActionState{}, false, tt.path)
		})
	}
}

func TestApplyOutcomeEnforcesStateInvariants(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*lifecycleFixture)
	}{
		{"pending with recorded evidence", func(f *lifecycleFixture) {
			f.action.LastEvidence = f.outcome.Evidence.Ref
		}},
		{"succeeded without recorded evidence", func(f *lifecycleFixture) {
			f.action = terminalState(*f, state.ActionSucceeded)
			f.action.LastEvidence = state.EvidenceRef{}
		}},
		{"failed with blank evidence version", func(f *lifecycleFixture) {
			f.action = terminalState(*f, state.ActionFailed)
			f.action.LastEvidence.Version = ""
		}},
		{"unknown with blank evidence id", func(f *lifecycleFixture) {
			f.action = terminalState(*f, state.ActionUnknown)
			f.action.LastEvidence.ID = ""
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture()
			tt.change(&f)
			checkApply(t, f, state.ErrInvalidActionState, state.ActionState{}, false, "action_state.last_evidence")
		})
	}
}

func TestApplyOutcomeRejectsMalformedMetadata(t *testing.T) {
	stateFields := []struct {
		path string
		set  func(*lifecycleFixture, string)
	}{
		{"action_state.identity.run_id", func(f *lifecycleFixture, v string) {
			f.action.Identity.RunID = v
			f.outcome.Action.Identity.RunID = v
			f.outcome.Evidence.Producer.Identity.RunID = v
		}},
		{"action_state.identity.worker_id", func(f *lifecycleFixture, v string) {
			f.action.Identity.WorkerID = v
			f.outcome.Action.Identity.WorkerID = v
			f.outcome.Evidence.Producer.Identity.WorkerID = v
		}},
		{"action_state.action_id", func(f *lifecycleFixture, v string) {
			f.action.ActionID = v
			f.outcome.Action.ID = v
			f.outcome.Evidence.Producer.ID = v
		}},
	}
	for _, field := range stateFields {
		for _, value := range []struct {
			name string
			text string
			err  error
		}{
			{"empty", "", state.ErrInvalidActionState},
			{"blank", " \t\n", state.ErrInvalidActionState},
			{"invalid UTF-8", string([]byte{0xff}), state.ErrInvalidActionState},
			{"oversized", strings.Repeat("x", 65), state.ErrLimitExceeded},
		} {
			t.Run(field.path+"/"+value.name, func(t *testing.T) {
				f := newLifecycleFixture()
				field.set(&f, value.text)
				checkApply(t, f, value.err, state.ActionState{}, false, field.path)
			})
		}
	}
	outcomeFields := []struct {
		path string
		err  error
		set  func(*lifecycleFixture, string)
	}{
		{"outcome.action.identity.run_id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Action.Identity.RunID = v }},
		{"outcome.action.identity.worker_id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Action.Identity.WorkerID = v }},
		{"outcome.action.id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Action.ID = v }},
		{"outcome.action.intent.evidence.id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Action.Intent.Evidence.ID = v }},
		{"outcome.action.intent.evidence.version", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Action.Intent.Evidence.Version = v }},
		{"evidence_record.ref.id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Evidence.Ref.ID = v }},
		{"evidence_record.ref.version", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Evidence.Ref.Version = v }},
		{"evidence_record.producer.identity.run_id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Evidence.Producer.Identity.RunID = v }},
		{"evidence_record.producer.identity.worker_id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Evidence.Producer.Identity.WorkerID = v }},
		{"evidence_record.producer.id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Evidence.Producer.ID = v }},
		{"evidence_record.producer.intent.evidence.id", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Evidence.Producer.Intent.Evidence.ID = v }},
		{"evidence_record.producer.intent.evidence.version", state.ErrInvalidObservation, func(f *lifecycleFixture, v string) { f.outcome.Evidence.Producer.Intent.Evidence.Version = v }},
	}
	for _, field := range outcomeFields {
		for _, value := range []struct {
			name string
			text string
			err  error
		}{
			{"empty", "", field.err},
			{"blank", " \t\n", field.err},
			{"invalid UTF-8", string([]byte{0xff}), field.err},
			{"oversized", strings.Repeat("x", 65), state.ErrLimitExceeded},
		} {
			t.Run(field.path+"/"+value.name, func(t *testing.T) {
				f := newLifecycleFixture()
				field.set(&f, value.text)
				checkApply(t, f, value.err, state.ActionState{}, false, field.path)
			})
		}
	}
}

func TestApplyOutcomeTerminalEvidenceByteLimits(t *testing.T) {
	for _, field := range []struct {
		name string
		path string
		set  func(*lifecycleFixture, string)
	}{
		{
			"last_evidence.id", "action_state.last_evidence",
			func(f *lifecycleFixture, v string) {
				f.action = terminalState(*f, state.ActionSucceeded)
				f.action.LastEvidence.ID = v
				f.outcome.Evidence.Ref.ID = v
			},
		},
		{
			"last_evidence.version", "action_state.last_evidence",
			func(f *lifecycleFixture, v string) {
				f.action = terminalState(*f, state.ActionSucceeded)
				f.action.LastEvidence.Version = v
				f.outcome.Evidence.Ref.Version = v
			},
		},
	} {
		for _, value := range []struct {
			name string
			text string
			err  error
		}{
			{"below", strings.Repeat("x", 63), nil},
			{"at", strings.Repeat("x", 64), nil},
			{"over", strings.Repeat("x", 65), state.ErrLimitExceeded},
			{"UTF-8 at", strings.Repeat("\u00e9", 32), nil},
			{"UTF-8 over", strings.Repeat("\u00e9", 33), state.ErrLimitExceeded},
			{"one MiB", strings.Repeat("x", 1<<20), state.ErrLimitExceeded},
		} {
			t.Run(field.name+"/"+value.name, func(t *testing.T) {
				f := newLifecycleFixture()
				field.set(&f, value.text)
				result := checkApply(t, f, value.err, state.ActionState{}, false, field.path)
				if value.err == nil {
					if result.LastEvidence != f.outcome.Evidence.Ref {
						t.Error("terminal state did not preserve the exact bounded evidence reference")
					}
					if result.Revision != f.action.Revision {
						t.Error("duplicate changed the lifecycle revision")
					}
				}
			})
		}
	}
}

func TestApplyOutcomeIdentifierByteLimits(t *testing.T) {
	fields := []struct {
		name string
		set  func(*lifecycleFixture, string)
		get  func(*state.ActionState) string
	}{
		{
			"run_id",
			func(f *lifecycleFixture, v string) {
				f.action.Identity.RunID = v
				f.outcome.Action.Identity.RunID = v
				f.outcome.Evidence.Producer.Identity.RunID = v
			},
			func(r *state.ActionState) string { return r.Identity.RunID },
		},
		{
			"worker_id",
			func(f *lifecycleFixture, v string) {
				f.action.Identity.WorkerID = v
				f.outcome.Action.Identity.WorkerID = v
				f.outcome.Evidence.Producer.Identity.WorkerID = v
			},
			func(r *state.ActionState) string { return r.Identity.WorkerID },
		},
		{
			"action_id",
			func(f *lifecycleFixture, v string) {
				f.action.ActionID = v
				f.outcome.Action.ID = v
				f.outcome.Evidence.Producer.ID = v
			},
			func(r *state.ActionState) string { return r.ActionID },
		},
	}
	for _, field := range fields {
		for _, value := range []struct {
			name string
			text string
			err  error
		}{
			{"below", strings.Repeat("x", 63), nil},
			{"at", strings.Repeat("x", 64), nil},
			{"over", strings.Repeat("x", 65), state.ErrLimitExceeded},
			{"UTF-8 at", strings.Repeat("\u00e9", 32), nil},
			{"UTF-8 over", strings.Repeat("\u00e9", 33), state.ErrLimitExceeded},
			{"one MiB", strings.Repeat("x", 1<<20), state.ErrLimitExceeded},
		} {
			t.Run(field.name+"/"+value.name, func(t *testing.T) {
				f := newLifecycleFixture()
				field.set(&f, value.text)
				result := checkApply(t, f, value.err, state.ActionState{}, false)
				if value.err == nil && field.get(result) != value.text {
					t.Error("transition did not preserve the exact bounded identifier")
				}
			})
		}
	}
}

func TestApplyOutcomeDetailLimits(t *testing.T) {
	for _, status := range []state.OutcomeStatus{state.OutcomeSucceeded, state.OutcomeFailed, state.OutcomeUnknown} {
		for _, value := range []struct {
			name string
			text string
			err  error
		}{
			{"below", strings.Repeat("x", 127), nil},
			{"at", strings.Repeat("x", 128), nil},
			{"over", strings.Repeat("x", 129), state.ErrLimitExceeded},
			{"UTF-8 at", strings.Repeat("\u00e9", 64), nil},
			{"UTF-8 over", strings.Repeat("\u00e9", 65), state.ErrLimitExceeded},
			{"one MiB", strings.Repeat("x", 1<<20), state.ErrLimitExceeded},
			{"empty", "", state.ErrInvalidObservation},
			{"blank", " \t\n", state.ErrInvalidObservation},
			{"invalid UTF-8", string([]byte{0xff}), state.ErrInvalidObservation},
		} {
			t.Run(string(status)+"/"+value.name, func(t *testing.T) {
				f := newLifecycleFixture()
				f.outcome.Status = status
				f.outcome.Detail = value.text
				checkApply(t, f, value.err, state.ActionState{}, false, "outcome.detail")
			})
		}
	}
}

func TestApplyOutcomeIdentifierLimitIsIndependentOfDetailLimit(t *testing.T) {
	for _, tt := range []struct {
		name  string
		limit int
		size  int
		err   error
	}{
		{"smaller than detail limit", 16, 17, state.ErrLimitExceeded},
		{"larger than detail limit", 256, 129, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture()
			f.limits.MaxIdentifierBytes = tt.limit
			f.action.ActionID = strings.Repeat("x", tt.size)
			f.outcome.Action.ID = f.action.ActionID
			f.outcome.Evidence.Producer.ID = f.action.ActionID
			result := checkApply(t, f, tt.err, state.ActionState{}, false, "action_state.action_id")
			if tt.err == nil && result.ActionID != f.action.ActionID {
				t.Error("identifier was constrained by the detail-text limit")
			}
		})
	}
}

func TestApplyOutcomeUsesConfiguredDetailLimit(t *testing.T) {
	for _, tt := range []struct {
		name  string
		limit int
		size  int
		err   error
	}{
		{"smaller limit at", 8, 8, nil},
		{"smaller limit over", 8, 9, state.ErrLimitExceeded},
		{"larger limit at", 256, 256, nil},
		{"larger limit over", 256, 257, state.ErrLimitExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture()
			f.limits.MaxDetailBytes = tt.limit
			f.outcome.Detail = strings.Repeat("x", tt.size)
			checkApply(t, f, tt.err, state.ActionState{}, false, "outcome.detail")
		})
	}
}

func TestApplyOutcomeRevisionSemantics(t *testing.T) {
	for _, tt := range []struct {
		name      string
		snapshot  uint64
		lifecycle uint64
		terminal  bool
		wantErr   error
	}{
		{"initial snapshot", 0, 0, false, nil},
		{"accepted snapshot", 8, 0, false, nil},
		{"maximum snapshot", ^uint64(0), 0, false, nil},
		{"exhausted pending lifecycle", 8, ^uint64(0), false, state.ErrInvalidActionState},
		{"exhausted terminal duplicate", 8, ^uint64(0), true, nil},
		{"exhausted terminal conflict", 8, ^uint64(0), true, state.ErrConflictingObservation},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newLifecycleFixture()
			f.action.SnapshotRevision = tt.snapshot
			f.action.Revision = tt.lifecycle
			f.outcome.Action.Revision = tt.snapshot
			f.outcome.Evidence.Producer.Revision = tt.snapshot
			if tt.terminal {
				f.action = terminalState(f, state.ActionSucceeded)
				f.action.SnapshotRevision = tt.snapshot
				f.action.Revision = tt.lifecycle
				if tt.wantErr == state.ErrConflictingObservation {
					f.outcome.Status = state.OutcomeFailed
					f.outcome.Evidence.Ref.Version = "v2"
				}
			}
			var want state.ActionState
			haveWant := false
			if tt.wantErr == nil {
				want = f.action
				haveWant = true
				if !tt.terminal {
					want.Status = state.ActionSucceeded
					want.LastEvidence = f.outcome.Evidence.Ref
					want.Revision++
				}
			}
			checkApply(t, f, tt.wantErr, want, haveWant)
		})
	}
}

func TestApplyOutcomeResultDoesNotAliasInputs(t *testing.T) {
	t.Run("mutating result", func(t *testing.T) {
		f := newLifecycleFixture()
		before := f
		result := checkApply(t, f, nil, state.ActionState{}, false)
		result.Identity.WorkerID = "changed"
		result.ActionID = "changed"
		result.LastEvidence.Version = "changed"
		result.Status = state.ActionFailed
		result.Revision = 99
		if !reflect.DeepEqual(f, before) {
			t.Fatal("mutating the outcome transition changed an input")
		}
	})
	t.Run("mutating inputs", func(t *testing.T) {
		f := newLifecycleFixture()
		result := checkApply(t, f, nil, state.ActionState{}, false)
		want := *result
		f.action.Identity.WorkerID = "changed"
		f.action.ActionID = "changed"
		f.action.SnapshotRevision = 99
		f.outcome.Action.Revision = 99
		if !reflect.DeepEqual(*result, want) {
			t.Fatal("mutating inputs changed the returned action state")
		}
	})
}

func TestApplyOutcomeChainsFromObservationValidation(t *testing.T) {
	f := newLifecycleFixture()
	observed, err := state.ValidateObservation(f.outcome.Action, f.outcome.Evidence, f.limits, state.Observation{
		SchemaVersion:    1,
		Identity:         f.outcome.Action.Identity,
		ActionID:         f.outcome.Action.ID,
		ExpectedRevision: f.outcome.Action.Revision,
		Status:           f.outcome.Status,
		Evidence:         f.outcome.Evidence.Ref,
		Detail:           f.outcome.Detail,
	})
	if err != nil {
		t.Fatalf("observation validation failed: %v", err)
	}
	next := checkApply(t, lifecycleFixture{action: f.action, outcome: *observed, limits: f.limits}, nil, state.ActionState{}, false)
	if next.Status != state.ActionSucceeded || next.LastEvidence != f.outcome.Evidence.Ref || next.Revision != 1 {
		t.Fatal("validated observation did not transition the pending action")
	}
	dup := lifecycleFixture{action: *next, outcome: *observed, limits: f.limits}
	again := checkApply(t, dup, nil, *next, true)
	if again.Revision != 1 {
		t.Fatal("replaying the validated observation advanced the lifecycle revision")
	}
}
