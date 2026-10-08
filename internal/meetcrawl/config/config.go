package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/index/privacy"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	ckconfig "github.com/openclaw/crawlkit/config"
	"github.com/pelletier/go-toml/v2"
)

const (
	AppName   = "meetcrawl"
	ConfigEnv = "MEETCRAWL_CONFIG"
)

type MCPConfig struct {
	RestrictedAllowlist []string `toml:"restricted_allowlist" json:"restricted_allowlist"`
}

type CalendarConfig struct {
	FixtureDir      string `toml:"fixture_dir,omitempty" json:"fixture_dir,omitempty"`
	OAuthClientPath string `toml:"oauth_client_path,omitempty" json:"oauth_client_path,omitempty"`
	TokenPath       string `toml:"token_path,omitempty" json:"token_path,omitempty"`
}

type Config struct {
	Version       int            `toml:"version" json:"version"`
	DBPath        string         `toml:"db_path" json:"db_path"`
	ReadsDBPath   string         `toml:"reads_db_path" json:"reads_db_path"`
	WhispcrawlDB  string         `toml:"whispcrawl_db" json:"whispcrawl_db"`
	GmeetcrawlDB  string         `toml:"gmeetcrawl_db" json:"gmeetcrawl_db"`
	ExportcrawlDB string         `toml:"exportcrawl_db" json:"exportcrawl_db"`
	Privacy       privacy.Config `toml:"privacy" json:"privacy"`
	Calendar      CalendarConfig `toml:"calendar" json:"calendar"`
	MCP           MCPConfig      `toml:"mcp" json:"mcp"`
}

func App() ckconfig.App {
	return ckconfig.App{
		Name:         AppName,
		ConfigEnv:    ConfigEnv,
		PlatformDirs: runtime.GOOS == "darwin" || runtime.GOOS == "windows" || runtime.GOOS == "linux",
	}
}

func Defaults() (Config, string, error) {
	paths, err := App().DefaultPaths()
	if err != nil {
		return Config{}, "", err
	}
	whispDefaults, _, err := wconfig.Defaults()
	if err != nil {
		return Config{}, "", err
	}
	gmeetDefaults, _, err := gconfig.Defaults()
	if err != nil {
		return Config{}, "", err
	}
	exportDefaults, _, err := econfig.Defaults()
	if err != nil {
		return Config{}, "", err
	}
	cfg := Config{
		Version:       1,
		DBPath:        paths.DBPath,
		ReadsDBPath:   filepath.Join(filepath.Dir(paths.DBPath), "reads.db"),
		WhispcrawlDB:  whispDefaults.DBPath,
		GmeetcrawlDB:  gmeetDefaults.DBPath,
		ExportcrawlDB: exportDefaults.DBPath,
		Calendar: CalendarConfig{
			OAuthClientPath: gmeetDefaults.OAuthClientPath,
			TokenPath:       gmeetDefaults.TokenPath,
		},
	}
	return cfg, paths.ConfigPath, nil
}

func Load(configPath string) (Config, string, error) {
	cfg, defaultPath, err := Defaults()
	if err != nil {
		return Config{}, "", err
	}
	resolved, err := App().ResolveConfigPath(configPath)
	if err != nil {
		return Config{}, "", err
	}
	if resolved == "" {
		resolved = defaultPath
	}
	if _, err := os.Stat(resolved); err == nil {
		if err := ckconfig.LoadTOML(resolved, &cfg); err != nil {
			return Config{}, resolved, err
		}
	} else if !os.IsNotExist(err) {
		return Config{}, resolved, err
	}
	cfg.DBPath = ckconfig.ExpandHome(cfg.DBPath)
	cfg.ReadsDBPath = ckconfig.ExpandHome(cfg.ReadsDBPath)
	cfg.WhispcrawlDB = ckconfig.ExpandHome(cfg.WhispcrawlDB)
	cfg.GmeetcrawlDB = ckconfig.ExpandHome(cfg.GmeetcrawlDB)
	cfg.ExportcrawlDB = ckconfig.ExpandHome(cfg.ExportcrawlDB)
	cfg.Calendar.FixtureDir = ckconfig.ExpandHome(cfg.Calendar.FixtureDir)
	cfg.Calendar.OAuthClientPath = ckconfig.ExpandHome(cfg.Calendar.OAuthClientPath)
	cfg.Calendar.TokenPath = ckconfig.ExpandHome(cfg.Calendar.TokenPath)
	if cfg.DBPath == "" {
		paths, err := App().DefaultPaths()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.DBPath = paths.DBPath
	}
	if cfg.ReadsDBPath == "" {
		cfg.ReadsDBPath = filepath.Join(filepath.Dir(cfg.DBPath), "reads.db")
	}
	if cfg.WhispcrawlDB == "" {
		whispDefaults, _, err := wconfig.Defaults()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.WhispcrawlDB = whispDefaults.DBPath
	}
	if cfg.GmeetcrawlDB == "" {
		gmeetDefaults, _, err := gconfig.Defaults()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.GmeetcrawlDB = gmeetDefaults.DBPath
	}
	if cfg.ExportcrawlDB == "" {
		exportDefaults, _, err := econfig.Defaults()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.ExportcrawlDB = exportDefaults.DBPath
	}
	if cfg.Calendar.OAuthClientPath == "" {
		gmeetDefaults, _, err := gconfig.Defaults()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.Calendar.OAuthClientPath = gmeetDefaults.OAuthClientPath
	}
	if cfg.Calendar.TokenPath == "" {
		gmeetDefaults, _, err := gconfig.Defaults()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.Calendar.TokenPath = gmeetDefaults.TokenPath
	}
	return cfg, resolved, nil
}

func Save(path string, cfg Config) error {
	return ckconfig.WriteTOML(path, cfg, 0o600)
}

func EnsureDirs(cfg Config) error {
	for _, dir := range []string{filepath.Dir(cfg.DBPath), filepath.Dir(cfg.ReadsDBPath)} {
		if dir == "" || dir == "." {
			continue
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	return nil
}

func MarshalPreview(cfg Config) ([]byte, error) {
	return toml.Marshal(cfg)
}
