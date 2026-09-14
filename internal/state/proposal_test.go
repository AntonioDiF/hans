package state_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/AntonioDiF/hans/internal/state"
)

type observedFact struct {
	text     string
	evidence state.EvidenceRef
}

type catalogFixture struct {
	current  state.WorkerState
	backing  []state.Hypothesis
	scope    state.Scope
	limits   state.Limits
	proposal state.Proposal
	facts    []observedFact
	verified bool
	complete bool
}

func newCatalogFixture() catalogFixture {
	identity := state.Identity{RunID: "catalog-run", WorkerID: "reader-1"}
	catalog := state.EvidenceRef{ID: "catalog-entry", Version: "v1"}
	note := state.EvidenceRef{ID: "acquisition-note", Version: "v1"}
	// Spare-capacity sentinels reveal appends that corrupt the input backing array.
	backing := []state.Hypothesis{
		{Text: "The date may be approximate.", Evidence: catalog},
		{Text: "unused slot one"},
		{Text: "unused slot two"},
	}
	return catalogFixture{
		current: state.WorkerState{
			SchemaVersion: 1,
			Identity:      identity,
			Revision:      7,
			Plan:          "Review",
			Hypotheses:    backing[:1],
		},
		backing: backing,
		scope: state.Scope{
			Identity:       identity,
			Evidence:       []state.EvidenceRef{catalog, note},
			AllowedActions: []state.ActionKind{state.InspectEvidence},
		},
		limits: state.Limits{
			MaxUpdates:         4,
			MaxPlanBytes:       64,
			MaxHypothesisBytes: 64,
			MaxHypotheses:      3,
		},
		proposal: state.Proposal{
			SchemaVersion:    1,
			Identity:         identity,
			ExpectedRevision: 7,
			Updates: []state.Update{
				{Kind: state.SetPlan, Text: "Inspect the acquisition note"},
				{Kind: state.AppendHypothesis, Text: "The vase may be a replica.", Evidence: note},
			},
			Action: &state.ActionIntent{Kind: state.InspectEvidence, Evidence: note},
		},
		facts: []observedFact{{text: "The catalog lists 1890.", evidence: catalog}},
	}
}

func snapshot(f catalogFixture) catalogFixture {
	f.current.Hypotheses = slices.Clone(f.current.Hypotheses)
	f.backing = slices.Clone(f.backing)
	f.scope.Evidence = slices.Clone(f.scope.Evidence)
	f.scope.AllowedActions = slices.Clone(f.scope.AllowedActions)
	f.proposal.Updates = slices.Clone(f.proposal.Updates)
	if f.proposal.Action != nil {
		action := *f.proposal.Action
		f.proposal.Action = &action
	}
	f.facts = slices.Clone(f.facts)
	return f
}

func checkFixture(t *testing.T, f catalogFixture, wantErr error, path string) *state.ValidationResult {
	t.Helper()
	before := snapshot(f)
	result, err := state.Validate(f.current, f.scope, f.limits, f.proposal)
	if !reflect.DeepEqual(f, before) {
		t.Errorf("validation mutated its inputs:\ngot  %#v\nwant %#v", f, before)
	}
	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("error = %v, want errors.Is(_, %v)", err, wantErr)
		}
		if err != nil && !strings.Contains(err.Error(), path) {
			t.Errorf("error %q does not identify %q", err, path)
		}
		if result != nil {
			t.Errorf("rejected proposal returned a candidate or accepted action: %#v", result)
		}
		return nil
	}
	if err != nil {
		t.Fatalf("valid proposal rejected: %v", err)
	}
	if result == nil {
		t.Fatal("valid proposal returned no result")
	}
	return result
}

func TestValidateBoundedProposal(t *testing.T) {
	f := newCatalogFixture()
	result := checkFixture(t, f, nil, "")
	want := &state.ValidationResult{
		Next: state.WorkerState{
			SchemaVersion: 1,
			Identity:      state.Identity{RunID: "catalog-run", WorkerID: "reader-1"},
			Revision:      8,
			Plan:          "Inspect the acquisition note",
			Hypotheses: []state.Hypothesis{
				{Text: "The date may be approximate.", Evidence: state.EvidenceRef{ID: "catalog-entry", Version: "v1"}},
				{Text: "The vase may be a replica.", Evidence: state.EvidenceRef{ID: "acquisition-note", Version: "v1"}},
			},
		},
		Action: &state.ActionIntent{
			Kind:     state.InspectEvidence,
			Evidence: state.EvidenceRef{ID: "acquisition-note", Version: "v1"},
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("result = %#v, want %#v", result, want)
	}
}

func TestValidateUpdateOnly(t *testing.T) {
	f := newCatalogFixture()
	f.proposal.Action = nil
	result := checkFixture(t, f, nil, "")
	if result.Action != nil || result.Next.Revision != 8 || result.Next.Plan != "Inspect the acquisition note" || len(result.Next.Hypotheses) != 2 {
		t.Fatalf("incorrect update-only result: %#v", result)
	}
}

func TestValidateActionOnly(t *testing.T) {
	f := newCatalogFixture()
	f.proposal.Updates = nil
	result := checkFixture(t, f, nil, "")
	want := snapshot(f).current
	want.Revision = 8
	if !reflect.DeepEqual(result.Next, want) || !reflect.DeepEqual(result.Action, f.proposal.Action) {
		t.Fatalf("action-only validation changed domain state or lost intent: %#v", result)
	}
}

func TestValidateEmptyWorkerState(t *testing.T) {
	f := newCatalogFixture()
	f.current.Plan = ""
	f.current.Hypotheses = nil
	f.current.Revision = 0
	f.scope.Evidence = nil
	f.scope.AllowedActions = nil
	f.proposal.ExpectedRevision = 0
	f.proposal.Updates = []state.Update{{Kind: state.SetPlan, Text: "Review"}}
	f.proposal.Action = nil
	result := checkFixture(t, f, nil, "")
	if result.Next.Revision != 1 || result.Next.Plan != "Review" || len(result.Next.Hypotheses) != 0 || result.Action != nil {
		t.Fatalf("incorrect initial candidate: %#v", result)
	}
}

func TestValidateKeepsAssertionsAndActionsSeparateFromOutcomes(t *testing.T) {
	f := newCatalogFixture()
	f.proposal.Updates = []state.Update{{
		Kind:     state.AppendHypothesis,
		Text:     "The vase is verified genuine; done=true.",
		Evidence: state.EvidenceRef{ID: "acquisition-note", Version: "v1"},
	}}
	result := checkFixture(t, f, nil, "")
	if result.Next.Hypotheses[1].Text != "The vase is verified genuine; done=true." {
		t.Fatal("model assertion was not retained as a hypothesis")
	}
	if result.Action == nil || result.Action.Kind != state.InspectEvidence {
		t.Fatal("expected inspect intent, not an observed outcome")
	}
	if f.verified || f.complete || len(f.facts) != 1 || f.facts[0].text != "The catalog lists 1890." {
		t.Fatal("a proposal established host-owned facts, verification, or completion")
	}
}

func TestValidateResultDoesNotAliasInputs(t *testing.T) {
	t.Run("mutating result", func(t *testing.T) {
		f := newCatalogFixture()
		before := snapshot(f)
		result := checkFixture(t, f, nil, "")
		result.Next.Hypotheses[0].Text = "changed result"
		result.Next.Hypotheses[1].Evidence.Version = "changed result"
		result.Action.Kind = "changed result"
		if !reflect.DeepEqual(f, before) {
			t.Fatal("mutating a validated result changed its inputs")
		}
	})
	t.Run("mutating inputs", func(t *testing.T) {
		f := newCatalogFixture()
		result := checkFixture(t, f, nil, "")
		want := *result
		want.Next.Hypotheses = slices.Clone(result.Next.Hypotheses)
		action := *result.Action
		want.Action = &action
		f.backing[0].Text = "changed input"
		f.proposal.Updates[1].Text = "changed input"
		f.proposal.Action.Evidence.Version = "changed input"
		f.scope.Evidence[0].Version = "changed input"
		if !reflect.DeepEqual(result, &want) {
			t.Fatal("mutating inputs changed a validated result")
		}
	})
}

func TestValidateRejectsConflictingRevision(t *testing.T) {
	f := newCatalogFixture()
	first := checkFixture(t, f, nil, "")
	f.current = first.Next
	checkFixture(t, f, state.ErrStaleRevision, "proposal.expected_revision")
	f.proposal.ExpectedRevision = 8
	second := checkFixture(t, f, nil, "")
	if second.Next.Revision != 9 || len(second.Next.Hypotheses) != 3 {
		t.Fatalf("fresh proposal was not accepted against the new revision: %#v", second)
	}
}

func TestValidateRevisionsAreWorkerScoped(t *testing.T) {
	first := newCatalogFixture()
	checkFixture(t, first, nil, "")
	second := newCatalogFixture()
	second.current.Identity.WorkerID = "reader-2"
	second.scope.Identity.WorkerID = "reader-2"
	second.proposal.Identity.WorkerID = "reader-2"
	result := checkFixture(t, second, nil, "")
	if result.Next.Revision != 8 || result.Next.Identity.WorkerID != "reader-2" {
		t.Fatalf("another worker's revision affected this proposal: %#v", result)
	}
}

func TestValidateRejectsInvalidProposals(t *testing.T) {
	tests := []struct {
		name   string
		change func(*catalogFixture)
		err    error
		path   string
	}{
		{"stale revision", func(f *catalogFixture) { f.proposal.ExpectedRevision = 6 }, state.ErrStaleRevision, "proposal.expected_revision"},
		{"future revision", func(f *catalogFixture) { f.proposal.ExpectedRevision = 8 }, state.ErrStaleRevision, "proposal.expected_revision"},
		{"wrong run", func(f *catalogFixture) { f.proposal.Identity.RunID = "other-run" }, state.ErrScopeViolation, "proposal.identity"},
		{"wrong worker", func(f *catalogFixture) { f.proposal.Identity.WorkerID = "other-worker" }, state.ErrScopeViolation, "proposal.identity"},
		{"self-declared scope", func(f *catalogFixture) {
			f.current.Identity.WorkerID = "other-worker"
			f.proposal.Identity.WorkerID = "other-worker"
		}, state.ErrScopeViolation, "state.identity"},
		{"unsupported schema", func(f *catalogFixture) { f.proposal.SchemaVersion = 2 }, state.ErrInvalidProposal, "proposal.schema_version"},
		{"missing schema", func(f *catalogFixture) { f.proposal.SchemaVersion = 0 }, state.ErrInvalidProposal, "proposal.schema_version"},
		{"empty proposal", func(f *catalogFixture) {
			f.proposal.Updates = nil
			f.proposal.Action = nil
		}, state.ErrInvalidProposal, "proposal"},
		{"empty plan", func(f *catalogFixture) { f.proposal.Updates[0].Text = "" }, state.ErrInvalidProposal, "proposal.updates[0].text"},
		{"blank plan", func(f *catalogFixture) { f.proposal.Updates[0].Text = " \t\n" }, state.ErrInvalidProposal, "proposal.updates[0].text"},
		{"invalid plan UTF-8", func(f *catalogFixture) { f.proposal.Updates[0].Text = string([]byte{0xff}) }, state.ErrInvalidProposal, "proposal.updates[0].text"},
		{"empty hypothesis", func(f *catalogFixture) { f.proposal.Updates[1].Text = "" }, state.ErrInvalidProposal, "proposal.updates[1].text"},
		{"blank hypothesis", func(f *catalogFixture) { f.proposal.Updates[1].Text = " \t\n" }, state.ErrInvalidProposal, "proposal.updates[1].text"},
		{"invalid hypothesis UTF-8", func(f *catalogFixture) { f.proposal.Updates[1].Text = string([]byte{0xff}) }, state.ErrInvalidProposal, "proposal.updates[1].text"},
		{"unknown evidence", func(f *catalogFixture) { f.proposal.Updates[1].Evidence.ID = "other-note" }, state.ErrScopeViolation, "proposal.updates[1].evidence"},
		{"stale evidence version", func(f *catalogFixture) { f.proposal.Updates[1].Evidence.Version = "v0" }, state.ErrScopeViolation, "proposal.updates[1].evidence"},
		{"missing evidence", func(f *catalogFixture) { f.proposal.Updates[1].Evidence = state.EvidenceRef{} }, state.ErrScopeViolation, "proposal.updates[1].evidence"},
		{"unexpected plan evidence", func(f *catalogFixture) { f.proposal.Updates[0].Evidence = f.scope.Evidence[0] }, state.ErrInvalidProposal, "proposal.updates[0].evidence"},
		{"invalid after append", func(f *catalogFixture) {
			f.proposal.Updates = []state.Update{f.proposal.Updates[1], {Kind: "delete_hypothesis"}}
		}, state.ErrDisallowedOperation, "proposal.updates[1].kind"},
		{"fabricated successful outcome", func(f *catalogFixture) { f.proposal.Action.Kind = "record_success" }, state.ErrDisallowedOperation, "proposal.action.kind"},
		{"fabricated completion", func(f *catalogFixture) { f.proposal.Action.Kind = "complete_worker" }, state.ErrDisallowedOperation, "proposal.action.kind"},
		{"missing action kind", func(f *catalogFixture) { f.proposal.Action.Kind = "" }, state.ErrDisallowedOperation, "proposal.action.kind"},
		{"action not permitted", func(f *catalogFixture) { f.scope.AllowedActions = nil }, state.ErrDisallowedOperation, "proposal.action.kind"},
		{"action outside evidence scope", func(f *catalogFixture) { f.proposal.Action.Evidence.ID = "other-note" }, state.ErrScopeViolation, "proposal.action.evidence"},
		{"action on stale evidence", func(f *catalogFixture) { f.proposal.Action.Evidence.Version = "v0" }, state.ErrScopeViolation, "proposal.action.evidence"},
		{"action without evidence", func(f *catalogFixture) { f.proposal.Action.Evidence = state.EvidenceRef{} }, state.ErrScopeViolation, "proposal.action.evidence"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCatalogFixture()
			tt.change(&f)
			checkFixture(t, f, tt.err, tt.path)
		})
	}
}

func TestValidateRejectsDisallowedUpdateKinds(t *testing.T) {
	for _, kind := range []state.UpdateKind{
		"", "unknown", "delete_plan", "delete_hypothesis", "replace_state",
		"set_identity", "set_revision", "record_fact", "grant_permission",
		"set_verdict", "complete_worker",
	} {
		t.Run(string(kind), func(t *testing.T) {
			f := newCatalogFixture()
			f.proposal.Updates = []state.Update{{Kind: kind, Text: "true"}}
			checkFixture(t, f, state.ErrDisallowedOperation, "proposal.updates[0].kind")
		})
	}
}

func TestValidateRejectsInvalidLimits(t *testing.T) {
	fields := []struct {
		name string
		set  func(*state.Limits, int)
	}{
		{"max_updates", func(l *state.Limits, n int) { l.MaxUpdates = n }},
		{"max_plan_bytes", func(l *state.Limits, n int) { l.MaxPlanBytes = n }},
		{"max_hypothesis_bytes", func(l *state.Limits, n int) { l.MaxHypothesisBytes = n }},
		{"max_hypotheses", func(l *state.Limits, n int) { l.MaxHypotheses = n }},
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

func TestValidateRejectsInvalidScope(t *testing.T) {
	tests := []struct {
		name   string
		change func(*state.Scope)
		path   string
	}{
		{"missing run", func(s *state.Scope) { s.Identity.RunID = "" }, "scope.identity"},
		{"blank worker", func(s *state.Scope) { s.Identity.WorkerID = " \t" }, "scope.identity"},
		{"invalid identity UTF-8", func(s *state.Scope) { s.Identity.RunID = string([]byte{0xff}) }, "scope.identity"},
		{"missing evidence ID", func(s *state.Scope) { s.Evidence[0].ID = "" }, "scope.evidence[0]"},
		{"missing evidence version", func(s *state.Scope) { s.Evidence[0].Version = "" }, "scope.evidence[0]"},
		{"invalid evidence UTF-8", func(s *state.Scope) { s.Evidence[0].Version = string([]byte{0xff}) }, "scope.evidence[0]"},
		{"unknown permission", func(s *state.Scope) { s.AllowedActions[0] = "execute_command" }, "scope.allowed_actions[0]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCatalogFixture()
			tt.change(&f.scope)
			checkFixture(t, f, state.ErrInvalidScope, tt.path)
		})
	}
}

func TestValidateRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name   string
		change func(*catalogFixture)
		path   string
	}{
		{"missing schema", func(f *catalogFixture) { f.current.SchemaVersion = 0 }, "state.schema_version"},
		{"unsupported schema", func(f *catalogFixture) { f.current.SchemaVersion = 2 }, "state.schema_version"},
		{"blank plan", func(f *catalogFixture) { f.current.Plan = " \t" }, "state.plan"},
		{"invalid plan UTF-8", func(f *catalogFixture) { f.current.Plan = string([]byte{0xff}) }, "state.plan"},
		{"oversized plan", func(f *catalogFixture) { f.current.Plan = strings.Repeat("x", 65) }, "state.plan"},
		{"too many hypotheses", func(f *catalogFixture) {
			h := f.current.Hypotheses[0]
			f.current.Hypotheses = []state.Hypothesis{h, h, h, h}
		}, "state.hypotheses"},
		{"empty hypothesis", func(f *catalogFixture) { f.current.Hypotheses[0].Text = "" }, "state.hypotheses[0].text"},
		{"oversized hypothesis", func(f *catalogFixture) { f.current.Hypotheses[0].Text = strings.Repeat("x", 65) }, "state.hypotheses[0].text"},
		{"unscoped evidence", func(f *catalogFixture) { f.current.Hypotheses[0].Evidence.ID = "other-note" }, "state.hypotheses[0].evidence"},
		{"unversioned evidence", func(f *catalogFixture) { f.current.Hypotheses[0].Evidence.Version = "" }, "state.hypotheses[0].evidence"},
		{"exhausted revision", func(f *catalogFixture) {
			f.current.Revision = ^uint64(0)
			f.proposal.ExpectedRevision = ^uint64(0)
		}, "state.revision"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCatalogFixture()
			tt.change(&f)
			checkFixture(t, f, state.ErrInvalidState, tt.path)
		})
	}
}

func TestValidateTextLimits(t *testing.T) {
	texts := []struct {
		name string
		text string
		err  error
	}{
		{"below", "1234567", nil},
		{"at", "12345678", nil},
		{"over", "123456789", state.ErrLimitExceeded},
		{"UTF-8 at", "\u00e9\u00e9\u00e9\u00e9", nil},
		{"UTF-8 over", "\u00e9\u00e9\u00e9\u00e9\u00e9", state.ErrLimitExceeded},
	}
	for _, kind := range []state.UpdateKind{state.SetPlan, state.AppendHypothesis} {
		for _, tt := range texts {
			t.Run(string(kind)+"/"+tt.name, func(t *testing.T) {
				f := newCatalogFixture()
				update := state.Update{Kind: kind, Text: tt.text}
				if kind == state.SetPlan {
					f.limits.MaxPlanBytes = 8
				} else {
					f.limits.MaxHypothesisBytes = 8
					f.current.Hypotheses[0].Text = "Maybe"
					update.Evidence = f.scope.Evidence[1]
				}
				f.proposal.Updates = []state.Update{update}
				result := checkFixture(t, f, tt.err, "proposal.updates[0].text")
				if tt.err == nil {
					got := result.Next.Plan
					if kind == state.AppendHypothesis {
						got = result.Next.Hypotheses[1].Text
					}
					if got != tt.text {
						t.Fatalf("text = %q, want exact untruncated value %q", got, tt.text)
					}
				}
			})
		}
	}
}

func TestValidateUpdateCountLimits(t *testing.T) {
	for _, tt := range []struct {
		name  string
		count int
		err   error
	}{{"below", 1, nil}, {"at", 2, nil}, {"over", 3, state.ErrLimitExceeded}} {
		t.Run(tt.name, func(t *testing.T) {
			f := newCatalogFixture()
			f.limits.MaxUpdates = 2
			f.proposal.Updates = make([]state.Update, tt.count)
			for i := range f.proposal.Updates {
				f.proposal.Updates[i] = state.Update{Kind: state.SetPlan, Text: "Review again"}
			}
			result := checkFixture(t, f, tt.err, "proposal.updates")
			if tt.err == nil && (result.Next.Plan != "Review again" || result.Next.Revision != 8) {
				t.Fatalf("incorrect bounded update result: %#v", result)
			}
		})
	}
}

func TestValidateHypothesisCountLimits(t *testing.T) {
	for _, tt := range []struct {
		name    string
		appends int
		count   int
		err     error
	}{{"below", 1, 2, nil}, {"at", 2, 3, nil}, {"over", 3, 4, state.ErrLimitExceeded}} {
		t.Run(tt.name, func(t *testing.T) {
			f := newCatalogFixture()
			f.proposal.Updates = make([]state.Update, tt.appends)
			for i := range f.proposal.Updates {
				f.proposal.Updates[i] = state.Update{
					Kind: state.AppendHypothesis, Text: "Another possibility", Evidence: f.scope.Evidence[1],
				}
			}
			result := checkFixture(t, f, tt.err, "proposal.updates[2]")
			if tt.err == nil && len(result.Next.Hypotheses) != tt.count {
				t.Fatalf("hypothesis count = %d, want %d", len(result.Next.Hypotheses), tt.count)
			}
		})
	}
}

func TestValidateOrderedUpdates(t *testing.T) {
	f := newCatalogFixture()
	f.proposal.Updates = []state.Update{
		{Kind: state.SetPlan, Text: "Read the catalog"},
		{Kind: state.SetPlan, Text: "Compare the acquisition note"},
	}
	result := checkFixture(t, f, nil, "")
	if result.Next.Plan != "Compare the acquisition note" || result.Next.Revision != 8 {
		t.Fatalf("updates were not ordered within one candidate revision: %#v", result)
	}
}

func TestValidatePlanAtHypothesisCapacity(t *testing.T) {
	f := newCatalogFixture()
	f.limits.MaxHypotheses = 1
	f.proposal.Updates = f.proposal.Updates[:1]
	result := checkFixture(t, f, nil, "")
	if result.Next.Plan != "Inspect the acquisition note" || len(result.Next.Hypotheses) != 1 {
		t.Fatalf("updating the plan at hypothesis capacity failed: %#v", result)
	}
}

func TestValidateStaleRevisionErrorDetails(t *testing.T) {
	f := newCatalogFixture()
	f.proposal.ExpectedRevision = 6
	_, err := state.Validate(f.current, f.scope, f.limits, f.proposal)
	if !errors.Is(err, state.ErrStaleRevision) {
		t.Fatalf("error = %v, want stale revision", err)
	}
	for _, detail := range []string{"expected 6", "current 7"} {
		if !strings.Contains(err.Error(), detail) {
			t.Errorf("error %q lacks revision detail %q", err, detail)
		}
	}
}
