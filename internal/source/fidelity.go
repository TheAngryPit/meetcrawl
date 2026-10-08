package source

import "fmt"

// Fidelity ranks how complete meeting text is. Higher rank wins when deduping.
type Fidelity int

const (
	FidelitySummary Fidelity = iota + 1
	FidelityNotes
	FidelityTranscript
)

func (f Fidelity) String() string {
	switch f {
	case FidelitySummary:
		return "summary"
	case FidelityNotes:
		return "notes"
	case FidelityTranscript:
		return "transcript"
	default:
		return fmt.Sprintf("fidelity(%d)", f)
	}
}

func (f Fidelity) validate() error {
	switch f {
	case FidelitySummary, FidelityNotes, FidelityTranscript:
		return nil
	default:
		return fmt.Errorf("source: unknown fidelity %d", f)
	}
}

// Rank returns the ordering used by the meetings index (transcript > notes > summary).
func (f Fidelity) Rank() int {
	return int(f)
}

// BestFidelity returns the higher-ranked of a and b.
func BestFidelity(a, b Fidelity) Fidelity {
	if a.Rank() >= b.Rank() {
		return a
	}
	return b
}
