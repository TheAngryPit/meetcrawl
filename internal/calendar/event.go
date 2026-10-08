package calendar

import "time"

// Event is one Google Calendar event used for optional index-time enrichment.
type Event struct {
	ID                string
	ICalUID           string
	Start             time.Time
	End               time.Time
	AttendeeCount     int
	AttachmentFileIDs []string
	HangoutLink       string
}
