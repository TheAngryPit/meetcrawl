package schema_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/whisp/schema"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/testutil"
	_ "modernc.org/sqlite"
)

func TestDetectDesktopLayout(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "desktop.db")
	if err := testutil.WriteSupportedDB(path); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	layout, err := schema.DetectNotesLayout(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if layout != schema.LayoutDesktop {
		t.Fatalf("layout = %v, want desktop", layout)
	}
}

func TestDetectIOSLayout(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "ios.db")
	if err := testutil.WriteIOSSupportedDB(path); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	layout, err := schema.DetectNotesLayout(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if layout != schema.LayoutIOS {
		t.Fatalf("layout = %v, want ios", layout)
	}
}

func TestUnsupportedLayoutFailsClosed(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.db")
	if err := testutil.WriteUnsupportedDB(path); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := schema.DetectNotesLayout(context.Background(), db); err == nil {
		t.Fatal("expected unsupported layout error")
	}
}
