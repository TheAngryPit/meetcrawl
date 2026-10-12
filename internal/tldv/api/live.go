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

const apiPrefix = "/v1alpha1"

type LiveClient struct {
	http    *http.Client
	token   string
	baseURL string
}

func NewLive(ctx context.Context, token string) (*LiveClient, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("tldv api key is empty")
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
	path := apiPrefix + "/meetings/" + url.PathEscape(meetingID) + "/transcript"
	raw, err := c.getRaw(ctx, path)
	if err != nil {
		return nil, err
	}
	items, err := ParseTranscriptItems(raw)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("meeting %q has empty transcript", meetingID)
	}
	return items, nil
}

func (c *LiveClient) listMeetings(ctx context.Context) ([]MeetingSummary, error) {
	var all []MeetingSummary
	page := 1
	const limit = 100
	for {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(limit))
		path := apiPrefix + "/meetings?" + q.Encode()
		var resp MeetingListResponse
		if err := c.getJSON(ctx, path, &resp); err != nil {
			return nil, err
		}
		for _, meeting := range resp.Results {
			if err := ValidateMeetingSummary(meeting); err != nil {
				return nil, err
			}
			all = append(all, meeting)
		}
		if resp.Pages == 0 || page >= resp.Pages {
			break
		}
		page++
	}
	return all, nil
}

func (c *LiveClient) getJSON(ctx context.Context, path string, out any) error {
	raw, err := c.getRaw(ctx, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("tldv %s decode: %w", path, err)
	}
	return nil
}

func (c *LiveClient) getRaw(ctx context.Context, path string) ([]byte, error) {
	reqURL := strings.TrimRight(c.baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &AuthError{Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tldv %s: http %d", path, resp.StatusCode)
	}
	return raw, nil
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
		return "tldv api auth failed"
	}
	return fmt.Sprintf("tldv api auth failed (http %d)", e.Status)
}
