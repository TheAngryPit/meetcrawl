package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	ckconfig "github.com/openclaw/crawlkit/config"
	"github.com/pelletier/go-toml/v2"
)

const (
	AppName   = "whispcrawl"
	ConfigEnv = "WHISPCRAWL_CONFIG"
)

type Config struct {
	Version  int    `toml:"version" json:"version"`
	DBPath   string `toml:"db_path" json:"db_path"`
	CacheDir string `toml:"cache_dir" json:"cache_dir"`
	LogDir   string `toml:"log_dir" json:"log_dir"`
	SourceDB string `toml:"source_db,omitempty" json:"source_db,omitempty"`
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
	cfg := Config{
		Version:  1,
		DBPath:   paths.DBPath,
		CacheDir: paths.CacheDir,
		LogDir:   paths.LogDir,
		SourceDB: DefaultOpenWhisprDB(),
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
	cfg.CacheDir = ckconfig.ExpandHome(cfg.CacheDir)
	cfg.LogDir = ckconfig.ExpandHome(cfg.LogDir)
	cfg.SourceDB = ckconfig.ExpandHome(cfg.SourceDB)
	if cfg.SourceDB == "" {
		cfg.SourceDB = DefaultOpenWhisprDB()
	}
	if cfg.DBPath == "" {
		paths, err := App().DefaultPaths()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.DBPath = paths.DBPath
	}
	if cfg.CacheDir == "" {
		paths, err := App().DefaultPaths()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.CacheDir = paths.CacheDir
	}
	if cfg.LogDir == "" {
		paths, err := App().DefaultPaths()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.LogDir = paths.LogDir
	}
	return cfg, resolved, nil
}

func Save(path string, cfg Config) error {
	return ckconfig.WriteTOML(path, cfg, 0o600)
}

func EnsureDirs(cfg Config) error {
	for _, dir := range []string{filepath.Dir(cfg.DBPath), cfg.CacheDir, cfg.LogDir} {
		if dir == "" || dir == "." {
			continue
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	return nil
}

// DefaultOpenWhisprDB returns the first existing OpenWhispr database, or the
// platform's usual app-data location when none exists yet.
func DefaultOpenWhisprDB() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidates := openWhisprCandidates(runtime.GOOS, home, os.Getenv("XDG_CONFIG_HOME"))
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return candidates[0]
}

// openWhisprCandidates lists likely transcriptions.db paths, native location first.
func openWhisprCandidates(goos, home, xdgConfigHome string) []string {
	var dirs []string
	if goos == "darwin" {
		dirs = append(dirs, filepath.Join(home, "Library", "Application Support"))
	}
	if filepath.IsAbs(xdgConfigHome) {
		dirs = append(dirs, xdgConfigHome)
	}
	dirs = append(dirs, filepath.Join(home, ".config"))
	var out []string
	for _, dir := range dedupe(dirs) {
		out = append(out,
			filepath.Join(dir, "OpenWhispr", "transcriptions.db"),
			filepath.Join(dir, "openwhispr", "transcriptions.db"),
		)
	}
	return out
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func MarshalPreview(cfg Config) ([]byte, error) {
	return toml.Marshal(cfg)
}
