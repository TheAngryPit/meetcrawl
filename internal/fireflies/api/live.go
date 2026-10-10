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

const (
	listTranscriptsQuery = `
query ListTranscripts($limit: Int, $skip: Int) {
  transcripts(limit: $limit, skip: $skip) {
    id
    title
    date
    dateString
    calendar_id
  }
}`

	transcriptDetailQuery = `
query TranscriptDetail($transcriptId: String!) {
  transcript(id: $transcriptId) {
    id
    title
    date
    dateString
    calendar_id
    duration
    sentences {
      index
      speaker_name
      speaker_id
      text
      start_time
      end_time
    }
  }
}`
)

type LiveClient struct {
	http  *http.Client
	token string
	url   string
}

func NewLive(ctx context.Context, token string) (*LiveClient, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("fireflies api key is empty")
	}
	_ = ctx
	return &LiveClient{
		http:  &http.Client{Timeout: 60 * time.Second},
		token: token,
		url:   GraphQLURL,
	}, nil
}

func (c *LiveClient) ListTranscriptsForSync(ctx context.Context) ([]TranscriptForSync, error) {
	summaries, err := c.listTranscripts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TranscriptForSync, 0, len(summaries))
	for _, summary := range summaries {
		detail := TranscriptDetail{
			ID:         summary.ID,
			Title:      summary.Title,
			Date:       summary.Date,
			DateString: summary.DateString,
			CalendarID: summary.CalendarID,
		}
		start, end, err := WindowFromDetail(detail, nil)
		if err != nil {
			return nil, fmt.Errorf("transcript %q window: %w", summary.ID, err)
		}
		out = append(out, TranscriptForSync{
			Summary:    summary,
			CalendarID: strings.TrimSpace(summary.CalendarID),
			Start:      start,
			End:        end,
			Revision:   RevisionFromDetail(detail),
		})
	}
	return out, nil
}

func (c *LiveClient) Sentences(ctx context.Context, transcriptID string) ([]Sentence, error) {
	detail, err := c.getTranscript(ctx, transcriptID)
	if err != nil {
		return nil, err
	}
	if len(detail.Sentences) == 0 {
		return nil, fmt.Errorf("transcript %q has empty sentences", transcriptID)
	}
	out := make([]Sentence, 0, len(detail.Sentences))
	for i, sentence := range detail.Sentences {
		raw, err := json.Marshal(sentence)
		if err != nil {
			return nil, err
		}
		parsed, err := parseSentence(raw)
		if err != nil {
			return nil, fmt.Errorf("sentence[%d]: %w", i, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func (c *LiveClient) listTranscripts(ctx context.Context) ([]TranscriptSummary, error) {
	var all []TranscriptSummary
	skip := 0
	for {
		var payload struct {
			Transcripts []TranscriptSummary `json:"transcripts"`
		}
		if err := c.postGraphQL(ctx, listTranscriptsQuery, map[string]any{
			"limit": listPageSize,
			"skip":  skip,
		}, &payload); err != nil {
			return nil, err
		}
		batch := payload.Transcripts
		if len(batch) == 0 {
			break
		}
		for _, summary := range batch {
			if err := ValidateTranscriptSummary(summary); err != nil {
				return nil, err
			}
			all = append(all, summary)
		}
		if len(batch) < listPageSize {
			break
		}
		skip += len(batch)
	}
	return all, nil
}

func (c *LiveClient) getTranscript(ctx context.Context, transcriptID string) (TranscriptDetail, error) {
	transcriptID = strings.TrimSpace(transcriptID)
	if transcriptID == "" {
		return TranscriptDetail{}, fmt.Errorf("transcript id is required")
	}
	var payload struct {
		Transcript TranscriptDetail `json:"transcript"`
	}
	if err := c.postGraphQL(ctx, transcriptDetailQuery, map[string]any{
		"transcriptId": transcriptID,
	}, &payload); err != nil {
		return TranscriptDetail{}, err
	}
	if strings.TrimSpace(payload.Transcript.ID) == "" {
		payload.Transcript.ID = transcriptID
	}
	return payload.Transcript, nil
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

func (c *LiveClient) postGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
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
		return fmt.Errorf("fireflies graphql: http %d", resp.StatusCode)
	}
	var envelope graphQLResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("fireflies graphql decode: %w", err)
	}
	if len(envelope.Errors) > 0 {
		msg := strings.TrimSpace(envelope.Errors[0].Message)
		if msg == "" {
			msg = "graphql error"
		}
		if strings.Contains(strings.ToLower(msg), "unauthorized") || strings.Contains(strings.ToLower(msg), "authentication") {
			return &AuthError{Status: resp.StatusCode}
		}
		return fmt.Errorf("fireflies graphql: %s", msg)
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("fireflies graphql data: %w", err)
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
		return "fireflies api auth failed"
	}
	return fmt.Sprintf("fireflies api auth failed (http %d)", e.Status)
}
