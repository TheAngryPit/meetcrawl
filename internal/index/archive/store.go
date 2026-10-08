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

type MeetingRecord struct {
	MeetingID     string
	BestFidelity  source.Fidelity
	Privacy       source.PrivacyClass
	ICalUID       string
	EventStartUTC string
	Fidelities    []source.Fidelity
}

type ContentRecord struct {
	MeetingID      string
	ContentHash    string
	NormalizedText string
	Fidelity       source.Fidelity
	Language       string
	Sources        []SourceLink
}

type SourceLink struct {
	Source         string
	SourceID       string
	SourceRevision string
	ProvenanceHash string
	Fidelity       source.Fidelity
}

type Status struct {
	Meetings  int
	Contents  int
	Sources   int
	LastIndex time.Time
	DBPath    string
}

type SearchHit struct {
	MeetingID string `json:"meeting_id"`
	Snippet   string `json:"snippet"`
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

func (s *Store) ReplaceIndex(ctx context.Context, meetings []MeetingRecord, contents []ContentRecord, finishedAt time.Time) error {
	return s.inner.WithTx(ctx, func(tx *sql.Tx) error {
		for _, table := range []string{"content_sources", "meeting_fidelities", "meeting_contents", "meetings"} {
			if _, err := tx.ExecContext(ctx, "delete from "+table); err != nil {
				return err
			}
		}
		meetingStmt, err := tx.PrepareContext(ctx, `
insert into meetings(meeting_id, best_fidelity, privacy_class, ical_uid, event_start_utc)
values (?, ?, ?, ?, ?)
`)
		if err != nil {
			return err
		}
		defer meetingStmt.Close()
		fidelityStmt, err := tx.PrepareContext(ctx, `
insert into meeting_fidelities(meeting_id, fidelity) values (?, ?)
`)
		if err != nil {
			return err
		}
		defer fidelityStmt.Close()
		contentStmt, err := tx.PrepareContext(ctx, `
insert into meeting_contents(meeting_id, content_hash, normalized_text, fidelity, language)
values (?, ?, ?, ?, ?)
`)
		if err != nil {
			return err
		}
		defer contentStmt.Close()
		sourceStmt, err := tx.PrepareContext(ctx, `
insert into content_sources(
  meeting_id, content_hash, source, source_id, source_revision, provenance_hash, fidelity
) values (?, ?, ?, ?, ?, ?, ?)
`)
		if err != nil {
			return err
		}
		defer sourceStmt.Close()

		for _, meeting := range meetings {
			if _, err := meetingStmt.ExecContext(ctx,
				meeting.MeetingID,
				meeting.BestFidelity.String(),
				string(meeting.Privacy),
				nullString(meeting.ICalUID),
				nullString(meeting.EventStartUTC),
			); err != nil {
				return err
			}
			for _, fidelity := range meeting.Fidelities {
				if _, err := fidelityStmt.ExecContext(ctx, meeting.MeetingID, fidelity.String()); err != nil {
					return err
				}
			}
		}
		sourceLinks := 0
		for _, content := range contents {
			if _, err := contentStmt.ExecContext(ctx,
				content.MeetingID,
				content.ContentHash,
				content.NormalizedText,
				content.Fidelity.String(),
				content.Language,
			); err != nil {
				return err
			}
			for _, link := range content.Sources {
				if _, err := sourceStmt.ExecContext(ctx,
					content.MeetingID,
					content.ContentHash,
					link.Source,
					link.SourceID,
					link.SourceRevision,
					link.ProvenanceHash,
					link.Fidelity.String(),
				); err != nil {
					return err
				}
				sourceLinks++
			}
		}
		_, err = tx.ExecContext(ctx, `
insert into index_runs(meeting_count, content_count, source_link_count, finished_at)
values (?, ?, ?, ?)
`, len(meetings), len(contents), sourceLinks, finishedAt.UTC().Format(time.RFC3339Nano))
		return err
	})
}

func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Store) Status(ctx context.Context) (Status, error) {
	st := Status{DBPath: s.Path()}
	if err := s.inner.DB().QueryRowContext(ctx, `select count(*) from meetings`).Scan(&st.Meetings); err != nil {
		return Status{}, err
	}
	if err := s.inner.DB().QueryRowContext(ctx, `select count(*) from meeting_contents`).Scan(&st.Contents); err != nil {
		return Status{}, err
	}
	if err := s.inner.DB().QueryRowContext(ctx, `select count(*) from content_sources`).Scan(&st.Sources); err != nil {
		return Status{}, err
	}
	var last sql.NullString
	if err := s.inner.DB().QueryRowContext(ctx, `select max(finished_at) from index_runs`).Scan(&last); err != nil {
		return Status{}, err
	}
	if last.Valid {
		if ts, err := time.Parse(time.RFC3339Nano, last.String); err == nil {
			st.LastIndex = ts
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
select c.meeting_id, snippet(meeting_contents_fts, 0, '', '', '…', 32)
from meeting_contents_fts
join meeting_contents c on c.rowid = meeting_contents_fts.rowid
where meeting_contents_fts match ?
limit ?
`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		var hit SearchHit
		if err := rows.Scan(&hit.MeetingID, &hit.Snippet); err != nil {
			return nil, err
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

func (s *Store) OrderedRowDump(ctx context.Context) (string, error) {
	tables := []string{
		"meetings",
		"meeting_fidelities",
		"meeting_contents",
		"content_sources",
	}
	var dump string
	for _, table := range tables {
		cols, err := tableColumns(ctx, s.inner.DB(), table)
		if err != nil {
			return "", err
		}
		order := stringsJoin(cols, ",")
		query := fmt.Sprintf("select %s from %s order by %s", order, table, order)
		rows, err := s.inner.DB().QueryContext(ctx, query)
		if err != nil {
			return "", err
		}
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return "", err
			}
			dump += table + "|"
			for i, v := range values {
				if i > 0 {
					dump += "\t"
				}
				dump += fmt.Sprint(v)
			}
			dump += "\n"
		}
		if err := rows.Close(); err != nil {
			return "", err
		}
		if err := rows.Err(); err != nil {
			return "", err
		}
	}
	return dump, nil
}

func tableColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("pragma table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}
