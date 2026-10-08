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
	Version        int             `json:"version"`
	DriveFiles     []DriveFile     `json:"drive_files"`
	CalendarEvents []CalendarEvent `json:"calendar_events"`
}

type DriveFile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType"`
	ModifiedTime string `json:"modifiedTime"`
	RevisionID   string `json:"revisionId"`
	ExportPath   string `json:"exportPath"`
}

type CalendarEvent struct {
	ID                string   `json:"id"`
	ICalUID           string   `json:"iCalUID"`
	Start             string   `json:"start"`
	End               string   `json:"end"`
	AttendeeCount     int      `json:"attendeeCount"`
	AttachmentFileIDs []string `json:"attachmentFileIds,omitempty"`
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
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	for i := range manifest.DriveFiles {
		if manifest.DriveFiles[i].ExportPath != "" && !filepath.IsAbs(manifest.DriveFiles[i].ExportPath) {
			manifest.DriveFiles[i].ExportPath = filepath.Join(dir, manifest.DriveFiles[i].ExportPath)
		}
	}
	return manifest, nil
}

func Validate(m Manifest) error {
	if m.Version != LayoutVersion {
		return fmt.Errorf("fixture layout version %d is unsupported (want %d)", m.Version, LayoutVersion)
	}
	if len(m.DriveFiles) == 0 {
		return fmt.Errorf("fixture manifest has no drive_files")
	}
	ids := make(map[string]struct{})
	for _, f := range m.DriveFiles {
		if strings.TrimSpace(f.ID) == "" {
			return fmt.Errorf("drive file missing id")
		}
		if strings.TrimSpace(f.Name) == "" {
			return fmt.Errorf("drive file %q missing name", f.ID)
		}
		if f.MimeType != "" && f.MimeType != "application/vnd.google-apps.document" {
			return fmt.Errorf("drive file %q has unexpected mimeType %q", f.ID, f.MimeType)
		}
		if strings.TrimSpace(f.ExportPath) == "" {
			return fmt.Errorf("drive file %q missing exportPath", f.ID)
		}
		if _, dup := ids[f.ID]; dup {
			return fmt.Errorf("duplicate drive file id %q", f.ID)
		}
		ids[f.ID] = struct{}{}
	}
	for _, ev := range m.CalendarEvents {
		if strings.TrimSpace(ev.ICalUID) == "" {
			return fmt.Errorf("calendar event %q missing iCalUID", ev.ID)
		}
		if _, err := time.Parse(time.RFC3339, ev.Start); err != nil {
			return fmt.Errorf("calendar event %q start: %w", ev.ID, err)
		}
		if ev.End != "" {
			if _, err := time.Parse(time.RFC3339, ev.End); err != nil {
				return fmt.Errorf("calendar event %q end: %w", ev.ID, err)
			}
		}
	}
	return nil
}

func ReadExport(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read export %s: %w", path, err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("export %s is empty", path)
	}
	return text, nil
}
