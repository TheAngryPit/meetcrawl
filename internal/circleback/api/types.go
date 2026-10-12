package api

import "time"

const BaseURL = "https://circleback.ai/api"

type CalendarEvent struct {
	IcalUID    string `json:"icalUid"`
	PlatformID string `json:"platformId"`
}

type MeetingSummary struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	Duration      float64        `json:"duration"`
	IcalUID       string         `json:"icalUid"`
	CalendarEvent *CalendarEvent `json:"calendarEvent"`
}

type TranscriptSegmentRaw struct {
	Speaker   string  `json:"speaker"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}

type TranscriptItem struct {
	SpeakerDisplay string
	Text           string
	StartSeconds   float64
	EndSeconds     float64
}

type MeetingForSync struct {
	Summary         MeetingSummary
	Start           time.Time
	End             time.Time
	Revision        string
	CalendarEventID string
}
