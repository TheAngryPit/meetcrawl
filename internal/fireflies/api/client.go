package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	fconfig "github.com/TheAngryPit/meetcrawl/internal/fireflies/config"
	"github.com/TheAngryPit/meetcrawl/internal/fireflies/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
)

type Client interface {
	ListTranscriptsForSync(ctx context.Context) ([]TranscriptForSync, error)
	Sentences(ctx context.Context, transcriptID string) ([]Sentence, error)
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
		keyRef = secret.AccountFirefliesAPIKey
	}
	if err := secret.ValidateRef(cfg.SecretsProvider, keyRef); err != nil {
		return nil, err
	}
	token, err := deps.Secrets.Resolve(ctx, keyRef)
	if err != nil {
		return nil, fmt.Errorf("fireflies api key: %w", err)
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

func (c *FixtureClient) ListTranscriptsForSync(ctx context.Context) ([]TranscriptForSync, error) {
	_ = ctx
	out := make([]TranscriptForSync, 0, len(c.manifest.Transcripts))
	for _, entry := range c.manifest.Transcripts {
		summary := TranscriptSummary{
			ID:         entry.ID,
			Title:      entry.Title,
			DateString: entry.DateString,
			CalendarID: entry.CalendarID,
		}
		if entry.Date != nil {
			summary.Date = float64(*entry.Date)
		}
		if err := ValidateTranscriptSummary(summary); err != nil {
			return nil, err
		}
		start, err := fixture.ParseMeetingStart(entry.Date, entry.DateString)
		if err != nil {
			return nil, err
		}
		var end time.Time
		if entry.Duration > 0 {
			end = start.Add(time.Duration(entry.Duration * float64(time.Minute)))
		}
		rev := strings.TrimSpace(entry.DateString)
		if rev == "" && entry.Date != nil {
			rev = fmt.Sprintf("%d", *entry.Date)
		}
		out = append(out, TranscriptForSync{
			Summary:    summary,
			CalendarID: strings.TrimSpace(entry.CalendarID),
			Start:      start,
			End:        end,
			Revision:   rev,
		})
	}
	return out, nil
}

func (c *FixtureClient) Sentences(ctx context.Context, transcriptID string) ([]Sentence, error) {
	_ = ctx
	for _, entry := range c.manifest.Transcripts {
		if entry.ID != transcriptID {
			continue
		}
		raw, err := fixture.ReadSentences(c.manifest.Dir, entry.TranscriptPath)
		if err != nil {
			return nil, err
		}
		return ParseSentences(raw)
	}
	return nil, fmt.Errorf("fixture: unknown transcript id %q", transcriptID)
}
