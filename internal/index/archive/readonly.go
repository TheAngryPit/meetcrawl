package archive

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/source"
	ckstore "github.com/openclaw/crawlkit/store"
)

// ReadOnlyStore opens meetcrawl.db with crawlkit's read-only SQLite mode.
type ReadOnlyStore struct {
	inner *ckstore.Store
}

func OpenReadOnly(ctx context.Context, path string) (*ReadOnlyStore, error) {
	inner, err := ckstore.OpenReadOnly(ctx, path)
	if err != nil {
		return nil, err
	}
	var meetings int
	if err := inner.DB().QueryRowContext(ctx, `select count(*) from sqlite_master where type='table' and name='meetings'`).Scan(&meetings); err != nil {
		_ = inner.Close()
		return nil, err
	}
	if meetings == 0 {
		_ = inner.Close()
		return nil, fmt.Errorf("index database %s is missing meetings table (run meetcrawl index)", path)
	}
	return &ReadOnlyStore{inner: inner}, nil
}

func (s *ReadOnlyStore) Close() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Close()
}

type MeetingSummary struct {
	MeetingID    string `json:"meeting_id"`
	BestFidelity string `json:"best_fidelity"`
	PrivacyClass string `json:"privacy_class"`
}

type MeetingDetail struct {
	MeetingID     string              `json:"meeting_id"`
	BestFidelity  string              `json:"best_fidelity"`
	PrivacyClass  string              `json:"privacy_class"`
	ICalUID       string              `json:"ical_uid,omitempty"`
	EventStartUTC string              `json:"event_start_utc,omitempty"`
	Fidelities    []string            `json:"fidelities"`
	Contents      []MeetingContent    `json:"contents"`
}

type MeetingContent struct {
	ContentHash    string `json:"content_hash"`
	Fidelity       string `json:"fidelity"`
	Language       string `json:"language"`
	NormalizedText string `json:"normalized_text"`
}

func (s *ReadOnlyStore) Search(ctx context.Context, query string, limit int, allow RestrictedAllowlist) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 50
	}
	ftsQuery := ckstore.FTS5TokenQuery(query)
	if ftsQuery == "" {
		return nil, fmt.Errorf("empty search query")
	}
	filter, args := privacyFilterSQL(allow, "m")
	args = append([]any{ftsQuery}, args...)
	args = append(args, limit)
	rows, err := s.inner.DB().QueryContext(ctx, fmt.Sprintf(`
select c.meeting_id, snippet(meeting_contents_fts, 0, '', '', '…', 32)
from meeting_contents_fts
join meeting_contents c on c.rowid = meeting_contents_fts.rowid
join meetings m on m.meeting_id = c.meeting_id
where meeting_contents_fts match ?
  and %s
limit ?
`, filter), args...)
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

func (s *ReadOnlyStore) ListMeetings(ctx context.Context, limit int, allow RestrictedAllowlist) ([]MeetingSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	filter, args := privacyFilterSQL(allow, "m")
	args = append(args, limit)
	rows, err := s.inner.DB().QueryContext(ctx, fmt.Sprintf(`
select m.meeting_id, m.best_fidelity, m.privacy_class
from meetings m
where %s
order by m.meeting_id
limit ?
`, filter), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MeetingSummary
	for rows.Next() {
		var item MeetingSummary
		if err := rows.Scan(&item.MeetingID, &item.BestFidelity, &item.PrivacyClass); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *ReadOnlyStore) GetMeeting(ctx context.Context, meetingID string, allow RestrictedAllowlist) (MeetingDetail, error) {
	meetingID = strings.TrimSpace(meetingID)
	if meetingID == "" {
		return MeetingDetail{}, fmt.Errorf("meeting_id is required")
	}
	var detail MeetingDetail
	var ical, start sql.NullString
	err := s.inner.DB().QueryRowContext(ctx, `
select meeting_id, best_fidelity, privacy_class, ical_uid, event_start_utc
from meetings where meeting_id = ?
`, meetingID).Scan(&detail.MeetingID, &detail.BestFidelity, &detail.PrivacyClass, &ical, &start)
	if err == sql.ErrNoRows {
		return MeetingDetail{}, fmt.Errorf("meeting not found")
	}
	if err != nil {
		return MeetingDetail{}, err
	}
	if ical.Valid {
		detail.ICalUID = ical.String
	}
	if start.Valid {
		detail.EventStartUTC = start.String
	}
	if !allow.Visible(source.PrivacyClass(detail.PrivacyClass), detail.MeetingID) {
		return MeetingDetail{}, fmt.Errorf("meeting not found")
	}
	fRows, err := s.inner.DB().QueryContext(ctx, `
select fidelity from meeting_fidelities where meeting_id = ? order by fidelity
`, meetingID)
	if err != nil {
		return MeetingDetail{}, err
	}
	defer fRows.Close()
	for fRows.Next() {
		var fidelity string
		if err := fRows.Scan(&fidelity); err != nil {
			return MeetingDetail{}, err
		}
		detail.Fidelities = append(detail.Fidelities, fidelity)
	}
	if err := fRows.Err(); err != nil {
		return MeetingDetail{}, err
	}
	cRows, err := s.inner.DB().QueryContext(ctx, `
select content_hash, fidelity, language, normalized_text
from meeting_contents where meeting_id = ? order by fidelity, content_hash
`, meetingID)
	if err != nil {
		return MeetingDetail{}, err
	}
	defer cRows.Close()
	for cRows.Next() {
		var c MeetingContent
		if err := cRows.Scan(&c.ContentHash, &c.Fidelity, &c.Language, &c.NormalizedText); err != nil {
			return MeetingDetail{}, err
		}
		detail.Contents = append(detail.Contents, c)
	}
	return detail, cRows.Err()
}

// RestrictedAllowlist lists meeting_ids that may surface despite restricted class.
type RestrictedAllowlist map[string]struct{}

func NewRestrictedAllowlist(ids []string) RestrictedAllowlist {
	out := make(RestrictedAllowlist, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}

func (a RestrictedAllowlist) Visible(class source.PrivacyClass, meetingID string) bool {
	if class != source.PrivacyRestricted {
		return true
	}
	_, ok := a[meetingID]
	return ok
}

func privacyFilterSQL(allow RestrictedAllowlist, alias string) (string, []any) {
	col := alias + ".privacy_class"
	idCol := alias + ".meeting_id"
	if len(allow) == 0 {
		return col + ` != ?`, []any{string(source.PrivacyRestricted)}
	}
	placeholders := make([]string, 0, len(allow))
	args := []any{string(source.PrivacyRestricted)}
	for id := range allow {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return fmt.Sprintf(`(%s != ? or %s in (%s))`, col, idCol, strings.Join(placeholders, ",")), args
}
