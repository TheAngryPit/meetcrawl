package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/detect"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/fixture"
)

type DriveDoc struct {
	ID           string
	Name         string
	MimeType     string
	ModifiedTime time.Time
	RevisionID   string
	ParentPath   string
}

type CalendarEvent struct {
	ID            string
	ICalUID       string
	Start         time.Time
	End           time.Time
	AttendeeCount int
	AttachmentIDs []string
}

type Client interface {
	ListDocs(ctx context.Context, folderRoots []string) ([]DriveDoc, error)
	ExportDocument(ctx context.Context, fileID string) (markdown string, plain string, err error)
	ListCalendarEvents(ctx context.Context) ([]CalendarEvent, error)
}

func New(ctx context.Context, cfg gconfig.Config, fixtureDir string) (Client, error) {
	if fixtureDir != "" {
		return NewFixture(fixtureDir)
	}
	return NewLive(ctx, cfg)
}

type FixtureClient struct {
	manifest fixture.Manifest
}

func NewFixture(dir string) (*FixtureClient, error) {
	manifest, err := fixture.LoadDir(dir)
	if err != nil {
		return nil, err
	}
	return &FixtureClient{manifest: manifest}, nil
}

func (c *FixtureClient) ListDocs(_ context.Context, folderRoots []string) ([]DriveDoc, error) {
	out := make([]DriveDoc, 0, len(c.manifest.DriveFiles))
	for _, f := range c.manifest.DriveFiles {
		if !detect.IsGeminiDocTitle(f.Name) {
			continue
		}
		if !detect.InFolderScope(f.ParentPath, folderRoots) {
			continue
		}
		mod, err := parseTime(f.ModifiedTime)
		if err != nil {
			return nil, fmt.Errorf("drive file %q modifiedTime: %w", f.ID, err)
		}
		mime := f.MimeType
		if mime == "" {
			mime = "application/vnd.google-apps.document"
		}
		out = append(out, DriveDoc{
			ID:           f.ID,
			Name:         f.Name,
			MimeType:     mime,
			ModifiedTime: mod,
			RevisionID:   f.RevisionID,
			ParentPath:   f.ParentPath,
		})
	}
	return out, nil
}

func (c *FixtureClient) ExportDocument(_ context.Context, fileID string) (string, string, error) {
	for _, f := range c.manifest.DriveFiles {
		if f.ID != fileID {
			continue
		}
		var md, plain string
		var err error
		if f.ExportMarkdownPath != "" {
			md, err = fixture.ReadExport(f.ExportMarkdownPath)
			if err != nil {
				return "", "", err
			}
		}
		path := f.ExportPlainPath
		if path == "" {
			path = f.ExportPath
		}
		if path != "" {
			plain, err = fixture.ReadExport(path)
			if err != nil {
				return "", "", err
			}
		}
		if md == "" && plain == "" {
			return "", "", fmt.Errorf("fixture: drive file %q has no export bytes", fileID)
		}
		return md, plain, nil
	}
	return "", "", fmt.Errorf("fixture: unknown drive file id %q", fileID)
}

func (c *FixtureClient) ListCalendarEvents(context.Context) ([]CalendarEvent, error) {
	out := make([]CalendarEvent, 0, len(c.manifest.CalendarEvents))
	for _, ev := range c.manifest.CalendarEvents {
		start, err := parseTime(ev.Start)
		if err != nil {
			return nil, err
		}
		var end time.Time
		if ev.End != "" {
			end, err = parseTime(ev.End)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, CalendarEvent{
			ID:            ev.ID,
			ICalUID:       ev.ICalUID,
			Start:         start,
			End:           end,
			AttendeeCount: ev.AttendeeCount,
			AttachmentIDs: ev.AttachmentFileIDs,
		})
	}
	return out, nil
}

func parseTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q", raw)
}
