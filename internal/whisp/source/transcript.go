package source

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Segment is one stored utterance of an OpenWhispr meeting transcript.
type Segment struct {
	Text         string   `json:"text"`
	Source       string   `json:"source"`
	Speaker      string   `json:"speaker"`
	SpeakerName  string   `json:"speakerName"`
	RecorderName string   `json:"recorderName"`
	Timestamp    *float64 `json:"timestamp"`
}

const (
	sourceMic    = "mic"
	sourceSystem = "system"
)

// ParseSegments reads the segment array OpenWhispr stores in notes.transcript.
// It reports false for plain text (v1.6.0 to v1.6.5) and for anything that is
// not an array of segment objects with a text field, so the caller keeps the
// stored text as it is.
func ParseSegments(raw string) ([]Segment, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "[") {
		return nil, false
	}
	var stored []*struct {
		Segment
		Text *string `json:"text"`
	}
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil, false
	}
	segments := make([]Segment, 0, len(stored))
	for _, s := range stored {
		if s == nil || s.Text == nil {
			return nil, false
		}
		segment := s.Segment
		segment.Text = *s.Text
		segments = append(segments, segment)
	}
	return segments, true
}

// Label names who spoke. A resolved name wins, then the recorder's own name on
// their microphone track, then the diarization label, then the audio track.
// The track names match what other recorders print, so a reader can tell the
// note owner's track from the room.
func (s Segment) Label() string {
	switch {
	case strings.TrimSpace(s.SpeakerName) != "":
		return strings.TrimSpace(s.SpeakerName)
	case s.Source == sourceMic && strings.TrimSpace(s.RecorderName) != "":
		return strings.TrimSpace(s.RecorderName)
	case strings.TrimSpace(s.Speaker) != "":
		return strings.TrimSpace(s.Speaker)
	case s.Source == sourceMic:
		return "Microphone"
	case s.Source == sourceSystem:
		return "System audio"
	}
	return ""
}

// FormatTranscript turns stored segments into one "[MM:SS] Speaker: text" line
// each, in stored order. Plain text and unreadable arrays come back unchanged.
// A segment array with no text yields an empty transcript.
func FormatTranscript(raw string) string {
	segments, ok := ParseSegments(raw)
	if !ok {
		return raw
	}
	lines := make([]string, 0, len(segments))
	for _, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		var prefix string
		if segment.Timestamp != nil && *segment.Timestamp >= 0 {
			prefix = "[" + clock(*segment.Timestamp) + "] "
		}
		if label := segment.Label(); label != "" {
			prefix += label + ": "
		}
		lines = append(lines, prefix+text)
	}
	return strings.Join(lines, "\n")
}

func clock(seconds float64) string {
	total := int(seconds)
	hours, minutes, secs := total/3600, total/60%60, total%60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}
