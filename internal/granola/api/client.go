package api

import (
	"context"
	"fmt"
	"strings"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/granola/config"
	"github.com/TheAngryPit/meetcrawl/internal/granola/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
)

type Client interface {
	ListNotesForSync(ctx context.Context) ([]NoteForSync, error)
	TranscriptItems(ctx context.Context, noteID string) ([]TranscriptItem, error)
}

type Deps struct {
	Secrets   secret.Provider
	APIKeyRef string
}

func New(ctx context.Context, cfg gconfig.Config, fixtureDir string, deps Deps) (Client, error) {
	if strings.TrimSpace(fixtureDir) != "" {
		return NewFixture(fixtureDir)
	}
	keyRef := strings.TrimSpace(deps.APIKeyRef)
	if keyRef == "" {
		keyRef = secret.AccountGranolaAPIKey
	}
	if err := secret.ValidateRef(cfg.SecretsProvider, keyRef); err != nil {
		return nil, err
	}
	token, err := deps.Secrets.Resolve(ctx, keyRef)
	if err != nil {
		return nil, fmt.Errorf("granola api key: %w", err)
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

func (c *FixtureClient) ListNotesForSync(ctx context.Context) ([]NoteForSync, error) {
	_ = ctx
	out := make([]NoteForSync, 0, len(c.manifest.Notes))
	for _, entry := range c.manifest.Notes {
		note := NoteSummary{
			ID:        entry.ID,
			Title:     entry.Title,
			CreatedAt: entry.CreatedAt,
			UpdatedAt: entry.UpdatedAt,
		}
		if note.UpdatedAt == "" {
			note.UpdatedAt = entry.CreatedAt
		}
		if err := ValidateNoteSummary(note); err != nil {
			return nil, err
		}
		start, end, err := fixture.ParseWindow(entry.StartDatetime, entry.EndDatetime)
		if err != nil {
			return nil, err
		}
		calID := ""
		if entry.CalendarEvent != nil && entry.CalendarEvent.CalendarEventID != nil {
			calID = strings.TrimSpace(*entry.CalendarEvent.CalendarEventID)
		}
		out = append(out, NoteForSync{
			Note:            note,
			CalendarEventID: calID,
			Start:           start,
			End:             end,
		})
	}
	return out, nil
}

func (c *FixtureClient) TranscriptItems(ctx context.Context, noteID string) ([]TranscriptItem, error) {
	_ = ctx
	for _, entry := range c.manifest.Notes {
		if entry.ID != noteID {
			continue
		}
		raw, err := fixture.ReadTranscript(c.manifest.Dir, entry.TranscriptPath)
		if err != nil {
			return nil, err
		}
		return ParseTranscriptItems(raw)
	}
	return nil, fmt.Errorf("fixture: unknown note id %q", noteID)
}
