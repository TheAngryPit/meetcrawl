package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParseTranscriptSegments(raw json.RawMessage) ([]TranscriptSegment, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("transcript: expected array: %w", err)
	}
	out := make([]TranscriptSegment, 0, len(arr))
	for i, item := range arr {
		seg, err := parseTranscriptSegment(item)
		if err != nil {
			return nil, fmt.Errorf("transcript segment[%d]: %w", i, err)
		}
		out = append(out, seg)
	}
	return out, nil
}

func parseTranscriptSegment(raw json.RawMessage) (TranscriptSegment, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return TranscriptSegment{}, fmt.Errorf("object: %w", err)
	}
	start, err := requireIntField(probe, "start")
	if err != nil {
		return TranscriptSegment{}, err
	}
	end, err := requireIntField(probe, "end")
	if err != nil {
		return TranscriptSegment{}, err
	}
	text, err := requireStringField(probe, "text")
	if err != nil {
		return TranscriptSegment{}, err
	}
	speaker, _ := optionalStringField(probe, "speaker")
	var participantID *string
	if rawPID, ok := probe["participant_id"]; ok && string(rawPID) != "null" {
		var pid string
		if err := json.Unmarshal(rawPID, &pid); err != nil {
			return TranscriptSegment{}, fmt.Errorf("participant_id: %w", err)
		}
		pid = strings.TrimSpace(pid)
		if pid != "" {
			participantID = &pid
		}
	}
	return TranscriptSegment{
		Start:         start,
		End:           end,
		Text:          text,
		Speaker:       speaker,
		ParticipantID: participantID,
	}, nil
}

func requireIntField(obj map[string]json.RawMessage, key string) (int64, error) {
	raw, ok := obj[key]
	if !ok {
		return 0, fmt.Errorf("missing %q", key)
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, fmt.Errorf("%q must be integer: %w", key, err)
	}
	return n, nil
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

func ValidateRecording(rec Recording) error {
	if strings.TrimSpace(rec.ID) == "" {
		return fmt.Errorf("recording missing id")
	}
	if strings.TrimSpace(rec.StartDatetime) == "" {
		return fmt.Errorf("recording %q missing start_datetime", rec.ID)
	}
	return nil
}

func ICalUID(rec Recording) string {
	if rec.CalendarEvent == nil || rec.CalendarEvent.ICalUID == nil {
		return ""
	}
	return strings.TrimSpace(*rec.CalendarEvent.ICalUID)
}
