package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type calendarEvent struct {
	ICalUID string
	Start   time.Time
	End     time.Time
}

// AssignMeetingIDs maps each crawler row to a meeting_id (docs/SPEC.md section 6).
func AssignMeetingIDs(rows []crawler.Row) (map[int]string, error) {
	events := collectCalendarEvents(rows)
	ids := make(map[int]string, len(rows))
	for i, row := range rows {
		id, err := meetingIDForRow(row, events)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

func collectCalendarEvents(rows []crawler.Row) []calendarEvent {
	byUID := map[string]calendarEvent{}
	for _, row := range rows {
		ical := strings.TrimSpace(row.CalendarICal)
		if ical == "" || row.WindowStart.IsZero() {
			continue
		}
		ev, ok := byUID[ical]
		if !ok {
			ev = calendarEvent{ICalUID: ical, Start: row.WindowStart, End: row.WindowEnd}
			byUID[ical] = ev
			continue
		}
		if row.WindowStart.Before(ev.Start) {
			ev.Start = row.WindowStart
		}
		if !row.WindowEnd.IsZero() && (ev.End.IsZero() || row.WindowEnd.After(ev.End)) {
			ev.End = row.WindowEnd
		}
		byUID[ical] = ev
	}
	out := make([]calendarEvent, 0, len(byUID))
	for _, ev := range byUID {
		out = append(out, ev)
	}
	return out
}

func meetingIDForRow(row crawler.Row, events []calendarEvent) (string, error) {
	ical := strings.TrimSpace(row.CalendarICal)
	if ical != "" && !row.WindowStart.IsZero() {
		for _, ev := range events {
			if ev.ICalUID == ical {
				return calendarMeetingID(ev)
			}
		}
	}
	for _, ev := range events {
		if windowOverlaps(ev, row) {
			return calendarMeetingID(ev)
		}
	}
	return adhocMeetingID(row)
}

func calendarMeetingID(ev calendarEvent) (string, error) {
	if ev.ICalUID == "" || ev.Start.IsZero() {
		return "", fmt.Errorf("index: calendar event missing iCalUID or start")
	}
	start := ev.Start.UTC().Format(time.RFC3339)
	payload := ev.ICalUID + "|" + start
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}

func adhocMeetingID(row crawler.Row) (string, error) {
	start := row.WindowStart
	if start.IsZero() {
		start = time.Unix(0, 0).UTC()
	}
	minute := start.UTC().Truncate(time.Minute).Format("2006-01-02T15:04")
	payload := row.Source.String() + row.SourceID + minute
	sum := sha256.Sum256([]byte(payload))
	return "adhoc:" + hex.EncodeToString(sum[:]), nil
}

func windowOverlaps(ev calendarEvent, row crawler.Row) bool {
	if ev.Start.IsZero() {
		return false
	}
	windowStart := ev.Start
	windowEnd := ev.End
	if windowEnd.IsZero() {
		windowEnd = ev.Start.Add(2 * time.Hour)
	}
	point := row.WindowStart
	if point.IsZero() {
		return false
	}
	return !point.Before(windowStart) && !point.After(windowEnd)
}

// BestMeetingFidelity returns the highest fidelity among rows in one meeting.
func BestMeetingFidelity(rows []crawler.Row) source.Fidelity {
	var best source.Fidelity
	for _, row := range rows {
		best = source.BestFidelity(best, row.Fidelity)
	}
	return best
}

// StrictestPrivacy picks the most restrictive class among artifacts.
func StrictestPrivacy(classes []source.PrivacyClass) source.PrivacyClass {
	best := source.PrivacyPrivate
	for _, class := range classes {
		if privacyRank(class) > privacyRank(best) {
			best = class
		}
	}
	return best
}

func privacyRank(p source.PrivacyClass) int {
	switch p {
	case source.PrivacyShareable:
		return 1
	case source.PrivacyPrivate:
		return 2
	case source.PrivacyRestricted:
		return 3
	default:
		return 0
	}
}
