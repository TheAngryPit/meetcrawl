package ingest

import (
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/api"
)

func MatchCalendarEvent(events []api.CalendarEvent, fileID string, modified time.Time) *api.CalendarEvent {
	for i := range events {
		ev := &events[i]
		for _, attach := range ev.AttachmentIDs {
			if attach == fileID {
				return ev
			}
		}
	}
	if modified.IsZero() {
		return nil
	}
	for i := range events {
		ev := &events[i]
		if windowOverlaps(ev.Start, ev.End, modified) {
			return ev
		}
	}
	return nil
}

func windowOverlaps(start, end, point time.Time) bool {
	if start.IsZero() {
		return false
	}
	windowStart := start.Add(-10 * time.Minute)
	windowEnd := end
	if windowEnd.IsZero() {
		windowEnd = start.Add(2 * time.Hour)
	} else {
		windowEnd = windowEnd.Add(10 * time.Minute)
	}
	return !point.Before(windowStart) && !point.After(windowEnd)
}
