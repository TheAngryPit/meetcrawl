package enrich_test

import (
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/calendar"
	"github.com/TheAngryPit/meetcrawl/internal/calendar/enrich"
	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

var syntheticEvents = []calendar.Event{
	{
		ID:                "cal-event-001",
		ICalUID:           "cal-synthetic-001",
		Start:             time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC),
		End:               time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC),
		AttendeeCount:     2,
		AttachmentFileIDs: []string{"gdrive-combined-001"},
		HangoutLink:       "https://meet.google.com/krr-xxzz-qqq",
	},
}

func TestMatchByCalendarEventID(t *testing.T) {
	t.Parallel()
	row := crawler.Row{
		Source:       source.KindOpenWhispr,
		SourceID:     "1:transcript",
		CalendarICal: "cal-synthetic-001",
		WindowStart:  time.Date(2026, 1, 15, 14, 5, 0, 0, time.UTC),
	}
	ev := enrich.MatchEvent(row, syntheticEvents)
	if ev == nil || ev.ICalUID != "cal-synthetic-001" {
		t.Fatalf("MatchEvent() = %#v, want cal-synthetic-001", ev)
	}
}

func TestMatchByMeetAttachment(t *testing.T) {
	t.Parallel()
	row := crawler.Row{
		Source:      source.KindGMeetGemini,
		SourceID:    "gdrive-combined-001#transcript",
		WindowStart: time.Date(2026, 1, 15, 14, 25, 0, 0, time.UTC),
	}
	ev := enrich.MatchEvent(row, syntheticEvents)
	if ev == nil || ev.ICalUID != "cal-synthetic-001" {
		t.Fatalf("MatchEvent() = %#v, want attachment match", ev)
	}
}

func TestMatchByMeetHangoutLink(t *testing.T) {
	t.Parallel()
	row := crawler.Row{
		Source:      source.KindExportFile,
		SourceID:    "sha256:synthetic-txt-id",
		MeetLink:    "https://meet.google.com/krr-xxzz-qqq",
		WindowStart: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
	}
	ev := enrich.MatchEvent(row, syntheticEvents)
	if ev == nil || ev.ICalUID != "cal-synthetic-001" {
		t.Fatalf("MatchEvent() = %#v, want hangout match", ev)
	}
}

func TestMatchByTimeOverlap(t *testing.T) {
	t.Parallel()
	row := crawler.Row{
		Source:      source.KindExportFile,
		SourceID:    "sha256:synthetic-vtt-id",
		WindowStart: time.Date(2026, 1, 15, 14, 25, 0, 0, time.UTC),
	}
	ev := enrich.MatchEvent(row, syntheticEvents)
	if ev == nil || ev.ICalUID != "cal-synthetic-001" {
		t.Fatalf("MatchEvent() = %#v, want overlap match", ev)
	}
}

func TestMatchByOverlapBackToBackDeterministic(t *testing.T) {
	t.Parallel()
	row := crawler.Row{
		Source:      source.KindExportFile,
		SourceID:    "sha256:overlap-tie-id",
		WindowStart: time.Date(2026, 1, 15, 14, 25, 0, 0, time.UTC),
	}
	first := calendar.Event{
		ICalUID: "cal-back-to-back-a",
		Start:   time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC),
		End:     time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC),
	}
	second := calendar.Event{
		ICalUID: "cal-back-to-back-b",
		Start:   time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC),
		End:     time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC),
	}
	orders := [][]calendar.Event{
		{first, second},
		{second, first},
	}
	var want string
	for i, events := range orders {
		ev := enrich.MatchEvent(row, events)
		if ev == nil {
			t.Fatal("MatchEvent() = nil, want overlap match")
		}
		if i == 0 {
			want = ev.ICalUID
		} else if ev.ICalUID != want {
			t.Fatalf("order %d ICalUID = %q, want %q", i, ev.ICalUID, want)
		}
	}
	if want != "cal-back-to-back-b" {
		t.Fatalf("picked %q, want cal-back-to-back-b (closest event start to 14:25)", want)
	}
}

func TestNoEnrichmentWithoutEvents(t *testing.T) {
	t.Parallel()
	row := crawler.Row{
		Source:      source.KindGMeetGemini,
		SourceID:    "gdrive-combined-001#notes",
		WindowStart: time.Date(2026, 1, 15, 14, 25, 0, 0, time.UTC),
	}
	rows := []crawler.Row{row}
	enrich.ApplyRows(rows, nil)
	if rows[0].CalendarICal != "" {
		t.Fatalf("CalendarICal = %q, want empty without events", rows[0].CalendarICal)
	}
	ev := enrich.MatchEvent(row, nil)
	if ev != nil {
		t.Fatal("expected no match without calendar events")
	}
}
