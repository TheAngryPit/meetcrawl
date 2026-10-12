package api_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/tldv/api"
)

func TestParseTranscriptItems(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
	  "id": "transcript-1",
	  "meetingId": "653663ac7c8dbd00130f11d9",
	  "data": [
	    {"speaker": "Jane Smith", "text": "Hello team.", "startTime": 0.5, "endTime": 2.0},
	    {"speaker": "Bob Lee", "text": "Hi.", "startTime": 2.5, "endTime": 4.0}
	  ]
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
	if items[0].EndSeconds <= items[0].StartSeconds {
		t.Fatalf("end = %v start = %v", items[0].EndSeconds, items[0].StartSeconds)
	}
}

func TestValidateMeetingSummary(t *testing.T) {
	t.Parallel()
	if err := api.ValidateMeetingSummary(api.MeetingSummary{ID: "abc", HappenedAt: "2025-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := api.ValidateMeetingSummary(api.MeetingSummary{HappenedAt: "2025-01-01T00:00:00Z"}); err == nil {
		t.Fatal("expected missing id error")
	}
}
