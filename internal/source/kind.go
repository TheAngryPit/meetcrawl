package source

import "fmt"

// Kind names a transcript provider behind the shared adapter contract.
// Values are stored in archives, provenance, and the meetings index.
type Kind string

const (
	KindOpenWhispr  Kind = "openwhispr"
	KindGMeetGemini Kind = "gmeet-gemini"
	// KindExportFile is the planned generic exported-file adapter (VTT, SRT, TXT, Markdown).
	KindExportFile Kind = "export-file"
)

func (k Kind) String() string { return string(k) }

// Valid reports whether k is a known Kind constant.
func (k Kind) Valid() bool {
	switch k {
	case KindOpenWhispr, KindGMeetGemini, KindExportFile:
		return true
	default:
		return false
	}
}

func (k Kind) validate() error {
	if k.Valid() {
		return nil
	}
	return fmt.Errorf("source: unknown kind %q", k)
}
