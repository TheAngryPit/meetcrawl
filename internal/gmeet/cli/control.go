package cli

import (
	"os"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/archive"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/openclaw/crawlkit/control"
)

func controlManifest(configPath string, cfg gconfig.Config) control.Manifest {
	manifest := control.NewManifest("gmeetcrawl", "Meet Gemini Archive", "gmeetcrawl")
	manifest.Description = "Local-first read-only archive of Google Meet Gemini notes and transcript Docs."
	manifest.Paths = control.Paths{
		DefaultConfig:   configPath,
		ConfigEnv:       gconfig.ConfigEnv,
		DefaultDatabase: cfg.DBPath,
		DefaultCache:    cfg.CacheDir,
		DefaultLogs:     cfg.LogDir,
	}
	manifest.Capabilities = []string{"metadata", "status", "doctor", "auth", "sync", "search", "sql"}
	manifest.Commands = map[string]control.Command{
		"metadata": {Title: "Metadata", Argv: []string{"gmeetcrawl", "metadata", "--json"}, JSON: true},
		"status":   {Title: "Status", Argv: []string{"gmeetcrawl", "status", "--json"}, JSON: true},
		"doctor":   {Title: "Doctor", Argv: []string{"gmeetcrawl", "doctor", "--json"}, JSON: true},
		"auth":     {Title: "OAuth login", Argv: []string{"gmeetcrawl", "auth", "--json"}, JSON: true, Mutates: true},
		"sync":     {Title: "Sync", Argv: []string{"gmeetcrawl", "sync", "--json"}, JSON: true, Mutates: true},
		"search":   {Title: "Search", Argv: []string{"gmeetcrawl", "search", "--json"}, JSON: true},
		"sql":      {Title: "Read-only SQL", Argv: []string{"gmeetcrawl", "--json", "sql", "select count(*) as artifacts from artifacts"}, JSON: true},
	}
	manifest.Privacy = control.Privacy{
		ContainsPrivateMessages: true,
		ExportsSecrets:          false,
		LocalOnlyScopes:         []string{"gmeetcrawl SQLite archive", "OAuth token path (outside archive)"},
	}
	return manifest
}

func controlStatus(configPath string, cfg gconfig.Config, status archive.Status) control.Status {
	counts := []control.Count{
		control.NewCount("artifacts", "Artifacts", int64(status.Artifacts)),
		control.NewCount("sync_runs", "Sync runs", int64(status.SyncRuns)),
	}
	out := control.NewStatus("gmeetcrawl", "Meet Gemini meeting archive")
	out.State = "ok"
	out.ConfigPath = configPath
	out.DatabasePath = status.DBPath
	out.Counts = counts
	out.DatabaseBytes = fileSize(status.DBPath)
	out.WALBytes = fileSize(status.DBPath + "-wal")
	out.Databases = []control.Database{
		control.SQLiteDatabase("primary", "Meet Gemini archive", "archive", status.DBPath, true, counts),
	}
	if !status.LastSync.IsZero() {
		out.LastSyncAt = status.LastSync.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	out.Remote = &control.Remote{Enabled: false}
	return out
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
