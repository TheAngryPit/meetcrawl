package api

import (
	"encoding/json"
	"testing"
)

func TestParseTranscriptSegmentsRejectsUnknownShape(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[{"start":"bad","end":1,"text":"x"}]`)
	_, err := ParseTranscriptSegments(raw)
	if err == nil {
		t.Fatal("expected schema error")
	}
}
