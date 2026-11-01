package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestOpenWhisprCandidates(t *testing.T) {
	t.Parallel()
	home := filepath.FromSlash("/home/u")
	tests := []struct {
		name string
		goos string
		xdg  string
		app  string
		want []string
	}{
		{
			name: "darwin prefers Application Support",
			goos: "darwin",
			want: []string{
				filepath.Join(home, "Library", "Application Support", "OpenWhispr", "transcriptions.db"),
				filepath.Join(home, "Library", "Application Support", "openwhispr", "transcriptions.db"),
				filepath.Join(home, ".config", "OpenWhispr", "transcriptions.db"),
				filepath.Join(home, ".config", "openwhispr", "transcriptions.db"),
			},
		},
		{
			name: "linux honors XDG_CONFIG_HOME first",
			goos: "linux",
			xdg:  filepath.FromSlash("/x/cfg"),
			want: []string{
				filepath.FromSlash("/x/cfg/OpenWhispr/transcriptions.db"),
				filepath.FromSlash("/x/cfg/openwhispr/transcriptions.db"),
				filepath.Join(home, ".config", "OpenWhispr", "transcriptions.db"),
				filepath.Join(home, ".config", "openwhispr", "transcriptions.db"),
			},
		},
		{
			name: "linux without XDG falls back to ~/.config",
			goos: "linux",
			want: []string{
				filepath.Join(home, ".config", "OpenWhispr", "transcriptions.db"),
				filepath.Join(home, ".config", "openwhispr", "transcriptions.db"),
			},
		},
		{
			name: "relative XDG_CONFIG_HOME is ignored",
			goos: "linux",
			xdg:  "relative/cfg",
			want: []string{
				filepath.Join(home, ".config", "OpenWhispr", "transcriptions.db"),
				filepath.Join(home, ".config", "openwhispr", "transcriptions.db"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := openWhisprCandidates(tt.goos, home, tt.xdg, tt.app)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("candidates = %v, want %v", got, tt.want)
			}
		})
	}
}
