package enrich

import (
	"strconv"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/calendar"
	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
)

// ApplyRows optionally links crawler rows to calendar events (read-only enrichment).
// When events is nil or empty, rows with source-provided calendar_event_id are unchanged.
func ApplyRows(rows []crawler.Row, events []calendar.Event) {
	if len(events) == 0 {
		return
	}
	for i := range rows {
		ev := MatchEvent(rows[i], events)
		if ev == nil {
			continue
		}
		rows[i].CalendarICal = ev.ICalUID
		if !ev.Start.IsZero() {
			rows[i].WindowStart = ev.Start
		}
		if !ev.End.IsZero() {
			rows[i].WindowEnd = ev.End
		}
		if ev.AttendeeCount > 0 && strings.TrimSpace(rows[i].Participants) == "" {
			rows[i].Participants = strconv.Itoa(ev.AttendeeCount)
		}
	}
}
