package cli

import (
	"os"

	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/archive"
	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	"github.com/openclaw/crawlkit/control"
)

func controlManifest(configPath string, cfg econfig.Config) control.Manifest {
	manifest := control.NewManifest("exportcrawl", "Export File Archive", "exportcrawl")
	manifest.Description = "Local-first read-only archive of exported meeting transcripts (VTT, SRT, TXT, Markdown)."
	manifest.Paths = control.Paths{
		DefaultConfig:   configPath,
		ConfigEnv:       econfig.ConfigEnv,
		DefaultDatabase: cfg.DBPath,
		DefaultCache:    cfg.CacheDir,
		DefaultLogs:     cfg.LogDir,
	}
	manifest.Capabilities = []string{"metadata", "status", "doctor", "sync", "search", "sql"}
	manifest.Commands = map[string]control.Command{
		"metadata": {Title: "Metadata", Argv: []string{"exportcrawl", "metadata", "--json"}, JSON: true},
		"status":   {Title: "Status", Argv: []string{"exportcrawl", "status", "--json"}, JSON: true},
		"doctor":   {Title: "Doctor", Argv: []string{"exportcrawl", "doctor", "--json"}, JSON: true},
		"sync":     {Title: "Sync", Argv: []string{"exportcrawl", "sync", "--json"}, JSON: true, Mutates: true},
		"search":   {Title: "Search", Argv: []string{"exportcrawl", "search", "--json"}, JSON: true},
		"sql":      {Title: "Read-only SQL", Argv: []string{"exportcrawl", "--json", "sql", "select count(*) as artifacts from artifacts"}, JSON: true},
	}
	manifest.Privacy = control.Privacy{
		ContainsPrivateMessages: true,
		ExportsSecrets:          false,
		LocalOnlyScopes:         []string{"export source directory (read-only)", "exportcrawl SQLite archive"},
	}
	return manifest
}

func controlStatus(configPath string, cfg econfig.Config, status archive.Status) control.Status {
	counts := []control.Count{
		control.NewCount("artifacts", "Artifacts", int64(status.Artifacts)),
		control.NewCount("sync_runs", "Sync runs", int64(status.SyncRuns)),
	}
	out := control.NewStatus("exportcrawl", "Exported meeting file archive")
	out.State = "ok"
	out.ConfigPath = configPath
	out.DatabasePath = status.DBPath
	out.Counts = counts
	out.DatabaseBytes = fileSize(status.DBPath)
	out.WALBytes = fileSize(status.DBPath + "-wal")
	out.Databases = []control.Database{
		control.SQLiteDatabase("primary", "Export file archive", "archive", status.DBPath, true, counts),
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
