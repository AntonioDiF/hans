// Package state validates bounded worker proposals without committing them.
package state

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const CurrentSchemaVersion uint32 = 1

type Identity struct {
	RunID    string
	WorkerID string
}

// EvidenceRef pins an opaque host-recorded source and version, not a claimed fact.
type EvidenceRef struct {
	ID      string
	Version string
}

type Hypothesis struct {
	Text     string
	Evidence EvidenceRef
}

type WorkerState struct {
	SchemaVersion uint32
	Identity      Identity
	Revision      uint64
	Plan          string
	Hypotheses    []Hypothesis
}

type UpdateKind string

const (
	SetPlan          UpdateKind = "set_plan"
	AppendHypothesis UpdateKind = "append_hypothesis"
)

// SetPlan uses Text alone; AppendHypothesis also requires an Evidence reference.
type Update struct {
	Kind     UpdateKind
	Text     string
	Evidence EvidenceRef
}

type ActionKind string

const InspectEvidence ActionKind = "inspect_evidence"

// ActionIntent requests an action; it never represents execution or success.
type ActionIntent struct {
	Kind     ActionKind
	Evidence EvidenceRef
}

type Proposal struct {
	SchemaVersion    uint32
	Identity         Identity
	ExpectedRevision uint64
	Updates          []Update
	Action           *ActionIntent
}

// Scope must come from the host, independently of the proposal's claimed identity.
type Scope struct {
	Identity       Identity
	Evidence       []EvidenceRef
	AllowedActions []ActionKind
}

// Limits are explicit, positive bounds. Text sizes count UTF-8 bytes, not tokens.
type Limits struct {
	MaxUpdates         int
	MaxPlanBytes       int
	MaxHypothesisBytes int
	MaxHypotheses      int
}

// ValidationResult is a detached candidate and optional intent, not a committed
// transition, execution permission, observation, verifier verdict, or completion.
type ValidationResult struct {
	Next   WorkerState
	Action *ActionIntent
}

var (
	ErrInvalidLimits       = errors.New("invalid limits")
	ErrInvalidScope        = errors.New("invalid scope")
	ErrInvalidState        = errors.New("invalid worker state")
	ErrInvalidProposal     = errors.New("invalid proposal")
	ErrStaleRevision       = errors.New("stale revision")
	ErrScopeViolation      = errors.New("scope violation")
	ErrDisallowedOperation = errors.New("disallowed operation")
	ErrLimitExceeded       = errors.New("limit exceeded")
)

// Validate checks all updates and the action together without mutating inputs.
// On rejection it returns a nil result, so no partial candidate or action escapes.
func Validate(current WorkerState, scope Scope, limits Limits, proposal Proposal) (*ValidationResult, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	if err := validateState(current, scope, limits); err != nil {
		return nil, err
	}
	if proposal.SchemaVersion != CurrentSchemaVersion {
		return nil, fmt.Errorf("%w: proposal.schema_version must be %d", ErrInvalidProposal, CurrentSchemaVersion)
	}
	if proposal.Identity != scope.Identity {
		return nil, fmt.Errorf("%w: proposal.identity does not match the host scope", ErrScopeViolation)
	}
	if proposal.ExpectedRevision != current.Revision {
		return nil, fmt.Errorf("%w: proposal.expected_revision: expected %d, current %d", ErrStaleRevision, proposal.ExpectedRevision, current.Revision)
	}
	if len(proposal.Updates) > limits.MaxUpdates {
		return nil, fmt.Errorf("%w: proposal.updates has %d entries, maximum %d", ErrLimitExceeded, len(proposal.Updates), limits.MaxUpdates)
	}
	if len(proposal.Updates) == 0 && proposal.Action == nil {
		return nil, fmt.Errorf("%w: proposal requires an update or action", ErrInvalidProposal)
	}

	next := current
	next.Hypotheses = slices.Clone(current.Hypotheses)
	for i, update := range proposal.Updates {
		path := fmt.Sprintf("proposal.updates[%d]", i)
		switch update.Kind {
		case SetPlan:
			if update.Evidence != (EvidenceRef{}) {
				return nil, fmt.Errorf("%w: %s.evidence is not permitted for set_plan", ErrInvalidProposal, path)
			}
			if err := validateText(update.Text, limits.MaxPlanBytes); err != nil {
				return nil, fmt.Errorf("%s.text: %w", path, err)
			}
			next.Plan = update.Text
		case AppendHypothesis:
			if err := validateText(update.Text, limits.MaxHypothesisBytes); err != nil {
				return nil, fmt.Errorf("%s.text: %w", path, err)
			}
			if !slices.Contains(scope.Evidence, update.Evidence) {
				return nil, fmt.Errorf("%w: %s.evidence is not an approved source/version", ErrScopeViolation, path)
			}
			if len(next.Hypotheses) >= limits.MaxHypotheses {
				return nil, fmt.Errorf("%w: %s would exceed %d hypotheses", ErrLimitExceeded, path, limits.MaxHypotheses)
			}
			next.Hypotheses = append(next.Hypotheses, Hypothesis{Text: update.Text, Evidence: update.Evidence})
		default:
			return nil, fmt.Errorf("%w: %s.kind is not a permitted update", ErrDisallowedOperation, path)
		}
	}

	var action *ActionIntent
	if proposal.Action != nil {
		if proposal.Action.Kind != InspectEvidence || !slices.Contains(scope.AllowedActions, proposal.Action.Kind) {
			return nil, fmt.Errorf("%w: proposal.action.kind is not permitted by the schema and host scope", ErrDisallowedOperation)
		}
		if !slices.Contains(scope.Evidence, proposal.Action.Evidence) {
			return nil, fmt.Errorf("%w: proposal.action.evidence is not an approved source/version", ErrScopeViolation)
		}
		intent := *proposal.Action
		action = &intent
	}

	next.Revision++
	return &ValidationResult{Next: next, Action: action}, nil
}

func validateLimits(limits Limits) error {
	for _, limit := range []struct {
		name  string
		value int
	}{
		{"max_updates", limits.MaxUpdates},
		{"max_plan_bytes", limits.MaxPlanBytes},
		{"max_hypothesis_bytes", limits.MaxHypothesisBytes},
		{"max_hypotheses", limits.MaxHypotheses},
	} {
		if limit.value <= 0 {
			return fmt.Errorf("%w: limits.%s must be positive, got %d", ErrInvalidLimits, limit.name, limit.value)
		}
	}
	return nil
}

func validateScope(scope Scope) error {
	if !nonblankUTF8(scope.Identity.RunID) || !nonblankUTF8(scope.Identity.WorkerID) {
		return fmt.Errorf("%w: scope.identity requires nonblank UTF-8 run and worker IDs", ErrInvalidScope)
	}
	for i, evidence := range scope.Evidence {
		if !nonblankUTF8(evidence.ID) || !nonblankUTF8(evidence.Version) {
			return fmt.Errorf("%w: scope.evidence[%d] requires a nonblank UTF-8 ID and version", ErrInvalidScope, i)
		}
	}
	for i, kind := range scope.AllowedActions {
		if kind != InspectEvidence {
			return fmt.Errorf("%w: scope.allowed_actions[%d] is not supported by this schema", ErrInvalidScope, i)
		}
	}
	return nil
}

func validateState(current WorkerState, scope Scope, limits Limits) error {
	if current.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: state.schema_version must be %d", ErrInvalidState, CurrentSchemaVersion)
	}
	if current.Identity != scope.Identity {
		return fmt.Errorf("%w: state.identity does not match the host scope", ErrScopeViolation)
	}
	if current.Revision == ^uint64(0) {
		return fmt.Errorf("%w: state.revision is exhausted", ErrInvalidState)
	}
	if current.Plan != "" {
		if err := validateText(current.Plan, limits.MaxPlanBytes); err != nil {
			return fmt.Errorf("%w: state.plan: %v", ErrInvalidState, err)
		}
	}
	if len(current.Hypotheses) > limits.MaxHypotheses {
		return fmt.Errorf("%w: state.hypotheses exceeds %d entries", ErrInvalidState, limits.MaxHypotheses)
	}
	for i, hypothesis := range current.Hypotheses {
		if err := validateText(hypothesis.Text, limits.MaxHypothesisBytes); err != nil {
			return fmt.Errorf("%w: state.hypotheses[%d].text: %v", ErrInvalidState, i, err)
		}
		if !slices.Contains(scope.Evidence, hypothesis.Evidence) {
			return fmt.Errorf("%w: state.hypotheses[%d].evidence is not an approved source/version", ErrInvalidState, i)
		}
	}
	return nil
}

func validateText(text string, maxBytes int) error {
	if len(text) > maxBytes {
		return fmt.Errorf("%w: text has %d bytes, maximum %d", ErrLimitExceeded, len(text), maxBytes)
	}
	if !nonblankUTF8(text) {
		return fmt.Errorf("%w: text must be nonblank UTF-8", ErrInvalidProposal)
	}
	return nil
}

func nonblankUTF8(value string) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) != ""
}
