package archive

import (
	"context"
	"database/sql"
	"fmt"
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
	IngestFlags     string
}

type Status struct {
	Artifacts int
	SyncRuns  int
	LastSync  time.Time
	DBPath    string
}

type SearchHit struct {
	SourceID       string `json:"source_id"`
	Fidelity       string `json:"fidelity"`
	Snippet        string `json:"snippet"`
	SourceRevision string `json:"source_revision,omitempty"`
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
  calendar_event_id, participants, ingest_flags, updated_at
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
				stored.IngestFlags,
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

type DriveFileRecord struct {
	FileID       string
	RevisionID   string
	ModifiedTime string
}

func (s *Store) ReplaceDriveFiles(ctx context.Context, files []DriveFileRecord, finishedAt time.Time) error {
	return s.inner.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `delete from drive_files`); err != nil {
			return err
		}
		if len(files) == 0 {
			return nil
		}
		stmt, err := tx.PrepareContext(ctx, `
insert into drive_files(file_id, revision_id, modified_time, updated_at)
values (?, ?, ?, ?)
`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		updated := finishedAt.UTC().Format(time.RFC3339Nano)
		for _, f := range files {
			if _, err := stmt.ExecContext(ctx, f.FileID, f.RevisionID, f.ModifiedTime, updated); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) RecordSyncRun(ctx context.Context, code, detail string, artifactCount int, finishedAt time.Time) error {
	_, err := s.inner.DB().ExecContext(ctx, `
insert into sync_runs(code, detail, artifact_count, finished_at)
values (?, ?, ?, ?)
`, code, detail, artifactCount, finishedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ArtifactCount(ctx context.Context) (int, error) {
	var count int
	err := s.inner.DB().QueryRowContext(ctx, `select count(*) from artifacts`).Scan(&count)
	return count, err
}

func (s *Store) Status(ctx context.Context) (Status, error) {
	st := Status{DBPath: s.Path()}
	if err := s.inner.DB().QueryRowContext(ctx, `select count(*) from artifacts`).Scan(&st.Artifacts); err != nil {
		return Status{}, err
	}
	if err := s.inner.DB().QueryRowContext(ctx, `select count(*) from sync_runs`).Scan(&st.SyncRuns); err != nil {
		return Status{}, err
	}
	var last sql.NullString
	if err := s.inner.DB().QueryRowContext(ctx, `select max(finished_at) from sync_runs`).Scan(&last); err != nil {
		return Status{}, err
	}
	if last.Valid {
		if ts, err := time.Parse(time.RFC3339Nano, last.String); err == nil {
			st.LastSync = ts
		}
	}
	return st, nil
}

func (s *Store) Search(ctx context.Context, query string, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 50
	}
	ftsQuery := ckstore.FTS5TokenQuery(query)
	if ftsQuery == "" {
		return nil, fmt.Errorf("empty search query")
	}
	rows, err := s.inner.DB().QueryContext(ctx, `
select a.source_id, a.fidelity, snippet(artifacts_fts, 0, '', '', '…', 32), a.source_revision
from artifacts_fts
join artifacts a on a.rowid = artifacts_fts.rowid
where artifacts_fts match ?
limit ?
`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		var hit SearchHit
		if err := rows.Scan(&hit.SourceID, &hit.Fidelity, &hit.Snippet, &hit.SourceRevision); err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

func (s *Store) QueryReadOnly(ctx context.Context, query string) (ckstore.QueryResult, error) {
	query = trimSQL(query)
	if !IsReadOnlySQL(query) {
		return ckstore.QueryResult{}, fmt.Errorf("only read-only sql is allowed")
	}
	return s.inner.Query(ctx, query)
}
