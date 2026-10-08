package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestSourceIDForPathUsesRealSHA256(t *testing.T) {
	t.Parallel()
	name := "standup.vtt"
	got := sourceIDForPath(name)
	sum := sha256.Sum256([]byte(name))
	want := "sha256:" + hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("SourceIDForPath() = %q, want %q", got, want)
	}
}
