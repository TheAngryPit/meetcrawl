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
	Version     int               `json:"version"`
	Dir         string            `json:"-"`
	Transcripts []TranscriptEntry `json:"transcripts"`
}

type TranscriptEntry struct {
	ID             string  `json:"id"`
	Title          string  `json:"title,omitempty"`
	Date           *int64  `json:"date,omitempty"`
	DateString     string  `json:"date_string,omitempty"`
	Duration       float64 `json:"duration_minutes,omitempty"`
	TranscriptPath string  `json:"transcript_path"`
	CalendarID     string  `json:"calendar_id,omitempty"`
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
	if len(m.Transcripts) == 0 {
		return fmt.Errorf("fixture manifest has no transcripts")
	}
	ids := make(map[string]struct{})
	for _, tr := range m.Transcripts {
		if strings.TrimSpace(tr.ID) == "" {
			return fmt.Errorf("transcript missing id")
		}
		if tr.Date == nil && strings.TrimSpace(tr.DateString) == "" {
			return fmt.Errorf("transcript %q missing date or date_string", tr.ID)
		}
		if strings.TrimSpace(tr.TranscriptPath) == "" {
			return fmt.Errorf("transcript %q missing transcript_path", tr.ID)
		}
		if _, dup := ids[tr.ID]; dup {
			return fmt.Errorf("duplicate transcript id %q", tr.ID)
		}
		ids[tr.ID] = struct{}{}
	}
	return nil
}

func ReadSentences(dir, relPath string) (json.RawMessage, error) {
	path := filepath.Join(dir, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read transcript %q: %w", relPath, err)
	}
	return json.RawMessage(data), nil
}

func ParseMeetingStart(dateMs *int64, dateString string) (time.Time, error) {
	if s := strings.TrimSpace(dateString); s != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if ts, err := time.Parse(layout, s); err == nil {
				return ts.UTC(), nil
			}
		}
		return time.Time{}, fmt.Errorf("parse date_string %q", dateString)
	}
	if dateMs != nil && *dateMs > 0 {
		return time.UnixMilli(*dateMs).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("meeting date is missing")
}
