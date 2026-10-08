package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/openclaw/crawlkit/control"
)

func ControlManifest(configPath string, cfg mconfig.Config) control.Manifest {
	manifest := control.NewManifest("meetcrawl", "Meetings Index", "meetcrawl")
	manifest.Description = "Local-first meetings index joining whispcrawl and gmeetcrawl archives."
	manifest.Paths = control.Paths{
		DefaultConfig:   portableHomePath(configPath),
		ConfigEnv:       mconfig.ConfigEnv,
		DefaultDatabase: portableHomePath(cfg.DBPath),
	}
	manifest.Capabilities = []string{"metadata", "status", "doctor", "index", "search"}
	manifest.Commands = map[string]control.Command{
		"metadata": {Title: "Metadata", Argv: []string{"meetcrawl", "metadata", "--json"}, JSON: true},
		"status":   {Title: "Status", Argv: []string{"meetcrawl", "status", "--json"}, JSON: true},
		"doctor":   {Title: "Doctor", Argv: []string{"meetcrawl", "doctor", "--json"}, JSON: true},
		"index":    {Title: "Index", Argv: []string{"meetcrawl", "index", "--json"}, JSON: true, Mutates: true},
		"search":   {Title: "Search", Argv: []string{"meetcrawl", "search", "--json"}, JSON: true},
	}
	manifest.Privacy = control.Privacy{
		ContainsPrivateMessages: true,
		ExportsSecrets:          false,
		LocalOnlyScopes: []string{
			"whispcrawl SQLite archive",
			"gmeetcrawl SQLite archive",
			"meetcrawl index SQLite archive",
		},
	}
	return manifest
}

func ValidateManifest(m control.Manifest) error {
	if m.SchemaVersion != control.SchemaVersion {
		return fmt.Errorf("schema_version: want %q, got %q", control.SchemaVersion, m.SchemaVersion)
	}
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(m.DisplayName) == "" {
		return fmt.Errorf("display_name is required")
	}
	if strings.TrimSpace(m.Binary.Name) == "" {
		return fmt.Errorf("binary.name is required")
	}
	if len(m.Commands) == 0 {
		return fmt.Errorf("commands are required")
	}
	for name, cmd := range m.Commands {
		if len(cmd.Argv) == 0 {
			return fmt.Errorf("command %q: argv is required", name)
		}
	}
	if !m.Privacy.ContainsPrivateMessages {
		return fmt.Errorf("privacy.contains_private_messages must be true")
	}
	if m.Privacy.ExportsSecrets {
		return fmt.Errorf("privacy.exports_secrets must be false")
	}
	if len(m.Privacy.LocalOnlyScopes) == 0 {
		return fmt.Errorf("privacy.local_only_scopes is required")
	}
	return nil
}

// portableHomePath rewrites absolute paths under the user home as ~/… so crawlbar
// manifests stay portable. crawlkit.config.ExpandHome accepts ~/ at runtime.
func portableHomePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(filepath.Separator)
	if strings.HasPrefix(path, prefix) {
		return "~/" + strings.TrimPrefix(path, prefix)
	}
	return path
}
