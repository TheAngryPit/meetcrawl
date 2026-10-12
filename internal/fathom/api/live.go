package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
		return nil, fmt.Errorf("fathom api key is empty")
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
			return nil, fmt.Errorf("meeting %d window: %w", summary.RecordingID, err)
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

func (c *LiveClient) TranscriptItems(ctx context.Context, recordingID string) ([]TranscriptItem, error) {
	recordingID = strings.TrimSpace(recordingID)
	if recordingID == "" {
		return nil, fmt.Errorf("recording id is required")
	}
	path := "/recordings/" + url.PathEscape(recordingID) + "/transcript"
	var resp TranscriptResponse
	if err := c.getJSON(ctx, path, &resp); err != nil {
		return nil, err
	}
	if len(resp.Transcript) == 0 {
		return nil, fmt.Errorf("recording %q has empty transcript", recordingID)
	}
	out := make([]TranscriptItem, 0, len(resp.Transcript))
	for i, item := range resp.Transcript {
		raw, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		parsed, err := parseTranscriptItem(raw)
		if err != nil {
			return nil, fmt.Errorf("transcript[%d]: %w", i, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func (c *LiveClient) listMeetings(ctx context.Context) ([]MeetingSummary, error) {
	var all []MeetingSummary
	cursor := ""
	for {
		q := url.Values{}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		path := "/meetings"
		if enc := q.Encode(); enc != "" {
			path += "?" + enc
		}
		var resp MeetingListResponse
		if err := c.getJSON(ctx, path, &resp); err != nil {
			return nil, err
		}
		for _, meeting := range resp.Items {
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
	req.Header.Set("X-Api-Key", c.token)
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
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &AuthError{Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fathom %s: http %d", path, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("fathom %s decode: %w", path, err)
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
		return "fathom api auth failed"
	}
	return fmt.Sprintf("fathom api auth failed (http %d)", e.Status)
}

func ParseRecordingID(recordingID string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(recordingID), 10, 64)
}
