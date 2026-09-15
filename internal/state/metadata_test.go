package state_test

import (
	"strings"
	"testing"

	"github.com/AntonioDiF/hans/internal/state"
)

func setCatalogProposalEvidence(f *catalogFixture, evidence state.EvidenceRef) {
	f.scope.Evidence[1] = evidence
	f.proposal.Updates[1].Evidence = evidence
	f.proposal.Action.Evidence = evidence
}

func TestValidateIdentifierByteLimits(t *testing.T) {
	fields := []struct {
		name string
		path string
		set  func(*catalogFixture, string)
		get  func(*state.ValidationResult) string
	}{
		{
			"run_id", "scope.identity.run_id",
			func(f *catalogFixture, value string) {
				f.current.Identity.RunID = value
				f.scope.Identity.RunID = value
				f.proposal.Identity.RunID = value
			},
			func(r *state.ValidationResult) string { return r.Next.Identity.RunID },
		},
		{
			"worker_id", "scope.identity.worker_id",
			func(f *catalogFixture, value string) {
				f.current.Identity.WorkerID = value
				f.scope.Identity.WorkerID = value
				f.proposal.Identity.WorkerID = value
			},
			func(r *state.ValidationResult) string { return r.Next.Identity.WorkerID },
		},
		{
			"evidence_id", "scope.evidence[1].id",
			func(f *catalogFixture, value string) {
				evidence := f.scope.Evidence[1]
				evidence.ID = value
				setCatalogProposalEvidence(f, evidence)
			},
			func(r *state.ValidationResult) string { return r.Next.Hypotheses[1].Evidence.ID },
		},
		{
			"evidence_version", "scope.evidence[1].version",
			func(f *catalogFixture, value string) {
				evidence := f.scope.Evidence[1]
				evidence.Version = value
				setCatalogProposalEvidence(f, evidence)
			},
			func(r *state.ValidationResult) string { return r.Next.Hypotheses[1].Evidence.Version },
		},
	}
	values := []struct {
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
	}
	for _, field := range fields {
		for _, value := range values {
			t.Run(field.name+"/"+value.name, func(t *testing.T) {
				f := newCatalogFixture()
				field.set(&f, value.text)
				result := checkFixture(t, f, value.err, field.path)
				if value.err == nil {
					if field.get(result) != value.text {
						t.Error("candidate did not preserve the exact bounded identifier")
					}
					if result.Action == nil || *result.Action != *f.proposal.Action {
						t.Error("candidate did not preserve the bounded action intent")
					}
				}
			})
		}
	}
}

func TestValidateMetadataLimitsAreIndependentOfTextLimits(t *testing.T) {
	for _, tt := range []struct {
		name  string
		limit int
		size  int
		err   error
	}{
		{"smaller than text limits", 16, 17, state.ErrLimitExceeded},
		{"larger than text limits", 128, 100, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newCatalogFixture()
			f.limits.MaxIdentifierBytes = tt.limit
			evidence := f.scope.Evidence[1]
			evidence.ID = strings.Repeat("x", tt.size)
			setCatalogProposalEvidence(&f, evidence)
			result := checkFixture(t, f, tt.err, "scope.evidence[1].id")
			if tt.err == nil && result.Next.Hypotheses[1].Evidence != evidence {
				t.Error("identifier was constrained by an unrelated text limit")
			}
		})
	}
}

func TestValidateRejectsInvalidMetadataLimits(t *testing.T) {
	fields := []struct {
		name string
		set  func(*state.Limits, int)
	}{
		{"max_identifier_bytes", func(l *state.Limits, n int) { l.MaxIdentifierBytes = n }},
		{"max_scope_entries", func(l *state.Limits, n int) { l.MaxScopeEntries = n }},
	}
	for _, field := range fields {
		for _, value := range []struct {
			name string
			n    int
		}{{"zero", 0}, {"negative", -1}} {
			t.Run(field.name+"/"+value.name, func(t *testing.T) {
				f := newCatalogFixture()
				field.set(&f.limits, value.n)
				checkFixture(t, f, state.ErrInvalidLimits, "limits."+field.name)
			})
		}
	}
}

func TestValidateScopeCollectionLimits(t *testing.T) {
	for _, collection := range []string{"evidence", "allowed_actions"} {
		for _, tt := range []struct {
			name  string
			count int
			err   error
		}{
			{"empty", 0, nil},
			{"below", 1, nil},
			{"at", 2, nil},
			{"over", 3, state.ErrLimitExceeded},
		} {
			t.Run(collection+"/"+tt.name, func(t *testing.T) {
				f := newCatalogFixture()
				f.limits.MaxScopeEntries = 2
				if collection == "evidence" {
					evidence := f.scope.Evidence[0]
					f.scope.Evidence = make([]state.EvidenceRef, tt.count)
					for i := range f.scope.Evidence {
						f.scope.Evidence[i] = evidence
					}
					f.proposal.Updates[1].Evidence = evidence
					f.proposal.Action.Evidence = evidence
					if tt.count == 0 {
						f.current.Hypotheses = nil
						f.proposal.Updates = f.proposal.Updates[:1]
						f.proposal.Action = nil
					}
				} else {
					f.scope.AllowedActions = make([]state.ActionKind, tt.count)
					for i := range f.scope.AllowedActions {
						f.scope.AllowedActions[i] = state.InspectEvidence
					}
					if tt.count == 0 {
						f.proposal.Action = nil
					}
				}
				checkFixture(t, f, tt.err, "scope."+collection)
			})
		}
	}
}

func TestValidateEvidenceScopeRollover(t *testing.T) {
	t.Run("removing retained version blocks plan-only proposal", func(t *testing.T) {
		f := newCatalogFixture()
		f.scope.Evidence[0].Version = "v2"
		f.proposal.Updates = f.proposal.Updates[:1]
		f.proposal.Action = nil
		checkFixture(t, f, state.ErrInvalidState, "state.hypotheses[0].evidence")
	})
	t.Run("approved historical version remains inspectable", func(t *testing.T) {
		f := newCatalogFixture()
		currentVersion := f.scope.Evidence[0]
		currentVersion.Version = "v2"
		f.scope.Evidence = append(f.scope.Evidence, currentVersion)
		f.proposal.Updates = nil
		f.proposal.Action.Evidence = f.scope.Evidence[0]
		result := checkFixture(t, f, nil, "")
		if result.Action == nil || *result.Action != *f.proposal.Action {
			t.Error("approved historical version was not retained as the inspection target")
		}
		if len(result.Next.Hypotheses) != 1 || result.Next.Hypotheses[0] != f.current.Hypotheses[0] {
			t.Error("historical hypothesis was dropped or relabeled")
		}
	})
}
