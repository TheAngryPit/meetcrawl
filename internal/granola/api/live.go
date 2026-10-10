package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LiveClient struct {
	http    *http.Client
	token   string
	baseURL string
}

func NewLive(ctx context.Context, token string) (*LiveClient, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("granola api key is empty")
	}
	_ = ctx
	return &LiveClient{
		http:    &http.Client{Timeout: 60 * time.Second},
		token:   token,
		baseURL: BaseURL,
	}, nil
}

func (c *LiveClient) ListNotesForSync(ctx context.Context) ([]NoteForSync, error) {
	summaries, err := c.listNotes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]NoteForSync, 0, len(summaries))
	for _, summary := range summaries {
		detail, err := c.getNote(ctx, summary.ID)
		if err != nil {
			return nil, err
		}
		start, end, err := WindowFromDetail(detail)
		if err != nil {
			return nil, fmt.Errorf("note %q window: %w", summary.ID, err)
		}
		out = append(out, NoteForSync{
			Note:            summary,
			CalendarEventID: CalendarEventIDFromDetail(detail),
			Start:           start,
			End:             end,
		})
	}
	return out, nil
}

func (c *LiveClient) listNotes(ctx context.Context) ([]NoteSummary, error) {
	var all []NoteSummary
	cursor := ""
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		q.Set("page_size", "30")
		path := "/v1/notes"
		if enc := q.Encode(); enc != "" {
			path += "?" + enc
		}
		var resp ListNotesResponse
		if err := c.getJSON(ctx, path, &resp); err != nil {
			return nil, err
		}
		for _, note := range resp.Notes {
			if err := ValidateNoteSummary(note); err != nil {
				return nil, err
			}
			all = append(all, note)
		}
		if !resp.HasMore || resp.Cursor == nil || strings.TrimSpace(*resp.Cursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*resp.Cursor)
	}
	return all, nil
}

func (c *LiveClient) getNote(ctx context.Context, noteID string) (NoteDetail, error) {
	noteID = strings.TrimSpace(noteID)
	var detail NoteDetail
	if err := c.getJSON(ctx, "/v1/notes/"+url.PathEscape(noteID), &detail); err != nil {
		return NoteDetail{}, err
	}
	if strings.TrimSpace(detail.ID) == "" {
		detail.ID = noteID
	}
	return detail, nil
}

func (c *LiveClient) TranscriptItems(ctx context.Context, noteID string) ([]TranscriptItem, error) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return nil, fmt.Errorf("note id is required")
	}
	var all []TranscriptItem
	cursor := ""
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		q.Set("page_size", "100")
		path := fmt.Sprintf("/v1/notes/%s/transcript", url.PathEscape(noteID))
		if enc := q.Encode(); enc != "" {
			path += "?" + enc
		}
		var page TranscriptPage
		if err := c.getJSON(ctx, path, &page); err != nil {
			return nil, err
		}
		for i, item := range page.Transcript {
			raw, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			parsed, err := parseTranscriptItem(raw)
			if err != nil {
				return nil, fmt.Errorf("transcript item[%d]: %w", len(all)+i, err)
			}
			all = append(all, parsed)
		}
		if !page.HasMore || page.Cursor == nil || strings.TrimSpace(*page.Cursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*page.Cursor)
	}
	return all, nil
}

func (c *LiveClient) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.applyHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &AuthError{Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("granola %s: http %d", path, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("granola %s decode: %w", path, err)
	}
	return nil
}

func (c *LiveClient) applyHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
}

type AuthError struct {
	Status int
}

func AsAuthError(err error, target **AuthError) bool {
	if err == nil {
		return false
	}
	auth, ok := err.(*AuthError)
	if !ok {
		return false
	}
	*target = auth
	return true
}

func (e *AuthError) Error() string {
	if e == nil {
		return "granola api auth failed"
	}
	return fmt.Sprintf("granola api auth failed (http %d)", e.Status)
}
