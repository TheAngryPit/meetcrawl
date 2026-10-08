package oauth_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
)

func TestInstallClient(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "client-src.json")
	dest := filepath.Join(dir, "cfg", "oauth-client.json")
	payload := []byte(`{"installed":{"client_id":"synthetic","client_secret":"synthetic","redirect_uris":["http://127.0.0.1"]}}`)
	if err := os.WriteFile(src, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := oauth.InstallClient(src, dest); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("client mode = %o, want 0600", info.Mode().Perm())
	}
}
