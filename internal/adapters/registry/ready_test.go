package registry

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/secret"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	"github.com/zalando/go-keyring"
)

type failingProvider struct{ err error }

func (p failingProvider) Resolve(context.Context, string) (string, error) { return "", p.err }

func TestFileReady(t *testing.T) {
	t.Parallel()
	existing := filepath.Join(t.TempDir(), "present.db")
	if err := os.WriteFile(existing, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		path      string
		wantReady bool
		wantIn    string
	}{
		{"present", existing, true, ""},
		{"absent", filepath.Join(t.TempDir(), "missing.db"), false, "not found"},
		{"unset", "  ", false, "not configured"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := fileReady(tt.path, "thing")
			if got.Ready != tt.wantReady || !strings.Contains(got.Reason, tt.wantIn) {
				t.Fatalf("fileReady() = %+v, want ready=%v reason containing %q", got, tt.wantReady, tt.wantIn)
			}
		})
	}
}

func TestSecretReady(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name      string
		provider  secret.Provider
		ref       string
		wantReady bool
		wantIn    string
	}{
		{"present", secret.MapProvider{"k": "v"}, "k", true, ""},
		{"fallback ref used when unset", secret.MapProvider{"fallback": "v"}, "", true, ""},
		{"missing", secret.MapProvider{}, "k", false, "no secret k"},
		{"store unavailable", failingProvider{errors.New("no keychain")}, "k", false, "unavailable: no keychain"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := secretReady(ctx, Deps{Secrets: tt.provider}, tt.ref, "fallback")
			if got.Ready != tt.wantReady || !strings.Contains(got.Reason, tt.wantIn) {
				t.Fatalf("secretReady() = %+v, want ready=%v reason containing %q", got, tt.wantReady, tt.wantIn)
			}
		})
	}
}

func TestTokenReadySeparatesAbsentFromUnreadable(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()

	absent := tokenReady(filepath.Join(dir, "none.json"))
	if absent.Ready || !strings.Contains(absent.Reason, "meetcrawl auth") {
		t.Fatalf("absent token = %+v, want skipped with a meetcrawl auth hint", absent)
	}

	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := tokenReady(broken)
	if got.Ready || !strings.Contains(got.Reason, "unreadable") || strings.Contains(got.Reason, "meetcrawl auth") {
		t.Fatalf("malformed token = %+v, want skipped as unreadable with the cause", got)
	}

	valid := filepath.Join(dir, "valid.json")
	if err := os.WriteFile(valid, []byte(`{"access_token":"x","token_type":"Bearer"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := tokenReady(valid); !got.Ready {
		t.Fatalf("valid token = %+v, want ready", got)
	}
}

func TestReadyForSkipsACheckThatDoesNotFinish(t *testing.T) {
	const name Name = "ready-timeout-test"
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	Register(Entry{
		Name:        name,
		Kind:        source.Kind("ready-timeout-test"),
		ArchivePath: func(ConfigView) string { return "" },
		Ready: func(context.Context, Deps) Readiness {
			<-release
			return ready()
		},
		NewAdapter: func(Deps, SyncOptions) source.Adapter { return nil },
	})
	t.Cleanup(func() { Unregister(name) })

	prev := readyTimeout
	readyTimeout = 50 * time.Millisecond
	t.Cleanup(func() { readyTimeout = prev })

	started := time.Now()
	got := ReadyFor(context.Background(), name, Deps{})
	if got.Ready || !strings.Contains(got.Reason, "timed out") {
		t.Fatalf("ReadyFor() = %+v, want skipped as timed out", got)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("ReadyFor() took %v, want it bounded by the timeout", elapsed)
	}
}
