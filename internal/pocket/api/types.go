package api

import "time"

const BaseURL = "https://public.heypocketai.com/api/v1"

type listRecordingsResponse struct {
	Success    bool              `json:"success"`
	Data       []MeetingSummary  `json:"data"`
	Pagination paginationPayload `json:"pagination"`
}

type paginationPayload struct {
	HasMore bool `json:"has_more"`
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
}

type recordingDetailResponse struct {
	Success bool            `json:"success"`
	Data    recordingDetail `json:"data"`
}

type recordingDetail struct {
	ID         string              `json:"id"`
	Transcript *transcriptEnvelope `json:"transcript"`
}

type transcriptEnvelope struct {
	Segments []TranscriptSegmentRaw `json:"segments"`
}

// MeetingSummary mirrors one Pocket list/detail recording row (transcripts only).
type MeetingSummary struct {
	ID         string  `json:"id"`
	Name       string  `json:"title"`
	HappenedAt string  `json:"recording_at"`
	Duration   float64 `json:"duration"`
	UpdatedAt  string  `json:"updated_at"`
}

type TranscriptSegmentRaw struct {
	Speaker     string  `json:"speaker"`
	SpeakerName string  `json:"speakerName"`
	Text        string  `json:"text"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
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
