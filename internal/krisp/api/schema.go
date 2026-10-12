package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseTranscriptDocument(raw json.RawMessage) ([]TranscriptItem, error) {
	var doc TranscriptDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("transcript: %w", err)
	}
	if len(doc.Segments) == 0 {
		return nil, fmt.Errorf("transcript has no segments")
	}
	out := make([]TranscriptItem, 0, len(doc.Segments))
	for i, seg := range doc.Segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			return nil, fmt.Errorf("segment[%d]: text is empty", i)
		}
		if seg.End <= seg.Start {
			return nil, fmt.Errorf("segment[%d]: end must be after start", i)
		}
		out = append(out, TranscriptItem{
			SpeakerDisplay: speakerDisplay(doc.Speakers, seg.Speaker),
			Text:           text,
			StartSeconds:   seg.Start,
			EndSeconds:     seg.End,
		})
	}
	return out, nil
}

func ParseTranscriptItems(raw json.RawMessage) ([]TranscriptItem, error) {
	var wrapper struct {
		Transcript json.RawMessage `json:"transcript"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Transcript) > 0 {
		return ParseTranscriptDocument(wrapper.Transcript)
	}
	return ParseTranscriptDocument(raw)
}

func speakerDisplay(speakers map[string]Participant, index int) string {
	if speakers != nil {
		if p, ok := speakers[strconv.Itoa(index)]; ok {
			name := strings.TrimSpace(strings.Join([]string{
				strings.TrimSpace(p.FirstName),
				strings.TrimSpace(p.LastName),
			}, " "))
			if name != "" {
				return name
			}
		}
	}
	return fmt.Sprintf("Speaker %d", index)
}

func ValidateMeetingSummary(summary MeetingSummary) error {
	if strings.TrimSpace(summary.ID) == "" {
		return fmt.Errorf("meeting missing id")
	}
	if strings.TrimSpace(summary.StartedAt) == "" {
		return fmt.Errorf("meeting %s missing started_at", summary.ID)
	}
	return nil
}

func WindowFromSummary(summary MeetingSummary) (time.Time, time.Time, error) {
	start, err := parseRFC3339(summary.StartedAt)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("started_at: %w", err)
	}
	end := time.Time{}
	if summary.Duration != nil && *summary.Duration > 0 {
		end = start.Add(time.Duration(*summary.Duration) * time.Second)
	}
	return start, end, nil
}

func RevisionFromSummary(summary MeetingSummary) string {
	if s := strings.TrimSpace(summary.StartedAt); s != "" {
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
