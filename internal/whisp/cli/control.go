package cli

import (
	"os"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/whisp/archive"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	"github.com/openclaw/crawlkit/control"
)

func controlManifest(configPath string, cfg wconfig.Config) control.Manifest {
	manifest := control.NewManifest("whispcrawl", "OpenWhispr Archive", "whispcrawl")
	manifest.Description = "Local-first read-only archive of OpenWhispr meeting notes and transcripts."
	manifest.Paths = control.Paths{
		DefaultConfig:   configPath,
		ConfigEnv:       wconfig.ConfigEnv,
		DefaultDatabase: cfg.DBPath,
		DefaultCache:    cfg.CacheDir,
		DefaultLogs:     cfg.LogDir,
	}
	manifest.Capabilities = []string{"metadata", "status", "doctor", "sync", "search", "sql"}
	manifest.Commands = map[string]control.Command{
		"metadata": {Title: "Metadata", Argv: []string{"whispcrawl", "metadata", "--json"}, JSON: true},
		"status":   {Title: "Status", Argv: []string{"whispcrawl", "status", "--json"}, JSON: true},
		"doctor":   {Title: "Doctor", Argv: []string{"whispcrawl", "doctor", "--json"}, JSON: true},
		"sync":     {Title: "Sync", Argv: []string{"whispcrawl", "sync", "--json"}, JSON: true, Mutates: true},
		"search":   {Title: "Search", Argv: []string{"whispcrawl", "search", "--json"}, JSON: true},
		"sql":      {Title: "Read-only SQL", Argv: []string{"whispcrawl", "--json", "sql", "select count(*) as artifacts from artifacts"}, JSON: true},
	}
	manifest.Privacy = control.Privacy{
		ContainsPrivateMessages: true,
		ExportsSecrets:          false,
		LocalOnlyScopes:         []string{"OpenWhispr transcriptions.db", "whispcrawl SQLite archive"},
	}
	return manifest
}

func controlStatus(configPath string, cfg wconfig.Config, status archive.Status) control.Status {
	counts := []control.Count{
		control.NewCount("artifacts", "Artifacts", int64(status.Artifacts)),
		control.NewCount("sync_runs", "Sync runs", int64(status.SyncRuns)),
	}
	out := control.NewStatus("whispcrawl", "OpenWhispr meeting archive")
	out.State = "ok"
	out.ConfigPath = configPath
	out.DatabasePath = status.DBPath
	out.Counts = counts
	out.DatabaseBytes = fileSize(status.DBPath)
	out.WALBytes = fileSize(status.DBPath + "-wal")
	out.Databases = []control.Database{
		control.SQLiteDatabase("primary", "OpenWhispr archive", "archive", status.DBPath, true, counts),
	}
	if !status.LastSync.IsZero() {
		out.LastSyncAt = status.LastSync.UTC().Format(time.RFC3339)
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
