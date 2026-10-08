package doctor

import (
	"os"

	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
)

type Report struct {
	ConfigPath  string       `json:"config_path"`
	DBPath      string       `json:"db_path"`
	SourceDB    string       `json:"source_db"`
	Source      FileState    `json:"source_db_file"`
	Archive     FileState    `json:"archive_db_file"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type FileState struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Bytes  int64  `json:"bytes,omitempty"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func Run(cfg wconfig.Config, configPath, sourceOverride string) Report {
	sourceDB := sourceOverride
	if sourceDB == "" {
		sourceDB = cfg.SourceDB
	}
	report := Report{
		ConfigPath: configPath,
		DBPath:     cfg.DBPath,
		SourceDB:   sourceDB,
		Source:     statFile(sourceDB),
		Archive:    statFile(cfg.DBPath),
	}
	if !report.Source.Exists {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:     "openwhispr_source_missing",
			Severity: "warning",
			Message:  "OpenWhispr transcriptions.db was not found at the configured path",
		})
	}
	return report
}

func statFile(path string) FileState {
	state := FileState{Path: path}
	info, err := os.Stat(path)
	if err != nil {
		return state
	}
	state.Exists = true
	state.Bytes = info.Size()
	return state
}
