package ingest

import (
	"context"
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/granola/api"
	"github.com/TheAngryPit/meetcrawl/internal/granola/archive"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Options struct {
	CrawlerVersion string
}

func BuildArtifacts(ctx context.Context, client api.Client, opts Options) ([]archive.StoredArtifact, error) {
	notes, err := client.ListNotesForSync(ctx)
	if err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return nil, fmt.Errorf("no granola notes returned")
	}
	version := strings.TrimSpace(opts.CrawlerVersion)
	if version == "" {
		version = "granolacrawl-dev"
	}
	var out []archive.StoredArtifact
	for _, summary := range notes {
		note := summary.Note
		items, err := client.TranscriptItems(ctx, note.ID)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("note %q has empty transcript", note.ID)
		}
		text, err := NormalizeTranscript(items)
		if err != nil {
			return nil, err
		}
		start, end := summary.Start, summary.End
		if start.IsZero() {
			start, end, err = api.WindowFromTranscript(items)
			if err != nil {
				return nil, err
			}
		} else if end.IsZero() {
			tStart, tEnd, wErr := api.WindowFromTranscript(items)
			if wErr == nil && !tEnd.IsZero() {
				end = tEnd
			}
			_ = tStart
		}
		revision := strings.TrimSpace(note.UpdatedAt)
		if revision == "" {
			revision = note.CreatedAt
		}
		window := source.Window{Start: start, End: end}
		artifact := source.Artifact{
			Identity: source.Identity{
				Kind:     source.KindGranola,
				SourceID: note.ID,
			},
			SourceRevision: revision,
			Fidelity:       source.FidelityTranscript,
			Privacy:        source.PrivacyPrivate,
			NormalizedText: text,
			Window:         window,
		}
		out = append(out, archive.StoredArtifact{
			Artifact:        artifact,
			CalendarEventID: summary.CalendarEventID,
		})
	}
	return out, nil
}

func NormalizeTranscript(items []api.TranscriptItem) (string, error) {
	var b strings.Builder
	for _, item := range items {
		line := strings.TrimSpace(item.Text)
		if line == "" {
			continue
		}
		speaker := formatSpeaker(item.Speaker)
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

func formatSpeaker(sp api.Speaker) string {
	if name := strings.TrimSpace(sp.Name); name != "" {
		return name
	}
	if label := strings.TrimSpace(sp.DiarizationLabel); label != "" {
		return label
	}
	if sp.Attribution == "me" {
		return "Me"
	}
	if sp.Attribution == "them" {
		return "Them"
	}
	switch sp.Source {
	case "microphone":
		return "Microphone"
	case "speaker":
		return "Speaker"
	default:
		return ""
	}
}
