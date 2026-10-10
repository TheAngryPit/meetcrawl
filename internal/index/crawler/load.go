package crawler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/source"
	ckstore "github.com/openclaw/crawlkit/store"
)

// LoadArtifacts opens path read-only and returns artifact rows for kind.
func LoadArtifacts(ctx context.Context, path string, kind source.Kind) ([]Row, error) {
	if !kind.Valid() {
		return nil, fmt.Errorf("index: empty source kind")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("index: empty archive path for %s", kind)
	}
	store, err := ckstore.OpenReadOnly(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("open %s archive: %w", kind, err)
	}
	defer store.Close()
	rows, err := store.DB().QueryContext(ctx, `
select
  source_id,
  fidelity,
  source_revision,
  privacy_class,
  normalized_text,
  content_hash,
  provenance_hash,
  coalesce(window_start, ''),
  coalesce(window_end, ''),
  language,
  coalesce(calendar_event_id, ''),
  coalesce(participants, '')
from artifacts
order by source_id
`)
	if err != nil {
		return nil, fmt.Errorf("query %s artifacts: %w", kind, err)
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		row, err := scanRow(rows, kind)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func scanRow(rows *sql.Rows, kind source.Kind) (Row, error) {
	var row Row
	var fidelity, privacy string
	var windowStart, windowEnd string
	if err := rows.Scan(
		&row.SourceID,
		&fidelity,
		&row.SourceRevision,
		&privacy,
		&row.NormalizedText,
		&row.ContentHash,
		&row.ProvenanceHash,
		&windowStart,
		&windowEnd,
		&row.Language,
		&row.CalendarICal,
		&row.Participants,
	); err != nil {
		return Row{}, err
	}
	row.Source = kind
	f, err := parseFidelity(fidelity)
	if err != nil {
		return Row{}, err
	}
	row.Fidelity = f
	switch source.PrivacyClass(privacy) {
	case source.PrivacyPrivate, source.PrivacyRestricted, source.PrivacyShareable:
		row.Privacy = source.PrivacyClass(privacy)
	default:
		return Row{}, fmt.Errorf("index: unknown privacy_class %q", privacy)
	}
	row.WindowStart = parseTime(windowStart)
	row.WindowEnd = parseTime(windowEnd)
	return row, nil
}

func parseFidelity(raw string) (source.Fidelity, error) {
	switch strings.TrimSpace(raw) {
	case "transcript":
		return source.FidelityTranscript, nil
	case "notes":
		return source.FidelityNotes, nil
	case "summary":
		return source.FidelitySummary, nil
	default:
		return 0, fmt.Errorf("index: unknown fidelity %q", raw)
	}
}

func parseTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC()
		}
	}
	return time.Time{}
}
