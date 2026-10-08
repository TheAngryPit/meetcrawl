package reads

import (
	"context"
	"fmt"
	"time"

	ckstore "github.com/openclaw/crawlkit/store"
)

const SurfaceMCP = "mcp"

// Entry is one append-only read_log row.
type Entry struct {
	Surface    string
	ClientName string
	Tool       string
	MeetingID  string
	ArtifactID string
	QueryHash  string
}

// LogStore appends read_log rows to reads.db.
type LogStore struct {
	inner *ckstore.Store
}

func OpenLog(ctx context.Context, path string) (*LogStore, error) {
	if err := EnsureSchema(ctx, path); err != nil {
		return nil, err
	}
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
	return &LogStore{inner: inner}, nil
}

func (l *LogStore) Close() error {
	if l == nil || l.inner == nil {
		return nil
	}
	return l.inner.Close()
}

func (l *LogStore) Append(ctx context.Context, e Entry) error {
	if l == nil || l.inner == nil {
		return fmt.Errorf("reads: closed log store")
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := l.inner.DB().ExecContext(ctx, `
insert into read_log(ts, surface, client_name, tool, meeting_id, artifact_id, query_hash)
values (?, ?, ?, ?, ?, ?, ?)
`, ts, e.Surface, nullString(e.ClientName), e.Tool, nullString(e.MeetingID), nullString(e.ArtifactID), e.QueryHash)
	return err
}

func CountRows(ctx context.Context, path string) (int, error) {
	store, err := ckstore.OpenReadOnly(ctx, path)
	if err != nil {
		return 0, err
	}
	defer store.Close()
	var count int
	err = store.DB().QueryRowContext(ctx, `select count(*) from read_log`).Scan(&count)
	return count, err
}

func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
