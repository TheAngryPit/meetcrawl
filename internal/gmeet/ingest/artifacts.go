package ingest

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/api"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/split"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type StoredRow struct {
	Artifact        source.Artifact
	CalendarEventID string
	Participants    string
	IngestFlags     string
}

type DriveRecord struct {
	FileID       string
	RevisionID   string
	ModifiedTime string
}

func BuildArtifacts(
	ctx context.Context,
	client api.Client,
	cfg gconfig.Config,
	crawlerVersion string,
) ([]StoredRow, []DriveRecord, error) {
	docs, err := client.ListDocs(ctx, cfg.MeetFolderRoots)
	if err != nil {
		return nil, nil, err
	}
	events, err := client.ListCalendarEvents(ctx)
	if err != nil {
		events = nil
	}
	var rows []StoredRow
	var driveRows []DriveRecord
	for _, doc := range docs {
		md, plain, err := client.ExportDocument(ctx, doc.ID)
		if err != nil {
			return nil, nil, err
		}
		parsed, err := split.Parse(md, plain)
		if err != nil {
			return nil, nil, err
		}
		ev := MatchCalendarEvent(events, doc.ID, doc.ModifiedTime)
		window := source.Window{Start: doc.ModifiedTime}
		calendarID := ""
		participants := ""
		if ev != nil {
			window.Start = ev.Start
			window.End = ev.End
			calendarID = ev.ICalUID
			if ev.AttendeeCount > 0 {
				participants = strconv.Itoa(ev.AttendeeCount)
			}
		}
		revision := doc.RevisionID
		if revision == "" {
			revision = doc.ModifiedTime.UTC().Format(time.RFC3339Nano)
		}
		flags := strings.Join(parsed.Flags, ",")
		notesRow, err := artifactRow(doc.ID, revision, source.FidelityNotes, parsed.NotesText, window, calendarID, participants, flags, crawlerVersion)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, notesRow)
		if parsed.TranscriptText != "" {
			transRow, err := artifactRow(doc.ID, revision, source.FidelityTranscript, parsed.TranscriptText, window, calendarID, participants, flags, crawlerVersion)
			if err != nil {
				return nil, nil, err
			}
			rows = append(rows, transRow)
		}
		driveRows = append(driveRows, DriveRecord{
			FileID:       doc.ID,
			RevisionID:   revision,
			ModifiedTime: doc.ModifiedTime.UTC().Format(time.RFC3339Nano),
		})
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("no Meet/Gemini docs matched ingest rules")
	}
	return rows, driveRows, nil
}

func artifactRow(
	docID, revision string,
	fidelity source.Fidelity,
	text string,
	window source.Window,
	calendarID, participants, flags, crawlerVersion string,
) (StoredRow, error) {
	sourceID := docID + "#" + fidelity.String()
	artifact := source.Artifact{
		Identity: source.Identity{
			Kind:     source.KindGMeetGemini,
			SourceID: sourceID,
		},
		SourceRevision: revision,
		Fidelity:       fidelity,
		Privacy:        source.PrivacyPrivate,
		NormalizedText: text,
		Window:         window,
		Language:       "und",
	}
	if err := artifact.Validate(); err != nil {
		return StoredRow{}, err
	}
	if _, err := source.ProvenanceForArtifact(artifact, crawlerVersion); err != nil {
		return StoredRow{}, err
	}
	return StoredRow{
		Artifact:        artifact,
		CalendarEventID: calendarID,
		Participants:    participants,
		IngestFlags:     flags,
	}, nil
}
