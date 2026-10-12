package api_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/circleback/api"
)

func TestParseNextCursor(t *testing.T) {
	t.Parallel()
	link := `</api/meetings?cursor=eyJwYWdlIjoxfQ>; rel="next"`
	got := api.ParseNextCursor(link)
	if got != "eyJwYWdlIjoxfQ" {
		t.Fatalf("cursor = %q", got)
	}
}
