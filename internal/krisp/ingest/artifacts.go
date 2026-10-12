package ingest

import (
	"context"
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/krisp/api"
	"github.com/TheAngryPit/meetcrawl/internal/krisp/archive"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Options struct {
	CrawlerVersion string
}

func BuildArtifacts(ctx context.Context, client api.Client, opts Options) ([]archive.StoredArtifact, error) {
	meetings, err := client.ListMeetingsForSync(ctx)
	if err != nil {
		return nil, err
	}
	if len(meetings) == 0 {
		return nil, fmt.Errorf("no krisp meetings returned")
	}
	version := strings.TrimSpace(opts.CrawlerVersion)
	if version == "" {
		version = "krispcrawl-dev"
	}
	var out []archive.StoredArtifact
	for _, meeting := range meetings {
		meetingID := api.MeetingIDString(meeting.Summary.ID)
		items, err := client.TranscriptItems(ctx, meetingID)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("meeting %q has empty transcript", meetingID)
		}
		text, err := NormalizeTranscript(items)
		if err != nil {
			return nil, err
		}
		start, end := meeting.Start, meeting.End
		if end.IsZero() {
			if wEnd, wErr := api.WindowEndFromTranscript(start, items); wErr == nil {
				end = wEnd
			}
		}
		revision := strings.TrimSpace(meeting.Revision)
		if revision == "" {
			revision = api.RevisionFromSummary(meeting.Summary)
		}
		artifact := source.Artifact{
			Identity: source.Identity{
				Kind:     source.KindKrisp,
				SourceID: meetingID,
			},
			SourceRevision: revision,
			Fidelity:       source.FidelityTranscript,
			Privacy:        source.PrivacyPrivate,
			NormalizedText: text,
			Window:         source.Window{Start: start, End: end},
		}
		out = append(out, archive.StoredArtifact{Artifact: artifact})
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
		if speaker := strings.TrimSpace(item.SpeakerDisplay); speaker != "" {
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
