package provider

import (
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/api"
)

func TestFromAPICopiesHangoutLink(t *testing.T) {
	t.Parallel()
	const link = "https://meet.google.com/abc-defg-hij"
	raw := []api.CalendarEvent{
		{
			ID:          "cal-event-live-001",
			ICalUID:     "cal-live-ical-001",
			Start:       time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
			HangoutLink: link,
		},
	}
	out := fromAPI(raw)
	if len(out) != 1 {
		t.Fatalf("fromAPI() len = %d, want 1", len(out))
	}
	if out[0].HangoutLink != link {
		t.Fatalf("HangoutLink = %q, want %q", out[0].HangoutLink, link)
	}
}
