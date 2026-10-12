package api

import "time"

const BaseURL = "https://pasta.tldv.io"

type MeetingListResponse struct {
	Page     int              `json:"page"`
	Pages    int              `json:"pages"`
	Total    int              `json:"total"`
	PageSize int              `json:"pageSize"`
	Results  []MeetingSummary `json:"results"`
}

type MeetingSummary struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	HappenedAt string  `json:"happenedAt"`
	Duration   float64 `json:"duration"`
}

type TranscriptResponse struct {
	ID        string                 `json:"id"`
	MeetingID string                 `json:"meetingId"`
	Data      []TranscriptSegmentRaw `json:"data"`
}

type TranscriptSegmentRaw struct {
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text"`
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
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
