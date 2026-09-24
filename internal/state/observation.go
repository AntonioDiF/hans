package state

import (
	"errors"
	"fmt"
)

// AcceptedAction is a host-owned acceptance snapshot, not a proposed action.
type AcceptedAction struct {
	SchemaVersion uint32
	Identity      Identity
	ID            string
	Revision      uint64
	Intent        ActionIntent
}

// EvidenceRecord binds host-recorded result evidence to its producing action.
// Ref identifies the result; Producer.Intent identifies the inspection target.
type EvidenceRecord struct {
	SchemaVersion uint32
	Ref           EvidenceRef
	Producer      AcceptedAction
}

type OutcomeStatus string

const (
	OutcomeSucceeded OutcomeStatus = "succeeded"
	OutcomeFailed    OutcomeStatus = "failed"
	OutcomeUnknown   OutcomeStatus = "unknown"
)

// Observation comes from the host's tool adapter, never from a worker proposal.
// ExpectedRevision identifies the accepted action, not the latest worker state.
type Observation struct {
	SchemaVersion    uint32
	Identity         Identity
	ActionID         string
	ExpectedRevision uint64
	Status           OutcomeStatus
	Evidence         EvidenceRef
	Detail           string
}

// ObservationLimits are explicit positive UTF-8 byte bounds, not token budgets.
type ObservationLimits struct {
	MaxIdentifierBytes int
	MaxDetailBytes     int
}

// ActionOutcome is a detached observation, not a commit, verdict, or completion.
type ActionOutcome struct {
	Action   AcceptedAction
	Status   OutcomeStatus
	Evidence EvidenceRecord
	Detail   string
}

var (
	ErrInvalidAcceptedAction = errors.New("invalid accepted action")
	ErrInvalidEvidenceRecord = errors.New("invalid evidence record")
	ErrInvalidObservation    = errors.New("invalid observation")
)

// ValidateObservation checks host-supplied records without changing them.
// A nil error means the record is valid, not that the action succeeded.
// The host supplies trusted acceptance, provenance, and observation inputs;
// matching those snapshots does not verify evidence contents, current-worker
// freshness, or replay history.
func ValidateObservation(accepted AcceptedAction, record EvidenceRecord, limits ObservationLimits, observation Observation) (*ActionOutcome, error) {
	if err := validateObservationLimits(limits); err != nil {
		return nil, err
	}
	if err := validateAcceptedAction(accepted, limits.MaxIdentifierBytes, ErrInvalidAcceptedAction, "accepted_action"); err != nil {
		return nil, err
	}
	if err := validateEvidenceRecord(record, accepted, limits.MaxIdentifierBytes, ErrInvalidEvidenceRecord); err != nil {
		return nil, err
	}
	if err := validateObservationFields(observation, limits); err != nil {
		return nil, err
	}
	if observation.Identity != accepted.Identity {
		return nil, fmt.Errorf("%w: observation.identity does not match the accepted action", ErrScopeViolation)
	}
	if observation.ActionID != accepted.ID {
		return nil, fmt.Errorf("%w: observation.action_id does not match the accepted action", ErrScopeViolation)
	}
	if observation.ExpectedRevision != accepted.Revision {
		return nil, fmt.Errorf("%w: observation.expected_revision: expected %d, current %d", ErrStaleRevision, observation.ExpectedRevision, accepted.Revision)
	}
	if observation.Evidence != record.Ref {
		return nil, fmt.Errorf("%w: observation.evidence does not match the recorded result source/version", ErrScopeViolation)
	}
	return &ActionOutcome{
		Action:   accepted,
		Status:   observation.Status,
		Evidence: record,
		Detail:   observation.Detail,
	}, nil
}

func validateObservationLimits(limits ObservationLimits) error {
	for _, limit := range []struct {
		name  string
		value int
	}{
		{"max_identifier_bytes", limits.MaxIdentifierBytes},
		{"max_detail_bytes", limits.MaxDetailBytes},
	} {
		if limit.value <= 0 {
			return fmt.Errorf("%w: limits.%s must be positive, got %d", ErrInvalidLimits, limit.name, limit.value)
		}
	}
	return nil
}

func validateAcceptedAction(action AcceptedAction, maxIdentifierBytes int, invalid error, path string) error {
	if action.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: %s.schema_version must be %d", invalid, path, CurrentSchemaVersion)
	}
	if err := validateIdentity(action.Identity, maxIdentifierBytes, invalid, path+".identity"); err != nil {
		return err
	}
	if err := validateText(action.ID, maxIdentifierBytes, invalid); err != nil {
		return fmt.Errorf("%s.id: %w", path, err)
	}
	if action.Intent.Kind != InspectEvidence {
		return fmt.Errorf("%w: %s.intent.kind is not supported by this schema", invalid, path)
	}
	return validateEvidenceRef(action.Intent.Evidence, maxIdentifierBytes, invalid, path+".intent.evidence")
}

func validateEvidenceRecord(record EvidenceRecord, accepted AcceptedAction, maxIdentifierBytes int, invalid error) error {
	if record.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: evidence_record.schema_version must be %d", invalid, CurrentSchemaVersion)
	}
	if err := validateEvidenceRef(record.Ref, maxIdentifierBytes, invalid, "evidence_record.ref"); err != nil {
		return err
	}
	if err := validateAcceptedAction(record.Producer, maxIdentifierBytes, invalid, "evidence_record.producer"); err != nil {
		return err
	}
	return bindEvidenceRecordProducer(record, accepted)
}

// bindEvidenceRecordProducer checks that the recorded producer is exactly the
// accepted action that produced the record.
func bindEvidenceRecordProducer(record EvidenceRecord, accepted AcceptedAction) error {
	if record.Producer.Identity != accepted.Identity {
		return fmt.Errorf("%w: evidence_record.producer.identity does not match the accepted action", ErrScopeViolation)
	}
	if record.Producer.ID != accepted.ID {
		return fmt.Errorf("%w: evidence_record.producer.id does not match the accepted action", ErrScopeViolation)
	}
	if record.Producer.Revision != accepted.Revision {
		return fmt.Errorf("%w: evidence_record.producer.revision: expected %d, current %d", ErrStaleRevision, record.Producer.Revision, accepted.Revision)
	}
	if record.Producer.Intent != accepted.Intent {
		return fmt.Errorf("%w: evidence_record.producer.intent does not match the accepted kind and source/version", ErrScopeViolation)
	}
	return nil
}

func validateObservationFields(observation Observation, limits ObservationLimits) error {
	if observation.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: observation.schema_version must be %d", ErrInvalidObservation, CurrentSchemaVersion)
	}
	if err := validateIdentity(observation.Identity, limits.MaxIdentifierBytes, ErrInvalidObservation, "observation.identity"); err != nil {
		return err
	}
	if err := validateText(observation.ActionID, limits.MaxIdentifierBytes, ErrInvalidObservation); err != nil {
		return fmt.Errorf("observation.action_id: %w", err)
	}
	if err := validateEvidenceRef(observation.Evidence, limits.MaxIdentifierBytes, ErrInvalidObservation, "observation.evidence"); err != nil {
		return err
	}
	switch observation.Status {
	case OutcomeSucceeded, OutcomeFailed, OutcomeUnknown:
	default:
		return fmt.Errorf("%w: observation.status is not supported by this schema", ErrInvalidObservation)
	}
	if err := validateText(observation.Detail, limits.MaxDetailBytes, ErrInvalidObservation); err != nil {
		return fmt.Errorf("observation.detail: %w", err)
	}
	return nil
}

func validateIdentity(identity Identity, maxIdentifierBytes int, invalid error, path string) error {
	if err := validateText(identity.RunID, maxIdentifierBytes, invalid); err != nil {
		return fmt.Errorf("%s.run_id: %w", path, err)
	}
	if err := validateText(identity.WorkerID, maxIdentifierBytes, invalid); err != nil {
		return fmt.Errorf("%s.worker_id: %w", path, err)
	}
	return nil
}

func validateEvidenceRef(ref EvidenceRef, maxIdentifierBytes int, invalid error, path string) error {
	if err := validateText(ref.ID, maxIdentifierBytes, invalid); err != nil {
		return fmt.Errorf("%s.id: %w", path, err)
	}
	if err := validateText(ref.Version, maxIdentifierBytes, invalid); err != nil {
		return fmt.Errorf("%s.version: %w", path, err)
	}
	return nil
}
