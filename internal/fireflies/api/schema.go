package api

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

func ParseSentences(raw json.RawMessage) ([]Sentence, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("sentences: expected array: %w", err)
	}
	out := make([]Sentence, 0, len(arr))
	for i, item := range arr {
		seg, err := parseSentence(item)
		if err != nil {
			return nil, fmt.Errorf("sentence[%d]: %w", i, err)
		}
		out = append(out, seg)
	}
	return out, nil
}

func parseSentence(raw json.RawMessage) (Sentence, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return Sentence{}, fmt.Errorf("object: %w", err)
	}
	text, err := requireStringField(probe, "text")
	if err != nil {
		return Sentence{}, err
	}
	start, err := requireFloatField(probe, "start_time")
	if err != nil {
		return Sentence{}, err
	}
	end, err := requireFloatField(probe, "end_time")
	if err != nil {
		return Sentence{}, err
	}
	if end < start {
		return Sentence{}, fmt.Errorf("end_time before start_time")
	}
	out := Sentence{Text: text, StartTime: start, EndTime: end}
	if rawIdx, ok := probe["index"]; ok && string(rawIdx) != "null" {
		var idx int
		if err := json.Unmarshal(rawIdx, &idx); err != nil {
			return Sentence{}, fmt.Errorf("index must be integer: %w", err)
		}
		out.Index = idx
	}
	out.SpeakerName, _ = optionalStringField(probe, "speaker_name")
	out.SpeakerID, _ = optionalStringField(probe, "speaker_id")
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

func requireFloatField(obj map[string]json.RawMessage, key string) (float64, error) {
	raw, ok := obj[key]
	if !ok {
		return 0, fmt.Errorf("missing %q", key)
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, fmt.Errorf("%q must be number: %w", key, err)
	}
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("%q is not a finite number", key)
	}
	return n, nil
}

func ValidateTranscriptSummary(summary TranscriptSummary) error {
	if strings.TrimSpace(summary.ID) == "" {
		return fmt.Errorf("transcript missing id")
	}
	if summary.Date <= 0 && strings.TrimSpace(summary.DateString) == "" {
		return fmt.Errorf("transcript %q missing date", summary.ID)
	}
	return nil
}

func CalendarIDFromDetail(d TranscriptDetail) string {
	return strings.TrimSpace(d.CalendarID)
}

func MeetingStart(d TranscriptDetail) (time.Time, error) {
	if s := strings.TrimSpace(d.DateString); s != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if ts, err := time.Parse(layout, s); err == nil {
				return ts.UTC(), nil
			}
		}
		return time.Time{}, fmt.Errorf("parse dateString %q", d.DateString)
	}
	if d.Date > 0 {
		ms := int64(d.Date)
		return time.UnixMilli(ms).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("transcript %q missing meeting date", d.ID)
}

func WindowFromDetail(d TranscriptDetail, sentences []Sentence) (time.Time, time.Time, error) {
	start, err := MeetingStart(d)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	var end time.Time
	if d.Duration > 0 {
		end = start.Add(time.Duration(d.Duration * float64(time.Minute)))
	} else if wEnd, wErr := windowEndFromSentences(start, sentences); wErr == nil {
		end = wEnd
	}
	return start, end, nil
}

func windowEndFromSentences(start time.Time, sentences []Sentence) (time.Time, error) {
	var maxEnd float64
	for _, s := range sentences {
		if s.EndTime > maxEnd {
			maxEnd = s.EndTime
		}
	}
	if maxEnd <= 0 {
		return time.Time{}, fmt.Errorf("sentences have no end_time")
	}
	return start.Add(time.Duration(maxEnd * float64(time.Second))), nil
}

func RevisionFromDetail(d TranscriptDetail) string {
	if s := strings.TrimSpace(d.DateString); s != "" {
		return s
	}
	if d.Date > 0 {
		return fmt.Sprintf("%.0f", d.Date)
	}
	return ""
}
