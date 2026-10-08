package doctor

import (
	"os"

	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
)

type Report struct {
	ConfigPath    string `json:"config_path"`
	IndexDB       string `json:"index_db"`
	ReadsDB       string `json:"reads_db"`
	WhispcrawlDB  string `json:"whispcrawl_db"`
	GmeetcrawlDB  string `json:"gmeetcrawl_db"`
	ExportcrawlDB string `json:"exportcrawl_db"`
	Archives      struct {
		Whispcrawl  Exists `json:"whispcrawl"`
		Gmeetcrawl  Exists `json:"gmeetcrawl"`
		Exportcrawl Exists `json:"exportcrawl"`
		Index       Exists `json:"index"`
		Reads       Exists `json:"reads"`
	} `json:"archives"`
}

type Exists struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func Run(cfg mconfig.Config, configPath string) Report {
	report := Report{
		ConfigPath:    configPath,
		IndexDB:       cfg.DBPath,
		ReadsDB:       cfg.ReadsDBPath,
		WhispcrawlDB:  cfg.WhispcrawlDB,
		GmeetcrawlDB:  cfg.GmeetcrawlDB,
		ExportcrawlDB: cfg.ExportcrawlDB,
	}
	report.Archives.Whispcrawl = statPath(cfg.WhispcrawlDB)
	report.Archives.Gmeetcrawl = statPath(cfg.GmeetcrawlDB)
	report.Archives.Exportcrawl = statPath(cfg.ExportcrawlDB)
	report.Archives.Index = statPath(cfg.DBPath)
	report.Archives.Reads = statPath(cfg.ReadsDBPath)
	return report
}

func statPath(path string) Exists {
	_, err := os.Stat(path)
	return Exists{Path: path, Exists: err == nil}
}
