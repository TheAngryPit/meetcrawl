package api

import "time"

const GraphQLURL = "https://api.fireflies.ai/graphql"

const listPageSize = 50

type TranscriptSummary struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Date       float64 `json:"date"`
	DateString string  `json:"dateString"`
	CalendarID string  `json:"calendar_id"`
}

type Sentence struct {
	Index       int     `json:"index"`
	SpeakerName string  `json:"speaker_name"`
	SpeakerID   string  `json:"speaker_id"`
	Text        string  `json:"text"`
	StartTime   float64 `json:"start_time"`
	EndTime     float64 `json:"end_time"`
}

type TranscriptDetail struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Date       float64    `json:"date"`
	DateString string     `json:"dateString"`
	CalendarID string     `json:"calendar_id"`
	Duration   float64    `json:"duration"`
	Sentences  []Sentence `json:"sentences"`
}

type TranscriptForSync struct {
	Summary    TranscriptSummary
	CalendarID string
	Start      time.Time
	End        time.Time
	Revision   string
}
