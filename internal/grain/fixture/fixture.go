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
	Version    int              `json:"version"`
	Dir        string           `json:"-"`
	Recordings []RecordingEntry `json:"recordings"`
}

type CalendarEvent struct {
	ICalUID *string `json:"ical_uid"`
}

type RecordingEntry struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
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
	if len(m.Recordings) == 0 {
		return fmt.Errorf("fixture manifest has no recordings")
	}
	ids := make(map[string]struct{})
	for _, rec := range m.Recordings {
		if strings.TrimSpace(rec.ID) == "" {
			return fmt.Errorf("recording missing id")
		}
		if strings.TrimSpace(rec.StartDatetime) == "" {
			return fmt.Errorf("recording %q missing start_datetime", rec.ID)
		}
		if strings.TrimSpace(rec.TranscriptPath) == "" {
			return fmt.Errorf("recording %q missing transcript_path", rec.ID)
		}
		if _, dup := ids[rec.ID]; dup {
			return fmt.Errorf("duplicate recording id %q", rec.ID)
		}
		ids[rec.ID] = struct{}{}
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
