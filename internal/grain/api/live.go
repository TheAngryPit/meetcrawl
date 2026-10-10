package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		return nil, fmt.Errorf("grain api token is empty")
	}
	_ = ctx
	return &LiveClient{
		http:    &http.Client{Timeout: 60 * time.Second},
		token:   token,
		baseURL: BaseURL,
	}, nil
}

func (c *LiveClient) ListRecordings(ctx context.Context) ([]RecordingSummary, error) {
	var all []RecordingSummary
	cursor := ""
	for {
		body := map[string]any{
			"include": map[string]bool{
				"calendar_event": true,
			},
		}
		if cursor != "" {
			body["cursor"] = cursor
		}
		var resp ListRecordingsResponse
		if err := c.postJSON(ctx, "/recordings", body, &resp); err != nil {
			return nil, err
		}
		for _, rec := range resp.Recordings {
			if err := ValidateRecording(rec); err != nil {
				return nil, err
			}
			start, err := parseRFC3339(rec.StartDatetime)
			if err != nil {
				return nil, fmt.Errorf("recording %q start_datetime: %w", rec.ID, err)
			}
			var end time.Time
			if rec.EndDatetime != "" {
				end, err = parseRFC3339(rec.EndDatetime)
				if err != nil {
					return nil, fmt.Errorf("recording %q end_datetime: %w", rec.ID, err)
				}
			}
			all = append(all, RecordingSummary{Recording: rec, Start: start, End: end})
		}
		if resp.Cursor == nil || strings.TrimSpace(*resp.Cursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*resp.Cursor)
	}
	return all, nil
}

func (c *LiveClient) TranscriptJSON(ctx context.Context, recordingID string) ([]TranscriptSegment, error) {
	recordingID = strings.TrimSpace(recordingID)
	if recordingID == "" {
		return nil, fmt.Errorf("recording id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/recordings/"+recordingID+"/transcript", nil)
	if err != nil {
		return nil, err
	}
	c.applyHeaders(req)
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
		return nil, fmt.Errorf("grain transcript %s: http %d", recordingID, resp.StatusCode)
	}
	return ParseTranscriptSegments(raw)
}

func (c *LiveClient) postJSON(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.applyHeaders(req)
	req.Header.Set("Content-Type", "application/json")
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
		return fmt.Errorf("grain %s: http %d", path, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("grain %s decode: %w", path, err)
	}
	return nil
}

func (c *LiveClient) applyHeaders(req *http.Request) {
	req.Header.Set(HeaderAPIVersion, APIVersion)
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
		return "grain api auth failed"
	}
	return fmt.Sprintf("grain api auth failed (http %d)", e.Status)
}

func parseRFC3339(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q", raw)
}
