package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var linkNextCursorPattern = regexp.MustCompile(`<[^>]*[?&]cursor=([^>&]+)[^>]*>;\s*rel="next"`)

type LiveClient struct {
	http    *http.Client
	token   string
	baseURL string
}

func NewLive(ctx context.Context, token string) (*LiveClient, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("circleback api key is empty")
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
			Summary:         summary,
			Start:           start,
			End:             end,
			Revision:        RevisionFromSummary(summary),
			CalendarEventID: CalendarEventIDFromSummary(summary),
		})
	}
	return out, nil
}

func (c *LiveClient) TranscriptItems(ctx context.Context, meetingID string) ([]TranscriptItem, error) {
	meetingID = strings.TrimSpace(meetingID)
	if meetingID == "" {
		return nil, fmt.Errorf("meeting id is required")
	}
	path := "/meeting/" + url.PathEscape(meetingID) + "/transcript"
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
		var page []MeetingSummary
		linkHeader, err := c.getJSONWithLink(ctx, path, &page)
		if err != nil {
			return nil, err
		}
		for _, meeting := range page {
			if err := ValidateMeetingSummary(meeting); err != nil {
				return nil, err
			}
			all = append(all, meeting)
		}
		next := parseNextCursor(linkHeader)
		if next == "" {
			break
		}
		cursor = next
	}
	return all, nil
}

// ParseNextCursor extracts the cursor query value from an RFC 8288 Link rel="next" header.
func ParseNextCursor(linkHeader string) string {
	return parseNextCursor(linkHeader)
}

func parseNextCursor(linkHeader string) string {
	linkHeader = strings.TrimSpace(linkHeader)
	if linkHeader == "" {
		return ""
	}
	for _, part := range strings.Split(linkHeader, ",") {
		part = strings.TrimSpace(part)
		m := linkNextCursorPattern.FindStringSubmatch(part)
		if len(m) == 2 {
			decoded, err := url.QueryUnescape(m[1])
			if err != nil {
				return strings.TrimSpace(m[1])
			}
			return strings.TrimSpace(decoded)
		}
	}
	return ""
}

func (c *LiveClient) getJSONWithLink(ctx context.Context, path string, out any) (string, error) {
	reqURL := strings.TrimRight(c.baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", &AuthError{Status: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("circleback %s: http %d", path, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return "", fmt.Errorf("circleback %s decode: %w", path, err)
	}
	return resp.Header.Get("Link"), nil
}

func (c *LiveClient) getRaw(ctx context.Context, path string) ([]byte, error) {
	reqURL := strings.TrimRight(c.baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
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
		return nil, fmt.Errorf("circleback %s: http %d", path, resp.StatusCode)
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
		return "circleback api auth failed"
	}
	return fmt.Sprintf("circleback api auth failed (http %d)", e.Status)
}
