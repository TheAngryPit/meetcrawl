package cli

import (
	"fmt"
	"runtime"
	"strings"

	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/openclaw/crawlkit/control"
)

func ControlManifest(configPath string, cfg mconfig.Config) control.Manifest {
	_ = configPath
	_ = cfg
	manifest := control.NewManifest("meetcrawl", "Meetings Index", "meetcrawl")
	manifest.Description = "Local-first meetings index joining whispcrawl and gmeetcrawl archives."
	manifest.Paths = portableManifestPaths()
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

// portableManifestPaths returns crawlkit default locations as ~/… strings (see
// crawlkit config.platformPaths fallbacks). Manifest paths ignore XDG_* overrides
// so shipped crawlbar JSON stays copy-safe; MEETCRAWL_CONFIG still wins at runtime.
func portableManifestPaths() control.Paths {
	out := control.Paths{ConfigEnv: mconfig.ConfigEnv}
	switch runtime.GOOS {
	case "darwin":
		out.DefaultConfig = "~/Library/Application Support/meetcrawl/config.toml"
		out.DefaultDatabase = "~/Library/Application Support/meetcrawl/meetcrawl.db"
	case "windows":
		out.DefaultConfig = "~/AppData/Local/meetcrawl/config.toml"
		out.DefaultDatabase = "~/AppData/Local/meetcrawl/meetcrawl.db"
	default:
		out.DefaultConfig = "~/.config/meetcrawl/config.toml"
		out.DefaultDatabase = "~/.local/share/meetcrawl/meetcrawl.db"
	}
	return out
}
