package detect_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/detect"
)

func TestIsGeminiDocTitle(t *testing.T) {
	t.Parallel()
	if !detect.IsGeminiDocTitle("Reunião iniciada a 2026/01/15 14:00 WET – Notas do Gemini") {
		t.Fatal("expected PT title match")
	}
	if !detect.IsGeminiDocTitle("Reunião iniciada a 2026/01/15 14:00 GMT+01:00 - Notas do Gemini") {
		t.Fatal("expected PT title with offset TZ and hyphen")
	}
	if !detect.IsGeminiDocTitle("NovaRelay | Sprint kickoff – 2026/03/12 09:30 WEST – Notas do Gemini") {
		t.Fatal("expected prefixed PT title with event name")
	}
	if !detect.IsGeminiDocTitle("NovaRelay | Sprint kickoff - 2026/03/12 09:30 GMT+01:00 - Notes by Gemini") {
		t.Fatal("expected prefixed EN title with event name")
	}
	if !detect.IsGeminiDocTitle("Synthetic standup - Notes by Gemini") {
		t.Fatal("expected EN title match")
	}
	if detect.IsGeminiDocTitle("Random doc") {
		t.Fatal("expected no match")
	}
	if detect.IsGeminiDocTitle("NovaRelay planning notes – 2026/03/12 09:30 WEST") {
		t.Fatal("expected non-Gemini title to fail closed")
	}
}

func TestInFolderScopeDefaultAndCustom(t *testing.T) {
	t.Parallel()
	if !detect.InFolderScope("Google Meet/krr-xxzz-qqq - 2026/01/15 14:00 WET", []string{"Google Meet"}) {
		t.Fatal("expected default root match")
	}
	if !detect.InFolderScope("Synthetic Archive/flat-doc", []string{"Synthetic Archive"}) {
		t.Fatal("expected custom root match")
	}
	if detect.InFolderScope("Other/place", []string{"Google Meet"}) {
		t.Fatal("expected out of scope")
	}
	if !detect.InFolderScope("folder-id-01234567890123456789/sub", []string{"folder-id-01234567890123456789"}) {
		t.Fatal("expected folder id root match")
	}
}
