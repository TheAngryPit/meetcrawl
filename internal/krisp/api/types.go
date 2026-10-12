package api

import "time"

const BaseURL = "https://meeting-api.krisp.ai/v1"

type MeetingSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartedAt string `json:"started_at"`
	Duration  *int   `json:"duration"`
}

type MeetingListResponse struct {
	Meetings   []MeetingSummary `json:"meetings"`
	NextCursor *string          `json:"next_cursor"`
}

type MeetingDetail struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	StartedAt  string              `json:"started_at"`
	Duration   *int                `json:"duration"`
	Transcript *TranscriptDocument `json:"transcript"`
}

type TranscriptDocument struct {
	Segments []TranscriptSegmentRaw `json:"segments"`
	Speakers map[string]Participant `json:"speakers"`
}

type TranscriptSegmentRaw struct {
	Speaker int     `json:"speaker"`
	Text    string  `json:"text"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
}

type Participant struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type TranscriptItem struct {
	SpeakerDisplay string
	Text           string
	StartSeconds   float64
	EndSeconds     float64
}

type MeetingForSync struct {
	Summary  MeetingSummary
	Start    time.Time
	End      time.Time
	Revision string
}
