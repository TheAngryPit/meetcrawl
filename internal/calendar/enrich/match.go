package enrich

import (
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/calendar"
	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

const matchLead = 10 * time.Minute
const matchTrail = 10 * time.Minute

// MatchEvent picks a calendar event for row using docs/SPEC.md §10 Q5 order:
// explicit calendar_event_id, Meet link (attachment or hangout), then time overlap.
func MatchEvent(row crawler.Row, events []calendar.Event) *calendar.Event {
	if len(events) == 0 {
		return nil
	}
	if hint := strings.TrimSpace(row.CalendarICal); hint != "" {
		if ev := matchByCalendarHint(hint, events); ev != nil {
			return ev
		}
	}
	if fileID := gmeetFileID(row); fileID != "" {
		if ev := matchByAttachment(fileID, events); ev != nil {
			return ev
		}
	}
	if link := strings.TrimSpace(row.MeetLink); link != "" {
		if ev := matchByHangout(link, events); ev != nil {
			return ev
		}
	}
	return matchByOverlap(row, events)
}

func matchByCalendarHint(hint string, events []calendar.Event) *calendar.Event {
	for i := range events {
		ev := &events[i]
		if ev.ICalUID == hint || ev.ID == hint {
			return ev
		}
	}
	return nil
}

func matchByAttachment(fileID string, events []calendar.Event) *calendar.Event {
	for i := range events {
		ev := &events[i]
		for _, attach := range ev.AttachmentFileIDs {
			if attach == fileID {
				return ev
			}
		}
	}
	return nil
}

func matchByHangout(link string, events []calendar.Event) *calendar.Event {
	link = strings.TrimSpace(link)
	for i := range events {
		ev := &events[i]
		if ev.HangoutLink != "" && ev.HangoutLink == link {
			return ev
		}
	}
	return nil
}

func matchByOverlap(row crawler.Row, events []calendar.Event) *calendar.Event {
	point := row.WindowStart
	if point.IsZero() {
		return nil
	}
	for i := range events {
		ev := &events[i]
		if windowOverlaps(ev.Start, ev.End, point) {
			return ev
		}
	}
	return nil
}

func windowOverlaps(start, end, point time.Time) bool {
	if start.IsZero() {
		return false
	}
	windowStart := start.Add(-matchLead)
	windowEnd := end
	if windowEnd.IsZero() {
		windowEnd = start.Add(2 * time.Hour)
	} else {
		windowEnd = windowEnd.Add(matchTrail)
	}
	return !point.Before(windowStart) && !point.After(windowEnd)
}

func gmeetFileID(row crawler.Row) string {
	if row.Source != source.KindGMeetGemini {
		return ""
	}
	sourceID := strings.TrimSpace(row.SourceID)
	if sourceID == "" {
		return ""
	}
	if idx := strings.Index(sourceID, "#"); idx >= 0 {
		return sourceID[:idx]
	}
	return sourceID
}
