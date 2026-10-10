package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteRepoFixtures(t *testing.T) {
	if os.Getenv("MEETCRAWL_WRITE_FIXTURES") != "1" {
		t.Skip("set MEETCRAWL_WRITE_FIXTURES=1 to regenerate testdata fixtures")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "fixtures", "openwhispr"))
	if err := WriteSupportedDB(filepath.Join(root, "supported", "transcriptions.db")); err != nil {
		t.Fatalf("WriteSupportedDB() = %v", err)
	}
	if err := WriteUnsupportedDB(filepath.Join(root, "unsupported", "transcriptions.db")); err != nil {
		t.Fatalf("WriteUnsupportedDB() = %v", err)
	}
	if err := WriteIOSSupportedDB(filepath.Join(root, "ios-supported", "transcriptions.db")); err != nil {
		t.Fatalf("WriteIOSSupportedDB() = %v", err)
	}
}
