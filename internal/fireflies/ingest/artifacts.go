package ingest

import (
	"context"
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/fireflies/api"
	"github.com/TheAngryPit/meetcrawl/internal/fireflies/archive"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Options struct {
	CrawlerVersion string
}

func BuildArtifacts(ctx context.Context, client api.Client, opts Options) ([]archive.StoredArtifact, error) {
	transcripts, err := client.ListTranscriptsForSync(ctx)
	if err != nil {
		return nil, err
	}
	if len(transcripts) == 0 {
		return nil, fmt.Errorf("no fireflies transcripts returned")
	}
	version := strings.TrimSpace(opts.CrawlerVersion)
	if version == "" {
		version = "fireflyescrawl-dev"
	}
	var out []archive.StoredArtifact
	for _, summary := range transcripts {
		tr := summary.Summary
		sentences, err := client.Sentences(ctx, tr.ID)
		if err != nil {
			return nil, err
		}
		if len(sentences) == 0 {
			return nil, fmt.Errorf("transcript %q has empty sentences", tr.ID)
		}
		text, err := NormalizeTranscript(sentences)
		if err != nil {
			return nil, err
		}
		start, end := summary.Start, summary.End
		if start.IsZero() {
			detail := api.TranscriptDetail{
				ID:         tr.ID,
				Date:       tr.Date,
				DateString: tr.DateString,
				Duration:   0,
			}
			start, end, err = api.WindowFromDetail(detail, sentences)
			if err != nil {
				return nil, err
			}
		} else if end.IsZero() {
			if _, wEnd, wErr := api.WindowFromDetail(api.TranscriptDetail{ID: tr.ID, Date: tr.Date, DateString: tr.DateString}, sentences); wErr == nil {
				end = wEnd
			}
		}
		revision := strings.TrimSpace(summary.Revision)
		if revision == "" {
			revision = api.RevisionFromDetail(api.TranscriptDetail{Date: tr.Date, DateString: tr.DateString})
		}
		artifact := source.Artifact{
			Identity: source.Identity{
				Kind:     source.KindFireflies,
				SourceID: tr.ID,
			},
			SourceRevision: revision,
			Fidelity:       source.FidelityTranscript,
			Privacy:        source.PrivacyPrivate,
			NormalizedText: text,
			Window:         source.Window{Start: start, End: end},
		}
		out = append(out, archive.StoredArtifact{
			Artifact:        artifact,
			CalendarEventID: summary.CalendarID,
		})
	}
	return out, nil
}

func NormalizeTranscript(sentences []api.Sentence) (string, error) {
	var b strings.Builder
	for _, item := range sentences {
		line := strings.TrimSpace(item.Text)
		if line == "" {
			continue
		}
		if speaker := strings.TrimSpace(item.SpeakerName); speaker != "" {
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
