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
		return nil, fmt.Errorf("krisp api key is empty")
	}
	_ = ctx
	return &LiveClient{
		http:    &http.Client{Timeout: 60 * time.Second},
		token:   token,
		baseURL: BaseURL,
	}, nil
}

func (c *LiveClient) ListMeetingsForSync(ctx context.Context) ([]MeetingForSync, error) {
	summaries, err := c.listMeetings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MeetingForSync, 0, len(summaries))
	for _, summary := range summaries {
		start, end, err := WindowFromSummary(summary)
		if err != nil {
			return nil, fmt.Errorf("meeting %s window: %w", summary.ID, err)
		}
		out = append(out, MeetingForSync{
			Summary:  summary,
			Start:    start,
			End:      end,
			Revision: RevisionFromSummary(summary),
		})
	}
	return out, nil
}

func (c *LiveClient) TranscriptItems(ctx context.Context, meetingID string) ([]TranscriptItem, error) {
	meetingID = strings.TrimSpace(meetingID)
	if meetingID == "" {
		return nil, fmt.Errorf("meeting id is required")
	}
	path := "/meetings/" + url.PathEscape(meetingID) + "?fields=transcript"
	var detail MeetingDetail
	if err := c.getJSON(ctx, path, &detail); err != nil {
		return nil, err
	}
	if detail.Transcript == nil || len(detail.Transcript.Segments) == 0 {
		return nil, fmt.Errorf("meeting %q has empty transcript", meetingID)
	}
	raw, err := json.Marshal(detail.Transcript)
	if err != nil {
		return nil, err
	}
	return ParseTranscriptDocument(raw)
}

func (c *LiveClient) listMeetings(ctx context.Context) ([]MeetingSummary, error) {
	var all []MeetingSummary
	cursor := ""
	for {
		q := url.Values{}
		q.Set("limit", "100")
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		path := "/meetings?" + q.Encode()
		var resp MeetingListResponse
		if err := c.getJSON(ctx, path, &resp); err != nil {
			return nil, err
		}
		for _, meeting := range resp.Meetings {
			if err := ValidateMeetingSummary(meeting); err != nil {
				return nil, err
			}
			all = append(all, meeting)
		}
		if resp.NextCursor == nil || strings.TrimSpace(*resp.NextCursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*resp.NextCursor)
	}
	return all, nil
}

func (c *LiveClient) getJSON(ctx context.Context, path string, out any) error {
	reqURL := strings.TrimRight(c.baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return &AuthError{Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("krisp %s: http %d", path, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("krisp %s decode: %w", path, err)
	}
	return nil
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
		return "krisp api auth failed"
	}
	return fmt.Sprintf("krisp api auth failed (http %d)", e.Status)
}
