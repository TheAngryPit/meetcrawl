package api

import "time"

const BaseURL = "https://api.fathom.ai/external/v1"

type MeetingSummary struct {
	Title              string `json:"title"`
	RecordingID        int64  `json:"recording_id"`
	CreatedAt          string `json:"created_at"`
	RecordingStartTime string `json:"recording_start_time"`
	RecordingEndTime   string `json:"recording_end_time"`
}

type MeetingListResponse struct {
	NextCursor *string          `json:"next_cursor"`
	Items      []MeetingSummary `json:"items"`
}

type TranscriptResponse struct {
	Transcript []TranscriptItem `json:"transcript"`
}

type TranscriptItemSpeaker struct {
	DisplayName string `json:"display_name"`
}

type TranscriptItem struct {
	Speaker   TranscriptItemSpeaker `json:"speaker"`
	Text      string                `json:"text"`
	Timestamp string                `json:"timestamp"`
}

type MeetingForSync struct {
	Summary  MeetingSummary
	Start    time.Time
	End      time.Time
	Revision string
}
