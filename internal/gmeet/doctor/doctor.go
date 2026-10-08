package doctor

import (
	"os"
	"path/filepath"
	"strings"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
)

type Report struct {
	ConfigPath      string       `json:"config_path"`
	DBPath          string       `json:"db_path"`
	FixtureDir      string       `json:"fixture_dir,omitempty"`
	OAuthClient     string       `json:"oauth_client_path"`
	TokenPath       string       `json:"token_path"`
	Archive         FileState    `json:"archive_db_file"`
	Fixture         FileState    `json:"fixture_manifest,omitempty"`
	OAuthClientFile FileState    `json:"oauth_client_file"`
	TokenFile       FileState    `json:"token_file"`
	Diagnostics     []Diagnostic `json:"diagnostics,omitempty"`
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

func Run(cfg gconfig.Config, configPath, fixtureOverride string) Report {
	fixtureDir := strings.TrimSpace(fixtureOverride)
	report := Report{
		ConfigPath:      configPath,
		DBPath:          cfg.DBPath,
		FixtureDir:      fixtureDir,
		OAuthClient:     cfg.OAuthClientPath,
		TokenPath:       cfg.TokenPath,
		Archive:         statFile(cfg.DBPath),
		OAuthClientFile: statFile(cfg.OAuthClientPath),
		TokenFile:       statFile(cfg.TokenPath),
	}
	if fixtureDir != "" {
		report.Fixture = statFile(filepath.Join(fixtureDir, "manifest.json"))
		if !report.Fixture.Exists {
			report.Diagnostics = append(report.Diagnostics, Diagnostic{
				Code:     "fixture_manifest_missing",
				Severity: "error",
				Message:  "Fixture manifest.json was not found under --fixture",
			})
		}
		return report
	}
	if !report.OAuthClientFile.Exists {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:     "oauth_client_missing",
			Severity: "warning",
			Message:  "OAuth client JSON not found; use --fixture for offline sync or place client JSON outside the archive",
		})
	}
	if !report.TokenFile.Exists {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:     "oauth_token_missing",
			Severity: "warning",
			Message:  "OAuth token not found; live sync needs a token file outside the archive",
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
