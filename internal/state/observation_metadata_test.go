package state_test

import (
	"strings"
	"testing"

	"github.com/AntonioDiF/hans/internal/state"
)

func TestValidateObservationRejectsInvalidLimits(t *testing.T) {
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
				f := newObservationFixture()
				field.set(&f.limits, value.n)
				checkObservation(t, f, state.ErrInvalidLimits, "limits."+field.name)
			})
		}
	}
}

func TestValidateObservationRejectsMalformedMetadata(t *testing.T) {
	fields := []struct {
		path string
		err  error
		set  func(*observationFixture, string)
	}{
		{"accepted_action.identity.run_id", state.ErrInvalidAcceptedAction, func(f *observationFixture, v string) { f.accepted.Identity.RunID = v }},
		{"accepted_action.identity.worker_id", state.ErrInvalidAcceptedAction, func(f *observationFixture, v string) { f.accepted.Identity.WorkerID = v }},
		{"accepted_action.id", state.ErrInvalidAcceptedAction, func(f *observationFixture, v string) { f.accepted.ID = v }},
		{"accepted_action.intent.evidence.id", state.ErrInvalidAcceptedAction, func(f *observationFixture, v string) { f.accepted.Intent.Evidence.ID = v }},
		{"accepted_action.intent.evidence.version", state.ErrInvalidAcceptedAction, func(f *observationFixture, v string) { f.accepted.Intent.Evidence.Version = v }},
		{"evidence_record.ref.id", state.ErrInvalidEvidenceRecord, func(f *observationFixture, v string) { f.record.Ref.ID = v }},
		{"evidence_record.ref.version", state.ErrInvalidEvidenceRecord, func(f *observationFixture, v string) { f.record.Ref.Version = v }},
		{"evidence_record.producer.identity.run_id", state.ErrInvalidEvidenceRecord, func(f *observationFixture, v string) { f.record.Producer.Identity.RunID = v }},
		{"evidence_record.producer.identity.worker_id", state.ErrInvalidEvidenceRecord, func(f *observationFixture, v string) { f.record.Producer.Identity.WorkerID = v }},
		{"evidence_record.producer.id", state.ErrInvalidEvidenceRecord, func(f *observationFixture, v string) { f.record.Producer.ID = v }},
		{"evidence_record.producer.intent.evidence.id", state.ErrInvalidEvidenceRecord, func(f *observationFixture, v string) { f.record.Producer.Intent.Evidence.ID = v }},
		{"evidence_record.producer.intent.evidence.version", state.ErrInvalidEvidenceRecord, func(f *observationFixture, v string) { f.record.Producer.Intent.Evidence.Version = v }},
		{"observation.identity.run_id", state.ErrInvalidObservation, func(f *observationFixture, v string) { f.observation.Identity.RunID = v }},
		{"observation.identity.worker_id", state.ErrInvalidObservation, func(f *observationFixture, v string) { f.observation.Identity.WorkerID = v }},
		{"observation.action_id", state.ErrInvalidObservation, func(f *observationFixture, v string) { f.observation.ActionID = v }},
		{"observation.evidence.id", state.ErrInvalidObservation, func(f *observationFixture, v string) { f.observation.Evidence.ID = v }},
		{"observation.evidence.version", state.ErrInvalidObservation, func(f *observationFixture, v string) { f.observation.Evidence.Version = v }},
	}
	for _, field := range fields {
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
				f := newObservationFixture()
				field.set(&f, value.text)
				checkObservation(t, f, value.err, field.path)
			})
		}
	}
}

func TestValidateObservationIdentifierByteLimits(t *testing.T) {
	fields := []struct {
		name string
		path string
		set  func(*observationFixture, string)
		get  func(*state.ActionOutcome) string
	}{
		{
			"run_id", "accepted_action.identity.run_id",
			func(f *observationFixture, v string) {
				f.accepted.Identity.RunID = v
				f.record.Producer.Identity.RunID = v
				f.observation.Identity.RunID = v
			},
			func(r *state.ActionOutcome) string { return r.Action.Identity.RunID },
		},
		{
			"worker_id", "accepted_action.identity.worker_id",
			func(f *observationFixture, v string) {
				f.accepted.Identity.WorkerID = v
				f.record.Producer.Identity.WorkerID = v
				f.observation.Identity.WorkerID = v
			},
			func(r *state.ActionOutcome) string { return r.Action.Identity.WorkerID },
		},
		{
			"action_id", "accepted_action.id",
			func(f *observationFixture, v string) {
				f.accepted.ID = v
				f.record.Producer.ID = v
				f.observation.ActionID = v
			},
			func(r *state.ActionOutcome) string { return r.Action.ID },
		},
		{
			"target_id", "accepted_action.intent.evidence.id",
			func(f *observationFixture, v string) {
				f.accepted.Intent.Evidence.ID = v
				f.record.Producer.Intent.Evidence.ID = v
			},
			func(r *state.ActionOutcome) string { return r.Action.Intent.Evidence.ID },
		},
		{
			"target_version", "accepted_action.intent.evidence.version",
			func(f *observationFixture, v string) {
				f.accepted.Intent.Evidence.Version = v
				f.record.Producer.Intent.Evidence.Version = v
			},
			func(r *state.ActionOutcome) string { return r.Action.Intent.Evidence.Version },
		},
		{
			"result_id", "evidence_record.ref.id",
			func(f *observationFixture, v string) {
				f.record.Ref.ID = v
				f.observation.Evidence.ID = v
			},
			func(r *state.ActionOutcome) string { return r.Evidence.Ref.ID },
		},
		{
			"result_version", "evidence_record.ref.version",
			func(f *observationFixture, v string) {
				f.record.Ref.Version = v
				f.observation.Evidence.Version = v
			},
			func(r *state.ActionOutcome) string { return r.Evidence.Ref.Version },
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
				f := newObservationFixture()
				field.set(&f, value.text)
				result := checkObservation(t, f, value.err, field.path)
				if value.err == nil {
					if field.get(result) != value.text {
						t.Error("outcome did not preserve the exact bounded identifier")
					}
					if result.Evidence.Producer != result.Action {
						t.Error("bounded identifiers lost their producing-action binding")
					}
				}
			})
		}
	}
}

func TestValidateObservationDetailLimits(t *testing.T) {
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
				f := newObservationFixture()
				f.observation.Status = status
				f.observation.Detail = value.text
				result := checkObservation(t, f, value.err, "observation.detail")
				if value.err == nil && (result.Detail != value.text || result.Status != status) {
					t.Error("bounded detail was truncated or changed the outcome status")
				}
			})
		}
	}
}

func TestValidateObservationIdentifierLimitIsIndependentOfDetailLimit(t *testing.T) {
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
			f := newObservationFixture()
			f.limits.MaxIdentifierBytes = tt.limit
			f.record.Ref.ID = strings.Repeat("x", tt.size)
			f.observation.Evidence.ID = f.record.Ref.ID
			result := checkObservation(t, f, tt.err, "evidence_record.ref.id")
			if tt.err == nil && result.Evidence.Ref.ID != f.record.Ref.ID {
				t.Error("result identifier was constrained by the detail-text limit")
			}
		})
	}
}

func TestValidateObservationUsesConfiguredDetailLimit(t *testing.T) {
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
			f := newObservationFixture()
			f.limits.MaxDetailBytes = tt.limit
			f.observation.Detail = strings.Repeat("x", tt.size)
			result := checkObservation(t, f, tt.err, "observation.detail")
			if tt.err == nil && result.Detail != f.observation.Detail {
				t.Error("detail was not preserved under its configured byte limit")
			}
		})
	}
}
