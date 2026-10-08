package enrich

import (
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/calendar"
	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
)

func matchByOverlap(row crawler.Row, events []calendar.Event) *calendar.Event {
	point := row.WindowStart
	if point.IsZero() {
		return nil
	}
	var best *calendar.Event
	var bestDist time.Duration
	var haveBest bool
	for i := range events {
		ev := &events[i]
		if !windowOverlaps(ev.Start, ev.End, point) {
			continue
		}
		dist := absDuration(ev.Start.Sub(point))
		if !haveBest || dist < bestDist || (dist == bestDist && overlapLess(ev, best)) {
			best = ev
			bestDist = dist
			haveBest = true
		}
	}
	return best
}

// overlapLess is tie-break after equal start-distance: earliest start, then iCalUID.
func overlapLess(a, b *calendar.Event) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	if !a.Start.Equal(b.Start) {
		return a.Start.Before(b.Start)
	}
	return a.ICalUID < b.ICalUID
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
