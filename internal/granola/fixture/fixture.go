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
	Version int         `json:"version"`
	Dir     string      `json:"-"`
	Notes   []NoteEntry `json:"notes"`
}

type CalendarEvent struct {
	CalendarEventID *string `json:"calendar_event_id,omitempty"`
}

type NoteEntry struct {
	ID             string         `json:"id"`
	Title          string         `json:"title,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at,omitempty"`
	StartDatetime  string         `json:"start_datetime"`
	EndDatetime    string         `json:"end_datetime,omitempty"`
	TranscriptPath string         `json:"transcript_path"`
	CalendarEvent  *CalendarEvent `json:"calendar_event,omitempty"`
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
	if len(m.Notes) == 0 {
		return fmt.Errorf("fixture manifest has no notes")
	}
	ids := make(map[string]struct{})
	for _, note := range m.Notes {
		if strings.TrimSpace(note.ID) == "" {
			return fmt.Errorf("note missing id")
		}
		if strings.TrimSpace(note.CreatedAt) == "" {
			return fmt.Errorf("note %q missing created_at", note.ID)
		}
		if strings.TrimSpace(note.StartDatetime) == "" {
			return fmt.Errorf("note %q missing start_datetime", note.ID)
		}
		if strings.TrimSpace(note.TranscriptPath) == "" {
			return fmt.Errorf("note %q missing transcript_path", note.ID)
		}
		if _, dup := ids[note.ID]; dup {
			return fmt.Errorf("duplicate note id %q", note.ID)
		}
		ids[note.ID] = struct{}{}
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

func ParseWindow(startRaw, endRaw string) (time.Time, time.Time, error) {
	start, err := parseTime(startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	var end time.Time
	if strings.TrimSpace(endRaw) != "" {
		end, err = parseTime(endRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return start, end, nil
}

func parseTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q", raw)
}
