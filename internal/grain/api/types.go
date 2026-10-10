package api

import "time"

const (
	BaseURL          = "https://api.grain.com/_/public-api/v2"
	APIVersion       = "2025-10-31"
	HeaderAPIVersion = "Public-Api-Version"
)

type CalendarEvent struct {
	ICalUID                *string `json:"ical_uid"`
	ScheduledStartDatetime *string `json:"scheduled_start_datetime"`
	ScheduledEndDatetime   *string `json:"scheduled_end_datetime"`
}

type Recording struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	StartDatetime string         `json:"start_datetime"`
	EndDatetime   string         `json:"end_datetime,omitempty"`
	CalendarEvent *CalendarEvent `json:"calendar_event,omitempty"`
}

type ListRecordingsResponse struct {
	Cursor     *string     `json:"cursor"`
	Recordings []Recording `json:"recordings"`
}

type TranscriptSegment struct {
	Start         int64   `json:"start"`
	End           int64   `json:"end"`
	Text          string  `json:"text"`
	Speaker       string  `json:"speaker"`
	ParticipantID *string `json:"participant_id"`
}

type RecordingSummary struct {
	Recording Recording
	Start     time.Time
	End       time.Time
}
