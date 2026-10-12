package api_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/circleback/api"
)

func TestParseTranscriptItems(t *testing.T) {
	t.Parallel()
	raw := []byte(`[
	  {"speaker": "Jane Smith", "text": "Hello team.", "timestamp": 0.5},
	  {"speaker": "Bob Lee", "text": "Hi.", "timestamp": 2.5}
	]`)
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
	if err := api.ValidateMeetingSummary(api.MeetingSummary{ID: "abc", CreatedAt: "2025-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := api.ValidateMeetingSummary(api.MeetingSummary{CreatedAt: "2025-01-01T00:00:00Z"}); err == nil {
		t.Fatal("expected missing id error")
	}
}

func TestCalendarEventIDFromSummary(t *testing.T) {
	t.Parallel()
	id := api.CalendarEventIDFromSummary(api.MeetingSummary{
		IcalUID: "event-venue-review@google.com",
	})
	if id != "event-venue-review@google.com" {
		t.Fatalf("ical = %q", id)
	}
	id = api.CalendarEventIDFromSummary(api.MeetingSummary{
		CalendarEvent: &api.CalendarEvent{PlatformID: "provider-event-id"},
	})
	if id != "provider-event-id" {
		t.Fatalf("platform = %q", id)
	}
}
