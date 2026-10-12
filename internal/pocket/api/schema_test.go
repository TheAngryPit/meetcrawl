package api_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/pocket/api"
)

func TestParseTranscriptItems(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
	  "success": true,
	  "data": {
	    "id": "rec_pocket_fixture_standup_001",
	    "transcript": {
	      "segments": [
	        {"speaker": "sp1", "speakerName": "Jane Smith", "text": "Hello team.", "start": 0.5, "end": 2.0},
	        {"speaker": "sp2", "speakerName": "Bob Lee", "text": "Hi.", "start": 2.5, "end": 4.0}
	      ]
	    }
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
