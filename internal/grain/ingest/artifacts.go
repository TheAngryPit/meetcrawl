package ingest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/grain/api"
	"github.com/TheAngryPit/meetcrawl/internal/grain/archive"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Options struct {
	CrawlerVersion string
}

func BuildArtifacts(ctx context.Context, client api.Client, opts Options) ([]archive.StoredArtifact, error) {
	recordings, err := client.ListRecordings(ctx)
	if err != nil {
		return nil, err
	}
	if len(recordings) == 0 {
		return nil, fmt.Errorf("no grain recordings returned")
	}
	version := strings.TrimSpace(opts.CrawlerVersion)
	if version == "" {
		version = "graincrawl-dev"
	}
	var out []archive.StoredArtifact
	for _, summary := range recordings {
		rec := summary.Recording
		segments, err := client.TranscriptJSON(ctx, rec.ID)
		if err != nil {
			return nil, err
		}
		if len(segments) == 0 {
			return nil, fmt.Errorf("recording %q has empty transcript", rec.ID)
		}
		text, err := NormalizeTranscript(segments)
		if err != nil {
			return nil, err
		}
		revision := rec.StartDatetime
		if revision == "" {
			revision = summary.Start.UTC().Format(time.RFC3339Nano)
		}
		window := source.Window{Start: summary.Start, End: summary.End}
		artifact := source.Artifact{
			Identity: source.Identity{
				Kind:     source.KindGrain,
				SourceID: rec.ID,
			},
			SourceRevision: revision,
			Fidelity:       source.FidelityTranscript,
			Privacy:        source.PrivacyPrivate,
			NormalizedText: text,
			Window:         window,
		}
		calendarID := api.ICalUID(rec)
		out = append(out, archive.StoredArtifact{
			Artifact:        artifact,
			CalendarEventID: calendarID,
		})
	}
	return out, nil
}

func NormalizeTranscript(segments []api.TranscriptSegment) (string, error) {
	var b strings.Builder
	for _, seg := range segments {
		line := strings.TrimSpace(seg.Text)
		if line == "" {
			continue
		}
		speaker := strings.TrimSpace(seg.Speaker)
		if speaker != "" {
			line = speaker + ": " + line
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("transcript normalized to empty text")
	}
	return text, nil
}
