package archive

import (
	"context"
	"database/sql"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/source"
	"github.com/openclaw/crawlkit/state"
	ckstore "github.com/openclaw/crawlkit/store"
)

type Store struct {
	inner *ckstore.Store
}

type StoredArtifact struct {
	Artifact        source.Artifact
	CalendarEventID string
	Participants    string
}

type Status struct {
	Artifacts int
	SyncRuns  int
	LastSync  time.Time
	DBPath    string
}

func Open(ctx context.Context, path string) (*Store, error) {
	inner, err := ckstore.Open(ctx, ckstore.Options{
		Path:          path,
		Schema:        Schema,
		SchemaVersion: SchemaVersion,
		MaxOpenConns:  1,
		MaxIdleConns:  1,
	})
	if err != nil {
		return nil, err
	}
	if err := state.EnsureSchema(ctx, inner.DB()); err != nil {
		_ = inner.Close()
		return nil, err
	}
	return &Store{inner: inner}, nil
}

func (s *Store) Close() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Close()
}

func (s *Store) Path() string {
	if s == nil || s.inner == nil {
		return ""
	}
	return s.inner.Path()
}

func (s *Store) DB() *sql.DB {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.DB()
}

func (s *Store) ReplaceArtifacts(ctx context.Context, artifacts []StoredArtifact, crawlerVersion string, finishedAt time.Time) error {
	return s.inner.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `delete from artifacts`); err != nil {
			return err
		}
		stmt, err := tx.PrepareContext(ctx, `
insert into artifacts(
  source_id, fidelity, source_revision, privacy_class, normalized_text,
  content_hash, provenance_hash, window_start, window_end, language,
  calendar_event_id, participants, updated_at
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		updated := finishedAt.UTC().Format(time.RFC3339Nano)
		for _, stored := range artifacts {
			artifact := stored.Artifact
			if err := artifact.Validate(); err != nil {
				return err
			}
			prov, err := source.ProvenanceForArtifact(artifact, crawlerVersion)
			if err != nil {
				return err
			}
			windowStart := ""
			if !artifact.Window.Start.IsZero() {
				windowStart = artifact.Window.Start.UTC().Format(time.RFC3339Nano)
			}
			windowEnd := ""
			if !artifact.Window.End.IsZero() {
				windowEnd = artifact.Window.End.UTC().Format(time.RFC3339Nano)
			}
			if _, err := stmt.ExecContext(ctx,
				artifact.SourceID,
				artifact.Fidelity.String(),
				artifact.SourceRevision,
				string(artifact.Privacy),
				artifact.NormalizedText,
				source.ContentHash(artifact.NormalizedText),
				prov,
				windowStart,
				windowEnd,
				artifact.Language,
				nullString(stored.CalendarEventID),
				nullString(stored.Participants),
				updated,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Store) RecordSyncRun(ctx context.Context, code, detail string, artifactCount int, finishedAt time.Time) error {
	_, err := s.inner.DB().ExecContext(ctx, `
insert into sync_runs(code, detail, artifact_count, finished_at)
values (?, ?, ?, ?)
`, code, detail, artifactCount, finishedAt.UTC().Format(time.RFC3339Nano))
	return err
}
