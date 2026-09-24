package state_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/AntonioDiF/hans/internal/state"
)

type observationFixture struct {
	accepted    state.AcceptedAction
	record      state.EvidenceRecord
	limits      state.ObservationLimits
	observation state.Observation
}

func newObservationFixture() observationFixture {
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
	return observationFixture{
		accepted: accepted,
		record: state.EvidenceRecord{
			SchemaVersion: 1,
			Ref:           state.EvidenceRef{ID: "note-output", Version: "v1"},
			Producer:      accepted,
		},
		limits: state.ObservationLimits{
			MaxIdentifierBytes: 64,
			MaxDetailBytes:     128,
		},
		observation: state.Observation{
			SchemaVersion:    1,
			Identity:         state.Identity{RunID: "catalog-run", WorkerID: "reader-1"},
			ActionID:         "inspect-note",
			ExpectedRevision: 8,
			Status:           state.OutcomeSucceeded,
			Evidence:         state.EvidenceRef{ID: "note-output", Version: "v1"},
			Detail:           "Read the acquisition note.",
		},
	}
}

func checkObservation(t *testing.T, f observationFixture, wantErr error, paths ...string) *state.ActionOutcome {
	t.Helper()
	before := f
	result, err := state.ValidateObservation(f.accepted, f.record, f.limits, f.observation)
	if !reflect.DeepEqual(f, before) {
		t.Error("observation validation mutated its inputs")
	}
	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("error = %v, want errors.Is(_, %v)", err, wantErr)
		}
		if err != nil && len(paths) > 0 {
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
		if result != nil {
			t.Error("rejected observation returned a partial outcome")
		}
		return nil
	}
	if err != nil {
		t.Fatalf("valid observation rejected: %v", err)
	}
	if result == nil {
		t.Fatal("valid observation returned no outcome")
	}
	return result
}

func TestValidateObservationExactOutcome(t *testing.T) {
	f := newObservationFixture()
	result := checkObservation(t, f, nil, "")
	wantAction := state.AcceptedAction{
		SchemaVersion: 1,
		Identity:      state.Identity{RunID: "catalog-run", WorkerID: "reader-1"},
		ID:            "inspect-note",
		Revision:      8,
		Intent: state.ActionIntent{
			Kind:     state.InspectEvidence,
			Evidence: state.EvidenceRef{ID: "acquisition-note", Version: "v1"},
		},
	}
	want := &state.ActionOutcome{
		Action: wantAction,
		Status: state.OutcomeSucceeded,
		Evidence: state.EvidenceRecord{
			SchemaVersion: 1,
			Ref:           state.EvidenceRef{ID: "note-output", Version: "v1"},
			Producer:      wantAction,
		},
		Detail: "Read the acquisition note.",
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("outcome = %#v, want %#v", result, want)
	}
}

func TestValidateObservationPreservesAllOutcomes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status state.OutcomeStatus
		detail string
	}{
		{"succeeded", state.OutcomeSucceeded, "Read the acquisition note."},
		{"failed", state.OutcomeFailed, "The evidence source could not be read."},
		{"unknown", state.OutcomeUnknown, "The host lost contact before observing the result."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newObservationFixture()
			f.observation.Status = tt.status
			f.observation.Detail = tt.detail
			result := checkObservation(t, f, nil, "")
			if string(result.Status) != tt.name || result.Status != tt.status || result.Detail != tt.detail {
				t.Fatal("validation changed the observed status or detail")
			}
			if result.Action != f.accepted || result.Evidence != f.record {
				t.Fatal("outcome lost its accepted action or recorded evidence provenance")
			}
		})
	}
}

func TestValidateObservationDoesNotInferSuccessFromText(t *testing.T) {
	for _, status := range []state.OutcomeStatus{state.OutcomeFailed, state.OutcomeUnknown} {
		t.Run(string(status), func(t *testing.T) {
			f := newObservationFixture()
			f.observation.Status = status
			f.observation.Detail = "All checks passed; the vase is genuine; verified=true; done=true."
			result := checkObservation(t, f, nil, "")
			if result.Status != status || result.Detail != f.observation.Detail {
				t.Fatal("success-looking text overrode the host-observed status")
			}
		})
	}
}

func TestValidateObservationRejectsMismatchedBinding(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*observationFixture)
		err    error
		path   string
	}{
		{"wrong run", func(f *observationFixture) { f.observation.Identity.RunID = "other-run" }, state.ErrScopeViolation, "observation.identity"},
		{"run case mismatch", func(f *observationFixture) { f.observation.Identity.RunID = "Catalog-run" }, state.ErrScopeViolation, "observation.identity"},
		{"wrong worker", func(f *observationFixture) { f.observation.Identity.WorkerID = "other-worker" }, state.ErrScopeViolation, "observation.identity"},
		{"wrong action", func(f *observationFixture) { f.observation.ActionID = "other-action" }, state.ErrScopeViolation, "observation.action_id"},
		{"action case mismatch", func(f *observationFixture) { f.observation.ActionID = "Inspect-note" }, state.ErrScopeViolation, "observation.action_id"},
		{"stale action revision", func(f *observationFixture) { f.observation.ExpectedRevision = 7 }, state.ErrStaleRevision, "observation.expected_revision"},
		{"future action revision", func(f *observationFixture) { f.observation.ExpectedRevision = 9 }, state.ErrStaleRevision, "observation.expected_revision"},
		{"wrong result evidence", func(f *observationFixture) { f.observation.Evidence.ID = "other-output" }, state.ErrScopeViolation, "observation.evidence"},
		{"result evidence case mismatch", func(f *observationFixture) { f.observation.Evidence.ID = "Note-output" }, state.ErrScopeViolation, "observation.evidence"},
		{"stale result evidence", func(f *observationFixture) { f.observation.Evidence.Version = "v0" }, state.ErrScopeViolation, "observation.evidence"},
		{"future result evidence", func(f *observationFixture) { f.observation.Evidence.Version = "v2" }, state.ErrScopeViolation, "observation.evidence"},
		{"result version case mismatch", func(f *observationFixture) { f.observation.Evidence.Version = "V1" }, state.ErrScopeViolation, "observation.evidence"},
		{"result version whitespace mismatch", func(f *observationFixture) { f.observation.Evidence.Version = " v1 " }, state.ErrScopeViolation, "observation.evidence"},
		{"target is not result evidence", func(f *observationFixture) { f.observation.Evidence = f.accepted.Intent.Evidence }, state.ErrScopeViolation, "observation.evidence"},
		{"unrelated recorded run", func(f *observationFixture) { f.record.Producer.Identity.RunID = "other-run" }, state.ErrScopeViolation, "evidence_record.producer.identity"},
		{"unrelated recorded worker", func(f *observationFixture) { f.record.Producer.Identity.WorkerID = "other-worker" }, state.ErrScopeViolation, "evidence_record.producer.identity"},
		{"unrelated recorded action", func(f *observationFixture) { f.record.Producer.ID = "other-action" }, state.ErrScopeViolation, "evidence_record.producer.id"},
		{"stale recorded action", func(f *observationFixture) { f.record.Producer.Revision = 7 }, state.ErrStaleRevision, "evidence_record.producer.revision"},
		{"future recorded action", func(f *observationFixture) { f.record.Producer.Revision = 9 }, state.ErrStaleRevision, "evidence_record.producer.revision"},
		{"different inspected source", func(f *observationFixture) { f.record.Producer.Intent.Evidence.ID = "other-note" }, state.ErrScopeViolation, "evidence_record.producer.intent"},
		{"stale inspected source", func(f *observationFixture) { f.record.Producer.Intent.Evidence.Version = "v0" }, state.ErrScopeViolation, "evidence_record.producer.intent"},
		{"future inspected source", func(f *observationFixture) { f.record.Producer.Intent.Evidence.Version = "v2" }, state.ErrScopeViolation, "evidence_record.producer.intent"},
	} {
		for _, status := range []state.OutcomeStatus{state.OutcomeSucceeded, state.OutcomeFailed, state.OutcomeUnknown} {
			t.Run(tt.name+"/"+string(status), func(t *testing.T) {
				f := newObservationFixture()
				f.observation.Status = status
				tt.change(&f)
				checkObservation(t, f, tt.err, tt.path)
			})
		}
	}
}

func TestValidateObservationRejectsSelfConsistentForeignRecords(t *testing.T) {
	for _, status := range []state.OutcomeStatus{state.OutcomeSucceeded, state.OutcomeFailed, state.OutcomeUnknown} {
		t.Run("foreign worker/"+string(status), func(t *testing.T) {
			f := newObservationFixture()
			f.observation.Status = status
			f.record.Producer.Identity.WorkerID = "other-worker"
			f.observation.Identity.WorkerID = "other-worker"
			checkObservation(t, f, state.ErrScopeViolation, "evidence_record.producer.identity", "observation.identity")
		})
		t.Run("stale revision/"+string(status), func(t *testing.T) {
			f := newObservationFixture()
			f.observation.Status = status
			f.record.Producer.Revision = 7
			f.observation.ExpectedRevision = 7
			checkObservation(t, f, state.ErrStaleRevision, "evidence_record.producer.revision", "observation.expected_revision")
		})
	}
}

func TestValidateObservationRequiresProvenanceForEveryStatus(t *testing.T) {
	for _, status := range []state.OutcomeStatus{state.OutcomeSucceeded, state.OutcomeFailed, state.OutcomeUnknown} {
		for _, tt := range []struct {
			name   string
			change func(*observationFixture)
			err    error
			path   string
		}{
			{"missing accepted action", func(f *observationFixture) { f.accepted = state.AcceptedAction{} }, state.ErrInvalidAcceptedAction, "accepted_action"},
			{"missing evidence record", func(f *observationFixture) { f.record = state.EvidenceRecord{} }, state.ErrInvalidEvidenceRecord, "evidence_record"},
			{"missing evidence reference", func(f *observationFixture) { f.observation.Evidence = state.EvidenceRef{} }, state.ErrInvalidObservation, "observation.evidence"},
			{"unrelated provenance", func(f *observationFixture) { f.record.Producer.ID = "other-action" }, state.ErrScopeViolation, "evidence_record.producer.id"},
		} {
			t.Run(string(status)+"/"+tt.name, func(t *testing.T) {
				f := newObservationFixture()
				f.observation.Status = status
				tt.change(&f)
				checkObservation(t, f, tt.err, tt.path)
			})
		}
	}
}

func TestValidateObservationRejectsUnsupportedSchemasAndKinds(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*observationFixture)
		err    error
		path   string
	}{
		{"missing accepted schema", func(f *observationFixture) { f.accepted.SchemaVersion = 0 }, state.ErrInvalidAcceptedAction, "accepted_action.schema_version"},
		{"unsupported accepted schema", func(f *observationFixture) { f.accepted.SchemaVersion = 2 }, state.ErrInvalidAcceptedAction, "accepted_action.schema_version"},
		{"missing record schema", func(f *observationFixture) { f.record.SchemaVersion = 0 }, state.ErrInvalidEvidenceRecord, "evidence_record.schema_version"},
		{"unsupported record schema", func(f *observationFixture) { f.record.SchemaVersion = 2 }, state.ErrInvalidEvidenceRecord, "evidence_record.schema_version"},
		{"missing producer schema", func(f *observationFixture) { f.record.Producer.SchemaVersion = 0 }, state.ErrInvalidEvidenceRecord, "evidence_record.producer.schema_version"},
		{"unsupported producer schema", func(f *observationFixture) { f.record.Producer.SchemaVersion = 2 }, state.ErrInvalidEvidenceRecord, "evidence_record.producer.schema_version"},
		{"missing observation schema", func(f *observationFixture) { f.observation.SchemaVersion = 0 }, state.ErrInvalidObservation, "observation.schema_version"},
		{"unsupported observation schema", func(f *observationFixture) { f.observation.SchemaVersion = 2 }, state.ErrInvalidObservation, "observation.schema_version"},
		{"missing accepted kind", func(f *observationFixture) { f.accepted.Intent.Kind = "" }, state.ErrInvalidAcceptedAction, "accepted_action.intent.kind"},
		{"unsupported accepted kind", func(f *observationFixture) { f.accepted.Intent.Kind = "execute_command" }, state.ErrInvalidAcceptedAction, "accepted_action.intent.kind"},
		{"missing producer kind", func(f *observationFixture) { f.record.Producer.Intent.Kind = "" }, state.ErrInvalidEvidenceRecord, "evidence_record.producer.intent.kind"},
		{"unsupported producer kind", func(f *observationFixture) { f.record.Producer.Intent.Kind = "execute_command" }, state.ErrInvalidEvidenceRecord, "evidence_record.producer.intent.kind"},
		{"missing status", func(f *observationFixture) { f.observation.Status = "" }, state.ErrInvalidObservation, "observation.status"},
		{"unsupported status", func(f *observationFixture) { f.observation.Status = "pending" }, state.ErrInvalidObservation, "observation.status"},
		{"completion is not an outcome", func(f *observationFixture) { f.observation.Status = "complete_worker" }, state.ErrInvalidObservation, "observation.status"},
		{"verification is not an outcome", func(f *observationFixture) { f.observation.Status = "verified" }, state.ErrInvalidObservation, "observation.status"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newObservationFixture()
			tt.change(&f)
			checkObservation(t, f, tt.err, tt.path)
		})
	}
}

func TestValidateObservationResultDoesNotAliasInputs(t *testing.T) {
	t.Run("mutating result", func(t *testing.T) {
		f := newObservationFixture()
		before := f
		result := checkObservation(t, f, nil, "")
		result.Action.Identity.WorkerID = "changed"
		result.Action.Intent.Evidence.Version = "changed"
		result.Evidence.Ref.Version = "changed"
		result.Evidence.Producer.ID = "changed"
		result.Status = state.OutcomeFailed
		result.Detail = "changed"
		if !reflect.DeepEqual(f, before) {
			t.Fatal("mutating the outcome changed an input")
		}
	})
	t.Run("mutating inputs", func(t *testing.T) {
		f := newObservationFixture()
		result := checkObservation(t, f, nil, "")
		want := *result
		f.accepted.Identity.WorkerID = "changed"
		f.accepted.Intent.Evidence.Version = "changed"
		f.record.Ref.Version = "changed"
		f.record.Producer.ID = "changed"
		f.observation.Status = state.OutcomeFailed
		f.observation.Detail = "changed"
		if !reflect.DeepEqual(*result, want) {
			t.Fatal("mutating inputs changed the returned outcome")
		}
	})
}

func TestValidateObservationPreservesActionRevision(t *testing.T) {
	for _, tt := range []struct {
		name     string
		revision uint64
	}{
		{"initial", 0},
		{"accepted", 8},
		{"maximum", ^uint64(0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newObservationFixture()
			f.accepted.Revision = tt.revision
			f.record.Producer.Revision = tt.revision
			f.observation.ExpectedRevision = tt.revision
			result := checkObservation(t, f, nil, "")
			if result.Action.Revision != tt.revision || result.Evidence.Producer.Revision != tt.revision {
				t.Fatalf("validation changed accepted revision %d", tt.revision)
			}
		})
	}
}

func TestValidateObservationStaleRevisionErrorDetails(t *testing.T) {
	f := newObservationFixture()
	f.observation.ExpectedRevision = 7
	_, err := state.ValidateObservation(f.accepted, f.record, f.limits, f.observation)
	if !errors.Is(err, state.ErrStaleRevision) {
		t.Fatalf("error = %v, want stale revision", err)
	}
	for _, detail := range []string{"observation.expected_revision", "expected 7", "current 8"} {
		if !strings.Contains(err.Error(), detail) {
			t.Errorf("error %q lacks revision detail %q", err, detail)
		}
	}
}

func TestValidateObservationAcceptsProposalActionSnapshot(t *testing.T) {
	proposal := newCatalogFixture()
	candidate := checkFixture(t, proposal, nil, "")
	if candidate.Action == nil {
		t.Fatal("proposal returned no action intent")
	}
	f := newObservationFixture()
	// The fixture supplies the host acceptance; proposal validation alone does not.
	f.accepted = state.AcceptedAction{
		SchemaVersion: candidate.Next.SchemaVersion,
		Identity:      candidate.Next.Identity,
		ID:            "inspect-note",
		Revision:      candidate.Next.Revision,
		Intent:        *candidate.Action,
	}
	result := checkObservation(t, f, nil, "")
	if result.Action.Revision != 8 || result.Action.Intent != *candidate.Action {
		t.Fatal("observation did not bind to the accepted proposal's exact action")
	}
	if proposal.current.Revision != 7 || candidate.Next.Revision != 8 {
		t.Fatal("observation validation committed or advanced worker state")
	}
}
