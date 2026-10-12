package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func ParseTranscriptItems(raw json.RawMessage) ([]TranscriptItem, error) {
	var wrapped TranscriptResponse
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Data) > 0 {
		return parseSegmentList(wrapped.Data)
	}
	var segments []TranscriptSegmentRaw
	if err := json.Unmarshal(raw, &segments); err != nil {
		return nil, fmt.Errorf("transcript: expected object with data array: %w", err)
	}
	return parseSegmentList(segments)
}

func parseSegmentList(segments []TranscriptSegmentRaw) ([]TranscriptItem, error) {
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
		if seg.StartTime < 0 {
			return nil, fmt.Errorf("segment[%d]: startTime must be non-negative", i)
		}
		end := seg.EndTime
		if end <= seg.StartTime {
			end = seg.StartTime + 1
		}
		out = append(out, TranscriptItem{
			SpeakerDisplay: speaker,
			Text:           text,
			StartSeconds:   seg.StartTime,
			EndSeconds:     end,
		})
	}
	return out, nil
}

func ValidateMeetingSummary(summary MeetingSummary) error {
	if strings.TrimSpace(summary.ID) == "" {
		return fmt.Errorf("meeting missing id")
	}
	if strings.TrimSpace(summary.HappenedAt) == "" {
		return fmt.Errorf("meeting %s missing happenedAt", summary.ID)
	}
	return nil
}

func WindowFromSummary(summary MeetingSummary) (time.Time, time.Time, error) {
	start, err := parseRFC3339(summary.HappenedAt)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("happenedAt: %w", err)
	}
	end := time.Time{}
	if summary.Duration > 0 {
		end = start.Add(time.Duration(summary.Duration * float64(time.Second)))
	}
	return start, end, nil
}

func RevisionFromSummary(summary MeetingSummary) string {
	if s := strings.TrimSpace(summary.HappenedAt); s != "" {
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
