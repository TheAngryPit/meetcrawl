package source

import (
	"fmt"
	"strings"
)

// Kind names a transcript provider behind the shared adapter contract.
// Values are stored in archives, provenance, and the meetings index.
// Any non-empty string is valid; shipped kinds are the Kind* constants below.
type Kind string

const (
	KindOpenWhispr  Kind = "openwhispr"
	KindGMeetGemini Kind = "gmeet-gemini"
	KindExportFile  Kind = "export-file"
	KindGrain       Kind = "grain"
	KindGranola     Kind = "granola"
)

func (k Kind) String() string { return string(k) }

// Valid reports whether k is a non-empty kind string.
func (k Kind) Valid() bool {
	return strings.TrimSpace(string(k)) != ""
}

func (k Kind) validate() error {
	if !k.Valid() {
		return fmt.Errorf("source: empty kind")
	}
	return nil
}
