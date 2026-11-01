package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/source"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/archive"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/schema"
	wsource "github.com/TheAngryPit/meetcrawl/internal/whisp/source"
	"github.com/openclaw/crawlkit/cache"
	ckstore "github.com/openclaw/crawlkit/store"
)

type Options struct {
	SourceDB       string
	CrawlerVersion string
}

type Result struct {
	Code      source.OutcomeCode `json:"code"`
	Detail    string             `json:"detail,omitempty"`
	Artifacts int                `json:"artifacts"`
	SourceDB  string             `json:"source_db"`
	Outcome   source.SyncOutcome `json:"-"`
}

type UnsupportedSchemaError struct {
	Detail string
}

func (e *UnsupportedSchemaError) Error() string {
	if e == nil || e.Detail == "" {
		return string(source.OutcomeUnsupportedSchema)
	}
	return e.Detail
}

func Run(ctx context.Context, cfg wconfig.Config, opts Options) (Result, error) {
	sourceDB := opts.SourceDB
	if sourceDB == "" {
		sourceDB = cfg.SourceDB
	}
	sourceDB = filepath.Clean(sourceDB)
	if sourceDB == "" {
		return Result{}, fmt.Errorf("source db path is required (--source-db or config source_db)")
	}
	if _, err := os.Stat(sourceDB); err != nil {
		return Result{}, fmt.Errorf("stat source db: %w", err)
	}

	finishedAt := time.Now().UTC()
	snapshotDir := filepath.Join(cfg.CacheDir, "source-snapshots")
	snapshot, err := cache.SnapshotSQLite(cache.SQLiteSnapshotOptions{
		SourcePath:     sourceDB,
		DestinationDir: snapshotDir,
		Name:           "transcriptions.db",
	})
	if err != nil {
		return Result{}, fmt.Errorf("snapshot openwhispr db: %w", err)
	}
	// The snapshot is a full copy of the source, including data meetcrawl
	// never reads (voice embeddings, share tokens). Remove it once read.
	defer removeSnapshot(snapshot.Path)

	readStore, err := ckstore.OpenReadOnly(ctx, snapshot.Path)
	if err != nil {
		return Result{}, fmt.Errorf("open snapshot read-only: %w", err)
	}
	defer readStore.Close()

	if err := schema.ValidateNotesTable(ctx, readStore.DB()); err != nil {
		return Result{
			Code:     source.OutcomeUnsupportedSchema,
			Detail:   err.Error(),
			SourceDB: sourceDB,
		}, &UnsupportedSchemaError{Detail: err.Error()}
	}

	notes, err := wsource.ReadMeetingNotes(ctx, readStore.DB())
	if err != nil {
		return Result{}, err
	}
	version := opts.CrawlerVersion
	if version == "" {
		version = "whispcrawl-dev"
	}
	var artifacts []source.Artifact
	var stored []archive.StoredArtifact
	for _, note := range notes {
		noteArtifacts, err := wsource.ArtifactsForNote(note, version)
		if err != nil {
			return Result{}, err
		}
		for _, artifact := range noteArtifacts {
			artifacts = append(artifacts, artifact)
			stored = append(stored, archive.StoredArtifact{
				Artifact:        artifact,
				CalendarEventID: note.CalendarEventID,
				Participants:    note.Participants,
			})
		}
	}
	if len(artifacts) == 0 {
		return Result{}, fmt.Errorf("no meeting artifacts found in source (note_type=meeting)")
	}
	outcome := source.SyncOutcome{Code: source.OutcomeOK, Artifacts: artifacts}
	if err := outcome.Validate(); err != nil {
		return Result{}, err
	}

	archiveStore, err := archive.Open(ctx, cfg.DBPath)
	if err != nil {
		return Result{}, err
	}
	defer archiveStore.Close()
	if err := archiveStore.ReplaceArtifacts(ctx, stored, version, finishedAt); err != nil {
		return Result{}, err
	}
	if err := archiveStore.RecordSyncRun(ctx, string(source.OutcomeOK), "", len(artifacts), finishedAt); err != nil {
		return Result{}, err
	}

	return Result{
		Code:      source.OutcomeOK,
		Artifacts: len(artifacts),
		SourceDB:  sourceDB,
		Outcome:   outcome,
	}, nil
}

func removeSnapshot(path string) {
	for _, p := range []string{path, path + "-wal", path + "-shm"} {
		_ = os.Remove(p)
	}
}

func IsUnsupportedSchema(err error) bool {
	var target *UnsupportedSchemaError
	return errors.As(err, &target)
}
