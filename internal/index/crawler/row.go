package crawler

import (
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/source"
)

// Row is one artifact row read from a crawler archive (read-only).
type Row struct {
	Source         source.Kind
	SourceID       string
	SourceRevision string
	Fidelity       source.Fidelity
	Privacy        source.PrivacyClass
	NormalizedText string
	ContentHash    string
	ProvenanceHash string
	WindowStart    time.Time
	WindowEnd      time.Time
	Language       string
	CalendarICal   string
	Participants   string
}
