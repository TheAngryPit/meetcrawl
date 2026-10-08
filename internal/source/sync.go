package source

import "fmt"

// OutcomeCode is the honest sync result reported to operators and the index.
type OutcomeCode string

const (
	OutcomeOK                OutcomeCode = "ok"
	OutcomeUnsupportedSchema OutcomeCode = "unsupported_schema"
	OutcomeFailed            OutcomeCode = "failed"
)

func (o OutcomeCode) String() string { return string(o) }

func (o OutcomeCode) validate() error {
	switch o {
	case OutcomeOK, OutcomeUnsupportedSchema, OutcomeFailed:
		return nil
	default:
		return fmt.Errorf("source: unknown outcome %q", o)
	}
}

// SyncOutcome summarizes one sync run. Artifacts are empty when the source layout is rejected.
type SyncOutcome struct {
	Code      OutcomeCode
	Detail    string
	Artifacts []Artifact
}

func (s SyncOutcome) Validate() error {
	if err := s.Code.validate(); err != nil {
		return err
	}
	switch s.Code {
	case OutcomeOK:
		if len(s.Artifacts) == 0 {
			return fmt.Errorf("source: ok outcome requires at least one artifact")
		}
	case OutcomeUnsupportedSchema, OutcomeFailed:
		if len(s.Artifacts) != 0 {
			return fmt.Errorf("source: %s outcome must not return artifacts", s.Code)
		}
	}
	for i := range s.Artifacts {
		if err := s.Artifacts[i].Validate(); err != nil {
			return fmt.Errorf("source: artifact[%d]: %w", i, err)
		}
	}
	return nil
}
