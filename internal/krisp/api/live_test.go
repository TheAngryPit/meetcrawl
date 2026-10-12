package api_test

import (
	"context"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/krisp/api"
)

func TestNewLiveRejectsEmptyToken(t *testing.T) {
	t.Parallel()
	if _, err := api.NewLive(context.Background(), "  "); err == nil {
		t.Fatal("expected error")
	}
}
