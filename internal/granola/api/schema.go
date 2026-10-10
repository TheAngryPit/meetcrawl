package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var noteIDRe = regexp.MustCompile(NoteIDPattern)

func ParseTranscriptItems(raw json.RawMessage) ([]TranscriptItem, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("transcript: expected array: %w", err)
	}
	out := make([]TranscriptItem, 0, len(arr))
	for i, item := range arr {
		seg, err := parseTranscriptItem(item)
		if err != nil {
			return nil, fmt.Errorf("transcript item[%d]: %w", i, err)
		}
		out = append(out, seg)
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
	startTime, err := requireStringField(probe, "start_time")
	if err != nil {
		return TranscriptItem{}, err
	}
	endTime, err := requireStringField(probe, "end_time")
	if err != nil {
		return TranscriptItem{}, err
	}
	speakerRaw, ok := probe["speaker"]
	if !ok {
		return TranscriptItem{}, fmt.Errorf("missing %q", "speaker")
	}
	speaker, err := parseSpeaker(speakerRaw)
	if err != nil {
		return TranscriptItem{}, fmt.Errorf("speaker: %w", err)
	}
	if _, err := parseRFC3339(startTime); err != nil {
		return TranscriptItem{}, fmt.Errorf("start_time: %w", err)
	}
	if _, err := parseRFC3339(endTime); err != nil {
		return TranscriptItem{}, fmt.Errorf("end_time: %w", err)
	}
	return TranscriptItem{
		Speaker:   speaker,
		Text:      text,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}

func parseSpeaker(raw json.RawMessage) (Speaker, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return Speaker{}, fmt.Errorf("object: %w", err)
	}
	source, err := requireStringField(probe, "source")
	if err != nil {
		return Speaker{}, err
	}
	switch source {
	case "microphone", "speaker":
	default:
		return Speaker{}, fmt.Errorf("source %q is unsupported", source)
	}
	out := Speaker{Source: source}
	if rawAttr, ok := probe["attribution"]; ok && string(rawAttr) != "null" {
		attr, err := optionalStringField(probe, "attribution")
		if err != nil {
			return Speaker{}, err
		}
		if attr != "" && attr != "me" && attr != "them" {
			return Speaker{}, fmt.Errorf("attribution %q is unsupported", attr)
		}
		out.Attribution = attr
	}
	out.DiarizationLabel, _ = optionalStringField(probe, "diarization_label")
	out.Name, _ = optionalStringField(probe, "name")
	return out, nil
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

func optionalStringField(obj map[string]json.RawMessage, key string) (string, error) {
	raw, ok := obj[key]
	if !ok {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("%q must be string: %w", key, err)
	}
	return strings.TrimSpace(s), nil
}

func ValidateNoteSummary(note NoteSummary) error {
	id := strings.TrimSpace(note.ID)
	if id == "" {
		return fmt.Errorf("note missing id")
	}
	if !noteIDRe.MatchString(id) {
		return fmt.Errorf("note id %q does not match Granola pattern", id)
	}
	if strings.TrimSpace(note.CreatedAt) == "" {
		return fmt.Errorf("note %q missing created_at", id)
	}
	if _, err := parseRFC3339(note.CreatedAt); err != nil {
		return fmt.Errorf("note %q created_at: %w", id, err)
	}
	return nil
}

func CalendarEventIDFromDetail(d NoteDetail) string {
	if d.CalendarEvent == nil || d.CalendarEvent.CalendarEventID == nil {
		return ""
	}
	return strings.TrimSpace(*d.CalendarEvent.CalendarEventID)
}

func WindowFromDetail(d NoteDetail) (time.Time, time.Time, error) {
	if d.CalendarEvent != nil {
		var start, end time.Time
		var err error
		if d.CalendarEvent.ScheduledStartTime != nil && strings.TrimSpace(*d.CalendarEvent.ScheduledStartTime) != "" {
			start, err = parseRFC3339(*d.CalendarEvent.ScheduledStartTime)
			if err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("scheduled_start_time: %w", err)
			}
		}
		if d.CalendarEvent.ScheduledEndTime != nil && strings.TrimSpace(*d.CalendarEvent.ScheduledEndTime) != "" {
			end, err = parseRFC3339(*d.CalendarEvent.ScheduledEndTime)
			if err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("scheduled_end_time: %w", err)
			}
		}
		if !start.IsZero() {
			return start, end, nil
		}
	}
	start, err := parseRFC3339(d.CreatedAt)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, time.Time{}, nil
}

func WindowFromTranscript(items []TranscriptItem) (time.Time, time.Time, error) {
	var start, end time.Time
	for _, item := range items {
		s, err := parseRFC3339(item.StartTime)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		e, err := parseRFC3339(item.EndTime)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		if start.IsZero() || s.Before(start) {
			start = s
		}
		if end.IsZero() || e.After(end) {
			end = e
		}
	}
	if start.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("transcript has no start times")
	}
	return start, end, nil
}

func parseRFC3339(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q", raw)
}
