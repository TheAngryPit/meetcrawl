package doctor

import (
	"os"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/registry"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
)

type Report struct {
	ConfigPath string           `json:"config_path"`
	IndexDB    string           `json:"index_db"`
	ReadsDB    string           `json:"reads_db"`
	Sources    []SourceArchive  `json:"sources"`
	Archives   IndexArchiveStat `json:"archives"`
}

type SourceArchive struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type IndexArchiveStat struct {
	Index Exists `json:"index"`
	Reads Exists `json:"reads"`
}

type Exists struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func Run(cfg mconfig.Config, configPath string) Report {
	report := Report{
		ConfigPath: configPath,
		IndexDB:    cfg.DBPath,
		ReadsDB:    cfg.ReadsDBPath,
	}
	for _, name := range registry.Names() {
		e, _ := registry.EntryFor(name)
		path := e.ArchivePath(mconfig.ConfigView(cfg))
		report.Sources = append(report.Sources, SourceArchive{
			Name:   string(name),
			Kind:   string(e.Kind),
			Path:   path,
			Exists: fileExists(path),
		})
	}
	report.Archives.Index = statPath(cfg.DBPath)
	report.Archives.Reads = statPath(cfg.ReadsDBPath)
	return report
}

func statPath(path string) Exists {
	return Exists{Path: path, Exists: fileExists(path)}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
