package source

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/source"
)

// MeetingNote is one OpenWhispr notes row selected for archive ingest.

type MeetingNote struct {
	ID              int64
	Transcript      string
	Content         string
	EnhancedContent string
	SourceRevision  string
	CalendarEventID string
	Participants    string
	Start           time.Time
	End             time.Time
}

func ReadMeetingNotes(ctx context.Context, db *sql.DB) ([]MeetingNote, error) {
	const query = `
select
  id,
  coalesce(transcript, ''),
  coalesce(content, ''),
  coalesce(enhanced_content, ''),
  coalesce(updated_at, ''),
  coalesce(calendar_event_id, ''),
  coalesce(participants, ''),
  coalesce(created_at, '')
from notes
where note_type = 'meeting'
  and (deleted_at is null or trim(deleted_at) = '')
`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query meeting notes: %w", err)
	}
	defer rows.Close()
	var notes []MeetingNote
	for rows.Next() {
		var note MeetingNote
		var updatedAt, createdAt string
		if err := rows.Scan(
			&note.ID,
			&note.Transcript,
			&note.Content,
			&note.EnhancedContent,
			&updatedAt,
			&note.CalendarEventID,
			&note.Participants,
			&createdAt,
		); err != nil {
			return nil, err
		}
		note.SourceRevision = strings.TrimSpace(updatedAt)
		note.Start = parseOpenWhisprTime(createdAt)
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func ArtifactsForNote(note MeetingNote, crawlerVersion string) ([]source.Artifact, error) {
	type field struct {
		text     string
		fidelity source.Fidelity
		suffix   string
	}
	fields := []field{
		{text: note.Transcript, fidelity: source.FidelityTranscript, suffix: "transcript"},
		{text: note.Content, fidelity: source.FidelityNotes, suffix: "notes"},
		{text: note.EnhancedContent, fidelity: source.FidelitySummary, suffix: "summary"},
	}
	var artifacts []source.Artifact
	for _, f := range fields {
		text := strings.TrimSpace(f.text)
		if text == "" {
			continue
		}
		artifact := source.Artifact{
			Identity: source.Identity{
				Kind:     source.KindOpenWhispr,
				SourceID: fmt.Sprintf("%d:%s", note.ID, f.suffix),
			},
			SourceRevision: note.SourceRevision,
			Fidelity:       f.fidelity,
			Privacy:        source.PrivacyPrivate,
			NormalizedText: text,
			Window: source.Window{
				Start: note.Start,
				End:   note.End,
			},
			Language: "und",
		}
		if err := artifact.Validate(); err != nil {
			return nil, err
		}
		if _, err := source.ProvenanceForArtifact(artifact, crawlerVersion); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func parseOpenWhisprTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC()
		}
	}
	return time.Time{}
}
