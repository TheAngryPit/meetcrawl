package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var timestampPattern = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)

func ParseTranscriptItems(raw json.RawMessage) ([]TranscriptItem, error) {
	var wrapper struct {
		Transcript []json.RawMessage `json:"transcript"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Transcript) > 0 {
		return parseTranscriptArray(wrapper.Transcript)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("transcript: expected object or array: %w", err)
	}
	return parseTranscriptArray(arr)
}

func parseTranscriptArray(items []json.RawMessage) ([]TranscriptItem, error) {
	out := make([]TranscriptItem, 0, len(items))
	for i, item := range items {
		parsed, err := parseTranscriptItem(item)
		if err != nil {
			return nil, fmt.Errorf("transcript[%d]: %w", i, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseTranscriptItem(raw json.RawMessage) (TranscriptItem, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return TranscriptItem{}, fmt.Errorf("object: %w", err)
	}
	text, err := requireStringField(probe, "text")
	if err != nil {
		return TranscriptItem{}, err
	}
	ts, err := requireStringField(probe, "timestamp")
	if err != nil {
		return TranscriptItem{}, err
	}
	if !timestampPattern.MatchString(ts) {
		return TranscriptItem{}, fmt.Errorf("timestamp must match HH:MM:SS")
	}
	speakerRaw, ok := probe["speaker"]
	if !ok {
		return TranscriptItem{}, fmt.Errorf("missing \"speaker\"")
	}
	var speakerObj map[string]json.RawMessage
	if err := json.Unmarshal(speakerRaw, &speakerObj); err != nil {
		return TranscriptItem{}, fmt.Errorf("speaker must be object: %w", err)
	}
	displayName, err := requireStringField(speakerObj, "display_name")
	if err != nil {
		return TranscriptItem{}, err
	}
	return TranscriptItem{
		Speaker:   TranscriptItemSpeaker{DisplayName: displayName},
		Text:      text,
		Timestamp: ts,
	}, nil
}

func requireStringField(obj map[string]json.RawMessage, key string) (string, error) {
	raw, ok := obj[key]
	if !ok {
		return "", fmt.Errorf("missing %q", key)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("%q must be string: %w", key, err)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("%q is empty", key)
	}
	return s, nil
}

func ValidateMeetingSummary(summary MeetingSummary) error {
	if summary.RecordingID <= 0 {
		return fmt.Errorf("meeting missing recording_id")
	}
	if strings.TrimSpace(summary.RecordingStartTime) == "" {
		return fmt.Errorf("meeting %d missing recording_start_time", summary.RecordingID)
	}
	return nil
}

func WindowFromSummary(summary MeetingSummary) (time.Time, time.Time, error) {
	start, err := parseRFC3339(summary.RecordingStartTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("recording_start_time: %w", err)
	}
	end := time.Time{}
	if s := strings.TrimSpace(summary.RecordingEndTime); s != "" {
		end, err = parseRFC3339(s)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("recording_end_time: %w", err)
		}
	}
	return start, end, nil
}

func RevisionFromSummary(summary MeetingSummary) string {
	if s := strings.TrimSpace(summary.CreatedAt); s != "" {
		return s
	}
	return strings.TrimSpace(summary.RecordingEndTime)
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

func RecordingIDString(id int64) string {
	return fmt.Sprintf("%d", id)
}

func WindowEndFromTranscript(start time.Time, items []TranscriptItem) (time.Time, error) {
	var maxOffset time.Duration
	for _, item := range items {
		offset, err := parseRelativeTimestamp(item.Timestamp)
		if err != nil {
			return time.Time{}, err
		}
		if offset > maxOffset {
			maxOffset = offset
		}
	}
	if maxOffset <= 0 {
		return time.Time{}, fmt.Errorf("transcript timestamps are empty")
	}
	return start.Add(maxOffset), nil
}

func parseRelativeTimestamp(value string) (time.Duration, error) {
	var h, m, s int
	if _, err := fmt.Sscanf(value, "%d:%d:%d", &h, &m, &s); err != nil {
		return 0, fmt.Errorf("parse timestamp %q", value)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second, nil
}
