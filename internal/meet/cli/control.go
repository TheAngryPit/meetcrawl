package cli

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/registry"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/openclaw/crawlkit/control"
)

func ControlManifest(configPath string, cfg mconfig.Config) control.Manifest {
	_ = configPath
	_ = cfg
	manifest := control.NewManifest("meet", "Meetings Index", "meet")
	manifest.Description = "Local-first meetings index with pluggable source adapters."
	manifest.Paths = portableManifestPaths()
	manifest.Capabilities = []string{"metadata", "status", "doctor", "sync", "index", "search"}
	manifest.Commands = map[string]control.Command{
		"metadata": {Title: "Metadata", Argv: []string{"meet", "metadata", "--json"}, JSON: true},
		"status":   {Title: "Status", Argv: []string{"meet", "status", "--json"}, JSON: true},
		"doctor":   {Title: "Doctor", Argv: []string{"meet", "doctor", "--json"}, JSON: true},
		"index":    {Title: "Index", Argv: []string{"meet", "index", "--json"}, JSON: true, Mutates: true},
		"search":   {Title: "Search", Argv: []string{"meet", "search", "--json"}, JSON: true},
	}
	for _, name := range registry.Names() {
		key := syncCommandKey(name)
		manifest.Commands[key] = control.Command{
			Title:   "Sync " + string(name),
			Argv:    []string{"meet", "sync", "--json", "--source", string(name)},
			JSON:    true,
			Mutates: true,
		}
	}
	manifest.Privacy = control.Privacy{
		ContainsPrivateMessages: true,
		ExportsSecrets:          false,
		LocalOnlyScopes:         registry.LocalOnlyScopes(),
	}
	return manifest
}

func syncCommandKey(name registry.Name) string {
	return "sync-" + strings.ReplaceAll(string(name), "-", "_")
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
