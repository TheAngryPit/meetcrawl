package oauth_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

type memKeyring map[string]string

func (m memKeyring) Get(service, account string) (string, error) {
	v, ok := m[service+"/"+account]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func (m memKeyring) Set(service, account, password string) error {
	m[service+"/"+account] = password
	return nil
}

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

func TestTokenStoreKeychainSuccessRemovesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	if err := os.WriteFile(path, []byte(`{"access_token":"stale"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	kr := memKeyring{}
	store := oauth.TokenStore{FilePath: path, KeyringEnabled: true, Keyring: kr}
	tok := &oauth2.Token{AccessToken: "fresh", TokenType: "Bearer"}
	storage, err := store.Save(tok)
	if err != nil {
		t.Fatal(err)
	}
	if storage != "keychain" {
		t.Fatalf("storage = %q, want keychain", storage)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("token file still present after keychain save: %v", err)
	}
	if store.StorageInUse() != "keychain" {
		t.Fatalf("StorageInUse = %q", store.StorageInUse())
	}
}

func TestTokenStoreMissingToken(t *testing.T) {
	t.Parallel()
	store := oauth.TokenStore{FilePath: filepath.Join(t.TempDir(), "missing.json"), KeyringEnabled: false}
	if _, err := store.Load(); err == nil {
		t.Fatal("expected error for missing token")
	}
}

type deniedKeyring struct{ memKeyring }

func (deniedKeyring) Get(string, string) (string, error) { return "", errors.New("access denied") }

func TestTokenStoreReportsUnreadableKeychainOnlyWithoutAFile(t *testing.T) {
	t.Parallel()
	missingFile := filepath.Join(t.TempDir(), "missing.json")

	denied := oauth.TokenStore{FilePath: missingFile, KeyringEnabled: true, Keyring: deniedKeyring{}}
	_, err := denied.Load()
	if err == nil || errors.Is(err, oauth.ErrTokenNotFound) {
		t.Fatalf("Load() with a denied keychain and no file = %v, want the keychain error", err)
	}

	absent := oauth.TokenStore{FilePath: missingFile, KeyringEnabled: true, Keyring: memKeyring{}}
	if _, err := absent.Load(); !errors.Is(err, oauth.ErrTokenNotFound) {
		t.Fatalf("Load() with an empty keychain and no file = %v, want ErrTokenNotFound", err)
	}

	file := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(file, []byte(`{"access_token":"from-file","token_type":"Bearer"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fallback := oauth.TokenStore{FilePath: file, KeyringEnabled: true, Keyring: deniedKeyring{}}
	tok, err := fallback.Load()
	if err != nil || tok.AccessToken != "from-file" {
		t.Fatalf("Load() with a denied keychain and a token file = %v, %v, want the file token", tok, err)
	}
}
