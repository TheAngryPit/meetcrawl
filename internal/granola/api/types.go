package api

import "time"

const BaseURL = "https://public-api.granola.ai"

// NoteIDPattern matches Granola public API note ids (not_ + 14 alphanumeric).
const NoteIDPattern = `^not_[a-zA-Z0-9]{14}$`

type NoteSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ListNotesResponse struct {
	Notes   []NoteSummary `json:"notes"`
	HasMore bool          `json:"hasMore"`
	Cursor  *string       `json:"cursor"`
}

type CalendarEvent struct {
	CalendarEventID    *string `json:"calendar_event_id"`
	ScheduledStartTime *string `json:"scheduled_start_time"`
	ScheduledEndTime   *string `json:"scheduled_end_time"`
}

type NoteDetail struct {
	ID            string         `json:"id"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	CalendarEvent *CalendarEvent `json:"calendar_event"`
}

type TranscriptPage struct {
	Transcript []TranscriptItem `json:"transcript"`
	HasMore    bool             `json:"hasMore"`
	Cursor     *string          `json:"cursor"`
}

type TranscriptItem struct {
	Speaker   Speaker `json:"speaker"`
	Text      string  `json:"text"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
}

type Speaker struct {
	Source           string `json:"source"`
	Attribution      string `json:"attribution,omitempty"`
	DiarizationLabel string `json:"diarization_label,omitempty"`
	Name             string `json:"name,omitempty"`
}

type NoteForSync struct {
	Note            NoteSummary
	CalendarEventID string
	Start           time.Time
	End             time.Time
}
