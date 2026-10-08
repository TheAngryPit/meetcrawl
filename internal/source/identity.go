package source

import (
	"fmt"
	"strings"
	"time"
)

// Identity is the stable handle for one artifact within a source.
type Identity struct {
	Kind     Kind
	SourceID string
}

func (id Identity) validate() error {
	if err := id.Kind.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(id.SourceID) == "" {
		return fmt.Errorf("source: empty source_id")
	}
	return nil
}

// Window bounds when an artifact was captured or the meeting occurred.
// End may be zero when the source only records a start time.
type Window struct {
	Start time.Time
	End   time.Time
}
