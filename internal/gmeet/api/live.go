package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	scopeDriveReadonly    = "https://www.googleapis.com/auth/drive.readonly"
	scopeCalendarReadonly = "https://www.googleapis.com/auth/calendar.events.readonly"
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
	client, err := oauthHTTPClient(ctx, cfg.OAuthClientPath, cfg.TokenPath)
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

func oauthHTTPClient(ctx context.Context, clientPath, tokenPath string) (*http.Client, error) {
	clientJSON, err := os.ReadFile(clientPath)
	if err != nil {
		return nil, fmt.Errorf("read oauth client: %w", err)
	}
	tokenJSON, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("read oauth token: %w", err)
	}
	cfg, err := google.ConfigFromJSON(clientJSON, scopeDriveReadonly, scopeCalendarReadonly)
	if err != nil {
		return nil, fmt.Errorf("parse oauth client: %w", err)
	}
	var token oauth2.Token
	if err := json.Unmarshal(tokenJSON, &token); err != nil {
		return nil, fmt.Errorf("parse oauth token: %w", err)
	}
	return oauth2.NewClient(ctx, cfg.TokenSource(ctx, &token)), nil
}

func (c *LiveClient) ListDocs(ctx context.Context) ([]DriveDoc, error) {
	const pageSize = 100
	var out []DriveDoc
	pageToken := ""
	for {
		call := c.drive.Files.List().
			Q("mimeType='application/vnd.google-apps.document' and trashed=false").
			Fields("nextPageToken, files(id, name, mimeType, modifiedTime, version)").
			PageSize(pageSize).
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		res, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("drive files.list: %w", err)
		}
		for _, f := range res.Files {
			mod, _ := time.Parse(time.RFC3339, f.ModifiedTime)
			out = append(out, DriveDoc{
				ID:           f.Id,
				Name:         f.Name,
				MimeType:     f.MimeType,
				ModifiedTime: mod.UTC(),
				RevisionID:   fmt.Sprintf("%d", f.Version),
			})
		}
		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return out, nil
}

func (c *LiveClient) ExportPlainText(ctx context.Context, fileID string) (string, error) {
	res, err := c.drive.Files.Export(fileID, "text/plain").Context(ctx).Download()
	if err != nil {
		return "", fmt.Errorf("drive export %s: %w", fileID, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("drive export %s is empty", fileID)
	}
	return text, nil
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
