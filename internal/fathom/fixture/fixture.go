package fixture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const LayoutVersion = 1

type Manifest struct {
	Version  int            `json:"version"`
	Dir      string         `json:"-"`
	Meetings []MeetingEntry `json:"meetings"`
}

type MeetingEntry struct {
	RecordingID        int64  `json:"recording_id"`
	Title              string `json:"title,omitempty"`
	CreatedAt          string `json:"created_at"`
	RecordingStartTime string `json:"recording_start_time"`
	RecordingEndTime   string `json:"recording_end_time"`
	TranscriptPath     string `json:"transcript_path"`
}

func LoadDir(dir string) (Manifest, error) {
	dir = filepath.Clean(dir)
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Manifest{}, fmt.Errorf("read fixture manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse fixture manifest: %w", err)
	}
	manifest.Dir = dir
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Validate(m Manifest) error {
	if m.Version != LayoutVersion {
		return fmt.Errorf("fixture layout version %d is unsupported (want %d)", m.Version, LayoutVersion)
	}
	if len(m.Meetings) == 0 {
		return fmt.Errorf("fixture manifest has no meetings")
	}
	ids := make(map[int64]struct{})
	for _, meeting := range m.Meetings {
		if meeting.RecordingID <= 0 {
			return fmt.Errorf("meeting missing recording_id")
		}
		if strings.TrimSpace(meeting.RecordingStartTime) == "" {
			return fmt.Errorf("meeting %d missing recording_start_time", meeting.RecordingID)
		}
		if strings.TrimSpace(meeting.TranscriptPath) == "" {
			return fmt.Errorf("meeting %d missing transcript_path", meeting.RecordingID)
		}
		if _, dup := ids[meeting.RecordingID]; dup {
			return fmt.Errorf("duplicate recording_id %d", meeting.RecordingID)
		}
		ids[meeting.RecordingID] = struct{}{}
	}
	return nil
}

func ReadTranscript(dir, relPath string) (json.RawMessage, error) {
	path := filepath.Join(dir, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read transcript %q: %w", relPath, err)
	}
	return json.RawMessage(data), nil
}

func ParseRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse timestamp %q", value)
}
