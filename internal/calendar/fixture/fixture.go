package fixture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/calendar"
)

const layoutVersion = 1

type eventsFile struct {
	Version int           `json:"version"`
	Events  []eventRecord `json:"events"`
}

type eventRecord struct {
	ID                string   `json:"id"`
	ICalUID           string   `json:"iCalUID"`
	Start             string   `json:"start"`
	End               string   `json:"end"`
	AttendeeCount     int      `json:"attendeeCount"`
	AttachmentFileIDs []string `json:"attachmentFileIds,omitempty"`
	HangoutLink       string   `json:"hangoutLink,omitempty"`
}

type gmeetManifest struct {
	CalendarEvents []eventRecord `json:"calendar_events"`
}

// LoadDir reads calendar events from dir/events.json or dir/manifest.json (calendar_events).
func LoadDir(dir string) ([]calendar.Event, error) {
	dir = filepath.Clean(dir)
	eventsPath := filepath.Join(dir, "events.json")
	if _, err := os.Stat(eventsPath); err == nil {
		return loadEventsFile(eventsPath)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("calendar fixture: read %s: %w", dir, err)
	}
	var manifest gmeetManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("calendar fixture: parse manifest: %w", err)
	}
	return parseRecords(manifest.CalendarEvents)
}

func loadEventsFile(path string) ([]calendar.Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read calendar events: %w", err)
	}
	var file eventsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse calendar events: %w", err)
	}
	if file.Version != 0 && file.Version != layoutVersion {
		return nil, fmt.Errorf("calendar events version %d unsupported (want %d)", file.Version, layoutVersion)
	}
	return parseRecords(file.Events)
}

func parseRecords(records []eventRecord) ([]calendar.Event, error) {
	out := make([]calendar.Event, 0, len(records))
	for _, rec := range records {
		if strings.TrimSpace(rec.ICalUID) == "" {
			return nil, fmt.Errorf("calendar event %q missing iCalUID", rec.ID)
		}
		start, err := time.Parse(time.RFC3339, rec.Start)
		if err != nil {
			return nil, fmt.Errorf("calendar event %q start: %w", rec.ID, err)
		}
		var end time.Time
		if rec.End != "" {
			end, err = time.Parse(time.RFC3339, rec.End)
			if err != nil {
				return nil, fmt.Errorf("calendar event %q end: %w", rec.ID, err)
			}
		}
		out = append(out, calendar.Event{
			ID:                rec.ID,
			ICalUID:           rec.ICalUID,
			Start:             start.UTC(),
			End:               end.UTC(),
			AttendeeCount:     rec.AttendeeCount,
			AttachmentFileIDs: append([]string(nil), rec.AttachmentFileIDs...),
			HangoutLink:       strings.TrimSpace(rec.HangoutLink),
		})
	}
	return out, nil
}
