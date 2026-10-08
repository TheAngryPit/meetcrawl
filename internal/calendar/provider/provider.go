package provider

import (
	"context"
	"os"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/calendar"
	calfixture "github.com/TheAngryPit/meetcrawl/internal/calendar/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/api"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
)

// Options selects how meetcrawl index loads calendar events (optional enrichment).
type Options struct {
	FixtureDir      string
	OAuthClientPath string
	TokenPath       string
}

// ListEvents returns calendar events or nil when enrichment is unavailable (no auth, no fixture).
func ListEvents(ctx context.Context, opts Options) ([]calendar.Event, error) {
	fixtureDir := strings.TrimSpace(opts.FixtureDir)
	if fixtureDir != "" {
		return calfixture.LoadDir(fixtureDir)
	}
	clientPath := strings.TrimSpace(opts.OAuthClientPath)
	tokenPath := strings.TrimSpace(opts.TokenPath)
	if clientPath == "" || tokenPath == "" {
		return nil, nil
	}
	if _, err := os.Stat(clientPath); err != nil {
		return nil, nil
	}
	if _, err := os.Stat(tokenPath); err != nil {
		return nil, nil
	}
	cfg := gconfig.Config{
		OAuthClientPath: clientPath,
		TokenPath:       tokenPath,
	}
	client, err := api.NewLive(ctx, cfg)
	if err != nil {
		return nil, nil
	}
	raw, err := client.ListCalendarEvents(ctx)
	if err != nil {
		return nil, nil
	}
	return fromAPI(raw), nil
}

func fromAPI(raw []api.CalendarEvent) []calendar.Event {
	out := make([]calendar.Event, 0, len(raw))
	for _, ev := range raw {
		out = append(out, calendar.Event{
			ID:                ev.ID,
			ICalUID:           ev.ICalUID,
			Start:             ev.Start,
			End:               ev.End,
			AttendeeCount:     ev.AttendeeCount,
			AttachmentFileIDs: append([]string(nil), ev.AttachmentIDs...),
		})
	}
	return out
}
