package index_test

import (
	"strings"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/index"
	"github.com/TheAngryPit/meetcrawl/internal/index/crawler"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestAdhocMeetingIDPrefix(t *testing.T) {
	t.Parallel()
	row := crawler.Row{
		Source:      source.KindGMeetGemini,
		SourceID:    "gdrive-adhoc-001",
		WindowStart: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
	}
	ids, err := index.AssignMeetingIDs([]crawler.Row{row})
	if err != nil {
		t.Fatalf("AssignMeetingIDs() = %v", err)
	}
	if !strings.HasPrefix(ids[0], "adhoc:") {
		t.Fatalf("meeting_id = %q, want adhoc: prefix", ids[0])
	}
}
