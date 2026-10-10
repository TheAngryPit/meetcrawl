package api

import (
	"context"
	"fmt"
	"strings"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/grain/config"
	"github.com/TheAngryPit/meetcrawl/internal/grain/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
)

type Client interface {
	ListRecordings(ctx context.Context) ([]RecordingSummary, error)
	TranscriptJSON(ctx context.Context, recordingID string) ([]TranscriptSegment, error)
}

type Deps struct {
	Secrets secret.Provider
	PATRef  string
}

func New(ctx context.Context, cfg gconfig.Config, fixtureDir string, deps Deps) (Client, error) {
	if strings.TrimSpace(fixtureDir) != "" {
		return NewFixture(fixtureDir)
	}
	patRef := strings.TrimSpace(deps.PATRef)
	if patRef == "" {
		patRef = secret.AccountGrainPAT
	}
	if err := secret.ValidateRef(cfg.SecretsProvider, patRef); err != nil {
		return nil, err
	}
	token, err := deps.Secrets.Resolve(ctx, patRef)
	if err != nil {
		return nil, fmt.Errorf("grain api token: %w", err)
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

func (c *FixtureClient) ListRecordings(ctx context.Context) ([]RecordingSummary, error) {
	_ = ctx
	out := make([]RecordingSummary, 0, len(c.manifest.Recordings))
	for _, rec := range c.manifest.Recordings {
		start, end, err := fixture.ParseWindow(rec.StartDatetime, rec.EndDatetime)
		if err != nil {
			return nil, err
		}
		var cal *CalendarEvent
		if rec.CalendarEvent != nil && rec.CalendarEvent.ICalUID != nil {
			uid := strings.TrimSpace(*rec.CalendarEvent.ICalUID)
			if uid != "" {
				cal = &CalendarEvent{ICalUID: &uid}
			}
		}
		apiRec := Recording{
			ID:            rec.ID,
			Title:         rec.Title,
			StartDatetime: rec.StartDatetime,
			EndDatetime:   rec.EndDatetime,
			CalendarEvent: cal,
		}
		if err := ValidateRecording(apiRec); err != nil {
			return nil, err
		}
		out = append(out, RecordingSummary{Recording: apiRec, Start: start, End: end})
	}
	return out, nil
}

func (c *FixtureClient) TranscriptJSON(ctx context.Context, recordingID string) ([]TranscriptSegment, error) {
	_ = ctx
	for _, rec := range c.manifest.Recordings {
		if rec.ID != recordingID {
			continue
		}
		raw, err := fixture.ReadTranscript(c.manifest.Dir, rec.TranscriptPath)
		if err != nil {
			return nil, err
		}
		return ParseTranscriptSegments(raw)
	}
	return nil, fmt.Errorf("fixture: unknown recording id %q", recordingID)
}
