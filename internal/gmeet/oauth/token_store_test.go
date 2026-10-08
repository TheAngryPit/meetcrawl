package oauth_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
	"golang.org/x/oauth2"
)

func TestTokenStoreFileRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	store := oauth.TokenStore{FilePath: path, KeyringEnabled: false}
	tok := &oauth2.Token{
		AccessToken: "synthetic-access",
		TokenType:   "Bearer",
		Expiry:      time.Now().UTC().Add(time.Hour),
	}
	storage, err := store.Save(tok)
	if err != nil {
		t.Fatal(err)
	}
	if storage != "file" {
		t.Fatalf("storage = %q, want file", storage)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("token mode = %o, want 0600", info.Mode().Perm())
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccessToken != tok.AccessToken {
		t.Fatalf("AccessToken = %q", loaded.AccessToken)
	}
}

func TestTokenStoreMissingToken(t *testing.T) {
	t.Parallel()
	store := oauth.TokenStore{FilePath: filepath.Join(t.TempDir(), "missing.json"), KeyringEnabled: false}
	if _, err := store.Load(); err == nil {
		t.Fatal("expected error for missing token")
	}
}
