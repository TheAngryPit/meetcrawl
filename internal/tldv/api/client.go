package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/secret"
	cconfig "github.com/TheAngryPit/meetcrawl/internal/tldv/config"
	"github.com/TheAngryPit/meetcrawl/internal/tldv/fixture"
)

type Client interface {
	ListMeetingsForSync(ctx context.Context) ([]MeetingForSync, error)
	TranscriptItems(ctx context.Context, meetingID string) ([]TranscriptItem, error)
}

type Deps struct {
	Secrets   secret.Provider
	APIKeyRef string
}

func New(ctx context.Context, cfg cconfig.Config, fixtureDir string, deps Deps) (Client, error) {
	if strings.TrimSpace(fixtureDir) != "" {
		return NewFixture(fixtureDir)
	}
	keyRef := strings.TrimSpace(deps.APIKeyRef)
	if keyRef == "" {
		keyRef = secret.AccountTldvAPIKey
	}
	if err := secret.ValidateRef(cfg.SecretsProvider, keyRef); err != nil {
		return nil, err
	}
	token, err := deps.Secrets.Resolve(ctx, keyRef)
	if err != nil {
		return nil, fmt.Errorf("tldv api key: %w", err)
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
			ID:         entry.ID,
			Name:       entry.Title,
			HappenedAt: entry.StartedAt,
		}
		if entry.Duration != nil {
			summary.Duration = float64(*entry.Duration)
		}
		if err := ValidateMeetingSummary(summary); err != nil {
			return nil, err
		}
		start, end, err := WindowFromSummary(summary)
		if err != nil {
			return nil, err
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

func (c *FixtureClient) TranscriptItems(ctx context.Context, meetingID string) ([]TranscriptItem, error) {
	_ = ctx
	want := strings.TrimSpace(meetingID)
	for _, entry := range c.manifest.Meetings {
		if entry.ID != want {
			continue
		}
		raw, err := fixture.ReadTranscript(c.manifest.Dir, entry.TranscriptPath)
		if err != nil {
			return nil, err
		}
		return ParseTranscriptItems(raw)
	}
	return nil, fmt.Errorf("fixture: unknown meeting id %q", meetingID)
}
