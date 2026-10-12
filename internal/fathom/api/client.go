package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	fconfig "github.com/TheAngryPit/meetcrawl/internal/fathom/config"
	"github.com/TheAngryPit/meetcrawl/internal/fathom/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
)

type Client interface {
	ListMeetingsForSync(ctx context.Context) ([]MeetingForSync, error)
	TranscriptItems(ctx context.Context, recordingID string) ([]TranscriptItem, error)
}

type Deps struct {
	Secrets   secret.Provider
	APIKeyRef string
}

func New(ctx context.Context, cfg fconfig.Config, fixtureDir string, deps Deps) (Client, error) {
	if strings.TrimSpace(fixtureDir) != "" {
		return NewFixture(fixtureDir)
	}
	keyRef := strings.TrimSpace(deps.APIKeyRef)
	if keyRef == "" {
		keyRef = secret.AccountFathomAPIKey
	}
	if err := secret.ValidateRef(cfg.SecretsProvider, keyRef); err != nil {
		return nil, err
	}
	token, err := deps.Secrets.Resolve(ctx, keyRef)
	if err != nil {
		return nil, fmt.Errorf("fathom api key: %w", err)
	}
	return NewLive(ctx, token)
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

func (c *FixtureClient) ListMeetingsForSync(ctx context.Context) ([]MeetingForSync, error) {
	_ = ctx
	out := make([]MeetingForSync, 0, len(c.manifest.Meetings))
	for _, entry := range c.manifest.Meetings {
		summary := MeetingSummary{
			Title:              entry.Title,
			RecordingID:        entry.RecordingID,
			CreatedAt:          entry.CreatedAt,
			RecordingStartTime: entry.RecordingStartTime,
			RecordingEndTime:   entry.RecordingEndTime,
		}
		if err := ValidateMeetingSummary(summary); err != nil {
			return nil, err
		}
		start, end, err := WindowFromSummary(summary)
		if err != nil {
			return nil, err
		}
		if end.IsZero() {
			if parsedEnd, pErr := fixture.ParseRFC3339(entry.RecordingEndTime); pErr == nil {
				end = parsedEnd
			}
		}
		out = append(out, MeetingForSync{
			Summary:  summary,
			Start:    start,
			End:      end,
			Revision: RevisionFromSummary(summary),
		})
	}
	return out, nil
}

func (c *FixtureClient) TranscriptItems(ctx context.Context, recordingID string) ([]TranscriptItem, error) {
	_ = ctx
	want, err := strconv.ParseInt(strings.TrimSpace(recordingID), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid recording id %q", recordingID)
	}
	for _, entry := range c.manifest.Meetings {
		if entry.RecordingID != want {
			continue
		}
		raw, err := fixture.ReadTranscript(c.manifest.Dir, entry.TranscriptPath)
		if err != nil {
			return nil, err
		}
		return ParseTranscriptItems(raw)
	}
	return nil, fmt.Errorf("fixture: unknown recording id %q", recordingID)
}
