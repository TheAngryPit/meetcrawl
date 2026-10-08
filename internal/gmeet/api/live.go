package api

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type LiveClient struct {
	drive    *drive.Service
	calendar *calendar.Service
}

func NewLive(ctx context.Context, cfg gconfig.Config) (*LiveClient, error) {
	if strings.TrimSpace(cfg.OAuthClientPath) == "" {
		return nil, fmt.Errorf("oauth_client_path is required for live sync (or use --fixture)")
	}
	if strings.TrimSpace(cfg.TokenPath) == "" {
		return nil, fmt.Errorf("token_path is required for live sync (or use --fixture)")
	}
	store := oauth.DefaultTokenStore(cfg.TokenPath)
	client, err := oauth.HTTPClient(ctx, cfg.OAuthClientPath, store)
	if err != nil {
		return nil, err
	}
	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("drive service: %w", err)
	}
	calSvc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("calendar service: %w", err)
	}
	return &LiveClient{drive: driveSvc, calendar: calSvc}, nil
}

func (c *LiveClient) ListDocs(ctx context.Context, folderRoots []string) ([]DriveDoc, error) {
	byID, err := c.collectDocsUnderRoots(ctx, folderRoots)
	if err != nil {
		return nil, err
	}
	out := make([]DriveDoc, 0, len(byID))
	for _, doc := range byID {
		out = append(out, doc)
	}
	return out, nil
}

func (c *LiveClient) ExportDocument(ctx context.Context, fileID string) (string, string, error) {
	md, err := c.exportBytes(ctx, fileID, "text/markdown")
	if err != nil {
		return "", "", fmt.Errorf("drive export %s: %w", fileID, err)
	}
	if strings.TrimSpace(md) == "" {
		return "", "", fmt.Errorf("drive export %s is empty", fileID)
	}
	return md, "", nil
}

func (c *LiveClient) ExportPlainText(ctx context.Context, fileID string) (string, error) {
	plain, err := c.exportBytes(ctx, fileID, "text/plain")
	if err != nil {
		return "", fmt.Errorf("drive export %s: %w", fileID, err)
	}
	if strings.TrimSpace(plain) == "" {
		return "", fmt.Errorf("drive export %s is empty", fileID)
	}
	return plain, nil
}

func (c *LiveClient) exportBytes(ctx context.Context, fileID, mimeType string) (string, error) {
	res, err := c.drive.Files.Export(fileID, mimeType).Context(ctx).Download()
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *LiveClient) ListCalendarEvents(ctx context.Context) ([]CalendarEvent, error) {
	now := time.Now().UTC()
	timeMin := now.AddDate(-2, 0, 0).Format(time.RFC3339)
	timeMax := now.AddDate(0, 1, 0).Format(time.RFC3339)
	var out []CalendarEvent
	pageToken := ""
	for {
		call := c.calendar.Events.List("primary").
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(timeMin).
			TimeMax(timeMax).
			MaxResults(250).
			Fields("nextPageToken, items(id, iCalUID, start, end, attendees, attachments(fileId))").
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		res, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("calendar events.list: %w", err)
		}
		for _, item := range res.Items {
			if item == nil {
				continue
			}
			start, end, err := eventWindow(item)
			if err != nil {
				continue
			}
			var attach []string
			for _, a := range item.Attachments {
				if a != nil && a.FileId != "" {
					attach = append(attach, a.FileId)
				}
			}
			attendees := 0
			if item.Attendees != nil {
				attendees = len(item.Attendees)
			}
			out = append(out, CalendarEvent{
				ID:            item.Id,
				ICalUID:       item.ICalUID,
				Start:         start,
				End:           end,
				AttendeeCount: attendees,
				AttachmentIDs: attach,
			})
		}
		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return out, nil
}

func eventWindow(item *calendar.Event) (time.Time, time.Time, error) {
	if item.Start == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("missing start")
	}
	startRaw := item.Start.DateTime
	if startRaw == "" {
		startRaw = item.Start.Date
	}
	start, err := time.Parse(time.RFC3339, startRaw)
	if err != nil {
		if t, e2 := time.Parse("2006-01-02", startRaw); e2 == nil {
			start = t.UTC()
		} else {
			return time.Time{}, time.Time{}, err
		}
	}
	var end time.Time
	if item.End != nil {
		endRaw := item.End.DateTime
		if endRaw == "" {
			endRaw = item.End.Date
		}
		if endRaw != "" {
			end, err = time.Parse(time.RFC3339, endRaw)
			if err != nil {
				if t, e2 := time.Parse("2006-01-02", endRaw); e2 == nil {
					end = t.UTC()
				}
			}
		}
	}
	return start.UTC(), end.UTC(), nil
}
