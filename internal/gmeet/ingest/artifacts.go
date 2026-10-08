package ingest

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/api"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/docclass"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type StoredRow struct {
	Artifact        source.Artifact
	CalendarEventID string
	Participants    string
}

type DriveRecord struct {
	FileID       string
	RevisionID   string
	ModifiedTime string
}

func BuildArtifacts(
	ctx context.Context,
	client api.Client,
	crawlerVersion string,
) ([]StoredRow, []DriveRecord, error) {
	docs, err := client.ListDocs(ctx)
	if err != nil {
		return nil, nil, err
	}
	events, err := client.ListCalendarEvents(ctx)
	if err != nil {
		return nil, nil, err
	}
	var rows []StoredRow
	var driveRows []DriveRecord
	for _, doc := range docs {
		fidelity, err := docclass.Classify(doc.Name)
		if err != nil {
			return nil, nil, err
		}
		text, err := client.ExportPlainText(ctx, doc.ID)
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
		artifact := source.Artifact{
			Identity: source.Identity{
				Kind:     source.KindGMeetGemini,
				SourceID: doc.ID,
			},
			SourceRevision: revision,
			Fidelity:       fidelity,
			Privacy:        source.PrivacyPrivate,
			NormalizedText: text,
			Window:         window,
			Language:       "und",
		}
		if err := artifact.Validate(); err != nil {
			return nil, nil, err
		}
		if _, err := source.ProvenanceForArtifact(artifact, crawlerVersion); err != nil {
			return nil, nil, err
		}
		rows = append(rows, StoredRow{
			Artifact:        artifact,
			CalendarEventID: calendarID,
			Participants:    participants,
		})
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
