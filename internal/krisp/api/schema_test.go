package api_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/krisp/api"
)

func TestParseTranscriptDocument(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
	  "transcript": {
	    "speakers": {"1": {"first_name": "Jane", "last_name": "Smith"}},
	    "segments": [
	      {"speaker": 1, "text": "Hello team.", "start": 0, "end": 2.5},
	      {"speaker": 2, "text": "Hi.", "start": 2.5, "end": 4}
	    ]
	  }
	}`)
	items, err := api.ParseTranscriptItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
	if items[0].SpeakerDisplay != "Jane Smith" {
		t.Fatalf("speaker = %q", items[0].SpeakerDisplay)
	}
	if items[1].SpeakerDisplay != "Speaker 2" {
		t.Fatalf("fallback speaker = %q", items[1].SpeakerDisplay)
	}
}

func TestValidateMeetingSummary(t *testing.T) {
	t.Parallel()
	duration := 60
	if err := api.ValidateMeetingSummary(api.MeetingSummary{ID: "abc", StartedAt: "2025-01-01T00:00:00Z", Duration: &duration}); err != nil {
		t.Fatal(err)
	}
	if err := api.ValidateMeetingSummary(api.MeetingSummary{StartedAt: "2025-01-01T00:00:00Z"}); err == nil {
		t.Fatal("expected missing id error")
	}
}
