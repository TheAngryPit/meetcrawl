package docclass

import (
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/source"
)

// Classify maps a Google Doc title to artifact fidelity using Meet/Gemini naming
// patterns exercised by synthetic fixtures. Real layouts may differ (see SPEC §10).
func Classify(title string) (source.Fidelity, error) {
	lower := strings.ToLower(strings.TrimSpace(title))
	switch {
	case strings.Contains(lower, "notes by gemini"):
		return source.FidelityNotes, nil
	case strings.Contains(lower, "transcript"):
		return source.FidelityTranscript, nil
	default:
		return 0, fmt.Errorf("unrecognized Meet/Gemini doc title %q", title)
	}
}
