package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func ParseTranscriptItems(raw json.RawMessage) ([]TranscriptItem, error) {
	var segments []TranscriptSegmentRaw
	if err := json.Unmarshal(raw, &segments); err != nil {
		return nil, fmt.Errorf("transcript: expected array: %w", err)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("transcript has no segments")
	}
	out := make([]TranscriptItem, 0, len(segments))
	for i, seg := range segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			return nil, fmt.Errorf("segment[%d]: text is empty", i)
		}
		speaker := strings.TrimSpace(seg.Speaker)
		if speaker == "" {
			return nil, fmt.Errorf("segment[%d]: speaker is empty", i)
		}
		if seg.Timestamp < 0 {
			return nil, fmt.Errorf("segment[%d]: timestamp must be non-negative", i)
		}
		end := seg.Timestamp + 1
		if i+1 < len(segments) {
			next := segments[i+1].Timestamp
			if next > seg.Timestamp {
				end = next
			}
		}
		out = append(out, TranscriptItem{
			SpeakerDisplay: speaker,
			Text:           text,
			StartSeconds:   seg.Timestamp,
			EndSeconds:     end,
		})
	}
	return out, nil
}

func ValidateMeetingSummary(summary MeetingSummary) error {
	if strings.TrimSpace(summary.ID) == "" {
		return fmt.Errorf("meeting missing id")
	}
	if strings.TrimSpace(summary.CreatedAt) == "" {
		return fmt.Errorf("meeting %s missing createdAt", summary.ID)
	}
	return nil
}

func CalendarEventIDFromSummary(summary MeetingSummary) string {
	if s := strings.TrimSpace(summary.IcalUID); s != "" {
		return s
	}
	if summary.CalendarEvent != nil {
		if s := strings.TrimSpace(summary.CalendarEvent.IcalUID); s != "" {
			return s
		}
		if s := strings.TrimSpace(summary.CalendarEvent.PlatformID); s != "" {
			return s
		}
	}
	return ""
}

func WindowFromSummary(summary MeetingSummary) (time.Time, time.Time, error) {
	start, err := parseRFC3339(summary.CreatedAt)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("createdAt: %w", err)
	}
	end := time.Time{}
	if summary.Duration > 0 {
		end = start.Add(time.Duration(summary.Duration * float64(time.Second)))
	}
	return start, end, nil
}

func RevisionFromSummary(summary MeetingSummary) string {
	if s := strings.TrimSpace(summary.UpdatedAt); s != "" {
		return s
	}
	if s := strings.TrimSpace(summary.CreatedAt); s != "" {
		return s
	}
	return strings.TrimSpace(summary.ID)
}

func parseRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse %q", value)
}

func MeetingIDString(id string) string {
	return strings.TrimSpace(id)
}

func WindowEndFromTranscript(start time.Time, items []TranscriptItem) (time.Time, error) {
	var maxEnd float64
	for _, item := range items {
		if item.EndSeconds > maxEnd {
			maxEnd = item.EndSeconds
		}
	}
	if maxEnd <= 0 {
		return time.Time{}, fmt.Errorf("transcript segment ends are empty")
	}
	return start.Add(time.Duration(int64(maxEnd * float64(time.Second)))), nil
}
