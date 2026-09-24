package state

import (
	"errors"
	"fmt"
)

// ActionStatus is the host-owned lifecycle state of an accepted action.
type ActionStatus string

const (
	ActionPending   ActionStatus = "pending"
	ActionSucceeded ActionStatus = "succeeded"
	ActionFailed    ActionStatus = "failed"
	ActionUnknown   ActionStatus = "unknown"
)

// ActionState is host-owned lifecycle state, not model-writable.
// SnapshotRevision pins the immutable accepted-action snapshot and never
// changes; Revision is the lifecycle revision that advances only when an
// outcome is first applied.
type ActionState struct {
	SchemaVersion    uint32
	Identity         Identity
	ActionID         string
	SnapshotRevision uint64
	Status           ActionStatus
	LastEvidence     EvidenceRef
	Revision         uint64
}

var (
	ErrInvalidActionState     = errors.New("invalid action state")
	ErrConflictingObservation = errors.New("conflicting observation")
)

// ApplyOutcome applies a validated host observation to host-owned action
// lifecycle state without mutating its inputs. A nil error means the
// transition (or duplicate no-op) is valid, not that the action succeeded.
//
// A pending action transitions to the observed status once, recording the
// outcome's result evidence. A terminal action accepts only an exact
// duplicate (same recorded evidence and status) as a no-op; any other
// observation is a conflict, and reconciliation stays host-owned. The
// contract has no timestamps: a late observation is observable only as a
// duplicate no-op or a conflict. It never persists, executes, retries, or
// advances worker state.
func ApplyOutcome(action ActionState, outcome ActionOutcome, limits ObservationLimits) (*ActionState, error) {
	if err := validateObservationLimits(limits); err != nil {
		return nil, err
	}
	if err := validateActionState(action, limits.MaxIdentifierBytes); err != nil {
		return nil, err
	}
	if err := validateOutcomeStructural(outcome, limits); err != nil {
		return nil, err
	}
	if outcome.Action.Identity != action.Identity {
		return nil, fmt.Errorf("%w: outcome.action.identity does not match the action state", ErrScopeViolation)
	}
	if outcome.Action.ID != action.ActionID {
		return nil, fmt.Errorf("%w: outcome.action.id does not match the action state", ErrScopeViolation)
	}
	if outcome.Action.Revision != action.SnapshotRevision {
		return nil, fmt.Errorf("%w: outcome.action.revision: expected %d, current %d", ErrStaleRevision, outcome.Action.Revision, action.SnapshotRevision)
	}
	if err := bindEvidenceRecordProducer(outcome.Evidence, outcome.Action); err != nil {
		return nil, err
	}
	if action.Status == ActionPending {
		if action.Revision == ^uint64(0) {
			return nil, fmt.Errorf("%w: action_state.revision is exhausted", ErrInvalidActionState)
		}
		next := action
		// Terminal outcome statuses share their exact values with action statuses.
		next.Status = ActionStatus(outcome.Status)
		next.LastEvidence = outcome.Evidence.Ref
		next.Revision++
		return &next, nil
	}
	// action.Status is terminal here, so its value is shared with OutcomeStatus.
	if outcome.Status == OutcomeStatus(action.Status) && outcome.Evidence.Ref == action.LastEvidence {
		next := action
		return &next, nil
	}
	return nil, fmt.Errorf("%w: observation status %q and evidence %s/%s conflict with terminal status %q and recorded evidence %s/%s",
		ErrConflictingObservation, outcome.Status, outcome.Evidence.Ref.ID, outcome.Evidence.Ref.Version,
		action.Status, action.LastEvidence.ID, action.LastEvidence.Version)
}

func validateActionState(action ActionState, maxIdentifierBytes int) error {
	if action.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: action_state.schema_version must be %d", ErrInvalidActionState, CurrentSchemaVersion)
	}
	if err := validateIdentity(action.Identity, maxIdentifierBytes, ErrInvalidActionState, "action_state.identity"); err != nil {
		return err
	}
	if err := validateText(action.ActionID, maxIdentifierBytes, ErrInvalidActionState); err != nil {
		return fmt.Errorf("action_state.action_id: %w", err)
	}
	switch action.Status {
	case ActionPending:
		if action.LastEvidence != (EvidenceRef{}) {
			return fmt.Errorf("%w: action_state.last_evidence must be empty while pending", ErrInvalidActionState)
		}
	case ActionSucceeded, ActionFailed, ActionUnknown:
		if err := validateEvidenceRef(action.LastEvidence, maxIdentifierBytes, ErrInvalidActionState, "action_state.last_evidence"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: action_state.status is not supported by this schema", ErrInvalidActionState)
	}
	return nil
}

func validateOutcomeStructural(outcome ActionOutcome, limits ObservationLimits) error {
	if err := validateAcceptedAction(outcome.Action, limits.MaxIdentifierBytes, ErrInvalidObservation, "outcome.action"); err != nil {
		return err
	}
	if outcome.Evidence.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: evidence_record.schema_version must be %d", ErrInvalidObservation, CurrentSchemaVersion)
	}
	if err := validateEvidenceRef(outcome.Evidence.Ref, limits.MaxIdentifierBytes, ErrInvalidObservation, "evidence_record.ref"); err != nil {
		return err
	}
	if err := validateAcceptedAction(outcome.Evidence.Producer, limits.MaxIdentifierBytes, ErrInvalidObservation, "evidence_record.producer"); err != nil {
		return err
	}
	switch outcome.Status {
	case OutcomeSucceeded, OutcomeFailed, OutcomeUnknown:
	default:
		return fmt.Errorf("%w: outcome.status is not supported by this schema", ErrInvalidObservation)
	}
	if err := validateText(outcome.Detail, limits.MaxDetailBytes, ErrInvalidObservation); err != nil {
		return fmt.Errorf("outcome.detail: %w", err)
	}
	return nil
}
