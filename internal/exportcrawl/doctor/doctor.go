package doctor

import (
	"os"

	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
)

type Report struct {
	ConfigPath  string       `json:"config_path"`
	DBPath      string       `json:"db_path"`
	SourceDir   string       `json:"source_dir"`
	Source      FileState    `json:"source_dir_path"`
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

func Run(cfg econfig.Config, configPath, sourceOverride string) Report {
	sourceDir := sourceOverride
	if sourceDir == "" {
		sourceDir = cfg.SourceDir
	}
	report := Report{
		ConfigPath: configPath,
		DBPath:     cfg.DBPath,
		SourceDir:  sourceDir,
		Source:     statPath(sourceDir),
		Archive:    statFile(cfg.DBPath),
	}
	if sourceDir != "" && !report.Source.Exists {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:     "export_source_missing",
			Severity: "warning",
			Message:  "Configured export source directory was not found",
		})
	}
	return report
}

func statPath(path string) FileState {
	state := FileState{Path: path}
	info, err := os.Stat(path)
	if err != nil {
		return state
	}
	state.Exists = true
	if info.IsDir() {
		return state
	}
	state.Bytes = info.Size()
	return state
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
