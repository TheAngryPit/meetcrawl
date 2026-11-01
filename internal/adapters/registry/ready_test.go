package registry

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/secret"
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
