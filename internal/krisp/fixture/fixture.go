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
	ID             string `json:"id"`
	Title          string `json:"title,omitempty"`
	StartedAt      string `json:"started_at"`
	Duration       *int   `json:"duration,omitempty"`
	TranscriptPath string `json:"transcript_path"`
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
	ids := make(map[string]struct{})
	for _, meeting := range m.Meetings {
		if strings.TrimSpace(meeting.ID) == "" {
			return fmt.Errorf("meeting missing id")
		}
		if strings.TrimSpace(meeting.StartedAt) == "" {
			return fmt.Errorf("meeting %s missing started_at", meeting.ID)
		}
		if strings.TrimSpace(meeting.TranscriptPath) == "" {
			return fmt.Errorf("meeting %s missing transcript_path", meeting.ID)
		}
		if _, dup := ids[meeting.ID]; dup {
			return fmt.Errorf("duplicate meeting id %s", meeting.ID)
		}
		ids[meeting.ID] = struct{}{}
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
