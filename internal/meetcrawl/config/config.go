package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/registry"
	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	grainconfig "github.com/TheAngryPit/meetcrawl/internal/grain/config"
	granolaconfig "github.com/TheAngryPit/meetcrawl/internal/granola/config"
	"github.com/TheAngryPit/meetcrawl/internal/index/privacy"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
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

// OpenWhisprSection holds OpenWhispr-specific settings; archive path stays whispcrawl_db.
type OpenWhisprSection struct {
	SourceDB string `toml:"source_db,omitempty" json:"source_db,omitempty"`
	CacheDir string `toml:"cache_dir,omitempty" json:"cache_dir,omitempty"`
	LogDir   string `toml:"log_dir,omitempty" json:"log_dir,omitempty"`
}

// GMeetSection holds Google Meet / Gemini ingest settings; archive path stays gmeetcrawl_db.
type GMeetSection struct {
	CacheDir        string   `toml:"cache_dir,omitempty" json:"cache_dir,omitempty"`
	LogDir          string   `toml:"log_dir,omitempty" json:"log_dir,omitempty"`
	MeetFolderRoots []string `toml:"meet_folder_roots,omitempty" json:"meet_folder_roots,omitempty"`
	OAuthClientPath string   `toml:"oauth_client_path,omitempty" json:"oauth_client_path,omitempty"`
	TokenPath       string   `toml:"token_path,omitempty" json:"token_path,omitempty"`
}

// ExportFileSection holds export-file ingest settings; archive path stays exportcrawl_db.
type ExportFileSection struct {
	SourceDir    string `toml:"source_dir,omitempty" json:"source_dir,omitempty"`
	CacheDir     string `toml:"cache_dir,omitempty" json:"cache_dir,omitempty"`
	LogDir       string `toml:"log_dir,omitempty" json:"log_dir,omitempty"`
	PrivacyClass string `toml:"privacy_class,omitempty" json:"privacy_class,omitempty"`
}

// SecretsSection selects where adapter API keys are resolved (never stored in repo or archives).
type SecretsSection struct {
	Provider  string `toml:"provider,omitempty" json:"provider,omitempty"`
	OpCommand string `toml:"op_command,omitempty" json:"op_command,omitempty"`
}

// GrainSection holds Grain API ingest settings; archive path stays graincrawl_db.
type GrainSection struct {
	CacheDir string `toml:"cache_dir,omitempty" json:"cache_dir,omitempty"`
	LogDir   string `toml:"log_dir,omitempty" json:"log_dir,omitempty"`
	PATRef   string `toml:"pat_ref,omitempty" json:"pat_ref,omitempty"`
}

// GranolaSection holds Granola REST API ingest settings; archive path stays granolacrawl_db.
type GranolaSection struct {
	CacheDir  string `toml:"cache_dir,omitempty" json:"cache_dir,omitempty"`
	LogDir    string `toml:"log_dir,omitempty" json:"log_dir,omitempty"`
	APIKeyRef string `toml:"api_key_ref,omitempty" json:"api_key_ref,omitempty"`
}

type Config struct {
	Version        int               `toml:"version" json:"version"`
	DBPath         string            `toml:"db_path" json:"db_path"`
	ReadsDBPath    string            `toml:"reads_db_path" json:"reads_db_path"`
	WhispcrawlDB   string            `toml:"whispcrawl_db" json:"whispcrawl_db"`
	GmeetcrawlDB   string            `toml:"gmeetcrawl_db" json:"gmeetcrawl_db"`
	ExportcrawlDB  string            `toml:"exportcrawl_db" json:"exportcrawl_db"`
	GraincrawlDB   string            `toml:"graincrawl_db" json:"graincrawl_db"`
	GranolacrawlDB string            `toml:"granolacrawl_db" json:"granolacrawl_db"`
	OpenWhispr     OpenWhisprSection `toml:"openwhispr" json:"openwhispr"`
	GMeet          GMeetSection      `toml:"gmeet" json:"gmeet"`
	ExportFile     ExportFileSection `toml:"export_file" json:"export_file"`
	Grain          GrainSection      `toml:"grain" json:"grain"`
	Granola        GranolaSection    `toml:"granola" json:"granola"`
	Secrets        SecretsSection    `toml:"secrets" json:"secrets"`
	Privacy        privacy.Config    `toml:"privacy" json:"privacy"`
	Calendar       CalendarConfig    `toml:"calendar" json:"calendar"`
	MCP            MCPConfig         `toml:"mcp" json:"mcp"`
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
	grainDefaults, _, err := grainconfig.Defaults()
	if err != nil {
		return Config{}, "", err
	}
	granolaDefaults, _, err := granolaconfig.Defaults()
	if err != nil {
		return Config{}, "", err
	}
	cfg := Config{
		Version:        1,
		DBPath:         paths.DBPath,
		ReadsDBPath:    filepath.Join(filepath.Dir(paths.DBPath), "reads.db"),
		WhispcrawlDB:   whispDefaults.DBPath,
		GmeetcrawlDB:   gmeetDefaults.DBPath,
		ExportcrawlDB:  exportDefaults.DBPath,
		GraincrawlDB:   grainDefaults.DBPath,
		GranolacrawlDB: granolaDefaults.DBPath,
		OpenWhispr: OpenWhisprSection{
			SourceDB: whispDefaults.SourceDB,
			CacheDir: whispDefaults.CacheDir,
			LogDir:   whispDefaults.LogDir,
		},
		GMeet: GMeetSection{
			CacheDir:        gmeetDefaults.CacheDir,
			LogDir:          gmeetDefaults.LogDir,
			MeetFolderRoots: append([]string(nil), gmeetDefaults.MeetFolderRoots...),
			OAuthClientPath: gmeetDefaults.OAuthClientPath,
			TokenPath:       gmeetDefaults.TokenPath,
		},
		ExportFile: ExportFileSection{
			CacheDir:     exportDefaults.CacheDir,
			LogDir:       exportDefaults.LogDir,
			PrivacyClass: exportDefaults.PrivacyClass,
		},
		Grain: GrainSection{
			CacheDir: grainDefaults.CacheDir,
			LogDir:   grainDefaults.LogDir,
			PATRef:   secret.AccountGrainPAT,
		},
		Granola: GranolaSection{
			CacheDir:  granolaDefaults.CacheDir,
			LogDir:    granolaDefaults.LogDir,
			APIKeyRef: secret.AccountGranolaAPIKey,
		},
		Secrets: SecretsSection{
			Provider: secret.ProviderKeychain,
		},
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
	cfg.GraincrawlDB = ckconfig.ExpandHome(cfg.GraincrawlDB)
	cfg.GranolacrawlDB = ckconfig.ExpandHome(cfg.GranolacrawlDB)
	cfg.OpenWhispr.SourceDB = ckconfig.ExpandHome(cfg.OpenWhispr.SourceDB)
	cfg.OpenWhispr.CacheDir = ckconfig.ExpandHome(cfg.OpenWhispr.CacheDir)
	cfg.OpenWhispr.LogDir = ckconfig.ExpandHome(cfg.OpenWhispr.LogDir)
	cfg.GMeet.CacheDir = ckconfig.ExpandHome(cfg.GMeet.CacheDir)
	cfg.GMeet.LogDir = ckconfig.ExpandHome(cfg.GMeet.LogDir)
	cfg.GMeet.OAuthClientPath = ckconfig.ExpandHome(cfg.GMeet.OAuthClientPath)
	cfg.GMeet.TokenPath = ckconfig.ExpandHome(cfg.GMeet.TokenPath)
	cfg.ExportFile.SourceDir = ckconfig.ExpandHome(cfg.ExportFile.SourceDir)
	cfg.ExportFile.CacheDir = ckconfig.ExpandHome(cfg.ExportFile.CacheDir)
	cfg.ExportFile.LogDir = ckconfig.ExpandHome(cfg.ExportFile.LogDir)
	cfg.Grain.CacheDir = ckconfig.ExpandHome(cfg.Grain.CacheDir)
	cfg.Grain.LogDir = ckconfig.ExpandHome(cfg.Grain.LogDir)
	cfg.Granola.CacheDir = ckconfig.ExpandHome(cfg.Granola.CacheDir)
	cfg.Granola.LogDir = ckconfig.ExpandHome(cfg.Granola.LogDir)
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
	if cfg.GraincrawlDB == "" {
		grainDefaults, _, err := grainconfig.Defaults()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.GraincrawlDB = grainDefaults.DBPath
	}
	if cfg.GranolacrawlDB == "" {
		granolaDefaults, _, err := granolaconfig.Defaults()
		if err != nil {
			return Config{}, resolved, err
		}
		cfg.GranolacrawlDB = granolaDefaults.DBPath
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
	cfg = fillSourceSections(cfg)
	return cfg, resolved, nil
}

func fillSourceSections(cfg Config) Config {
	whispDefaults, _, _ := wconfig.Defaults()
	gmeetDefaults, _, _ := gconfig.Defaults()
	exportDefaults, _, _ := econfig.Defaults()
	grainDefaults, _, _ := grainconfig.Defaults()
	granolaDefaults, _, _ := granolaconfig.Defaults()
	if cfg.OpenWhispr.SourceDB == "" {
		cfg.OpenWhispr.SourceDB = whispDefaults.SourceDB
	}
	if cfg.OpenWhispr.CacheDir == "" {
		cfg.OpenWhispr.CacheDir = whispDefaults.CacheDir
	}
	if cfg.OpenWhispr.LogDir == "" {
		cfg.OpenWhispr.LogDir = whispDefaults.LogDir
	}
	if cfg.GMeet.CacheDir == "" {
		cfg.GMeet.CacheDir = gmeetDefaults.CacheDir
	}
	if cfg.GMeet.LogDir == "" {
		cfg.GMeet.LogDir = gmeetDefaults.LogDir
	}
	if len(cfg.GMeet.MeetFolderRoots) == 0 {
		cfg.GMeet.MeetFolderRoots = append([]string(nil), gmeetDefaults.MeetFolderRoots...)
	}
	if cfg.GMeet.OAuthClientPath == "" {
		cfg.GMeet.OAuthClientPath = gmeetDefaults.OAuthClientPath
	}
	if cfg.GMeet.TokenPath == "" {
		cfg.GMeet.TokenPath = gmeetDefaults.TokenPath
	}
	if cfg.ExportFile.CacheDir == "" {
		cfg.ExportFile.CacheDir = exportDefaults.CacheDir
	}
	if cfg.ExportFile.LogDir == "" {
		cfg.ExportFile.LogDir = exportDefaults.LogDir
	}
	if cfg.ExportFile.PrivacyClass == "" {
		cfg.ExportFile.PrivacyClass = exportDefaults.PrivacyClass
	}
	if cfg.Grain.CacheDir == "" {
		cfg.Grain.CacheDir = grainDefaults.CacheDir
	}
	if cfg.Grain.LogDir == "" {
		cfg.Grain.LogDir = grainDefaults.LogDir
	}
	if strings.TrimSpace(cfg.Grain.PATRef) == "" {
		cfg.Grain.PATRef = secret.AccountGrainPAT
	}
	if cfg.Granola.CacheDir == "" {
		cfg.Granola.CacheDir = granolaDefaults.CacheDir
	}
	if cfg.Granola.LogDir == "" {
		cfg.Granola.LogDir = granolaDefaults.LogDir
	}
	if strings.TrimSpace(cfg.Granola.APIKeyRef) == "" {
		cfg.Granola.APIKeyRef = secret.AccountGranolaAPIKey
	}
	if strings.TrimSpace(cfg.Secrets.Provider) == "" {
		cfg.Secrets.Provider = secret.ProviderKeychain
	}
	return cfg
}

// WhispConfig projects the unified config into whispcrawl archive settings.
func (c Config) WhispConfig() wconfig.Config {
	return wconfig.Config{
		Version:  c.Version,
		DBPath:   c.WhispcrawlDB,
		CacheDir: c.OpenWhispr.CacheDir,
		LogDir:   c.OpenWhispr.LogDir,
		SourceDB: c.OpenWhispr.SourceDB,
	}
}

// GMeetConfig projects the unified config into gmeetcrawl archive settings.
func (c Config) GMeetConfig() gconfig.Config {
	return gconfig.Config{
		Version:         c.Version,
		DBPath:          c.GmeetcrawlDB,
		CacheDir:        c.GMeet.CacheDir,
		LogDir:          c.GMeet.LogDir,
		MeetFolderRoots: append([]string(nil), c.GMeet.MeetFolderRoots...),
		OAuthClientPath: c.GMeet.OAuthClientPath,
		TokenPath:       c.GMeet.TokenPath,
	}
}

// ExportFileConfig projects the unified config into exportcrawl archive settings.
func (c Config) ExportFileConfig() econfig.Config {
	return econfig.Config{
		Version:      c.Version,
		DBPath:       c.ExportcrawlDB,
		CacheDir:     c.ExportFile.CacheDir,
		LogDir:       c.ExportFile.LogDir,
		SourceDir:    c.ExportFile.SourceDir,
		PrivacyClass: c.ExportFile.PrivacyClass,
	}
}

func (c Config) GrainConfig() grainconfig.Config {
	return grainconfig.Config{
		Version:         c.Version,
		DBPath:          c.GraincrawlDB,
		CacheDir:        c.Grain.CacheDir,
		LogDir:          c.Grain.LogDir,
		PATRef:          c.Grain.PATRef,
		SecretsProvider: c.Secrets.Provider,
	}
}

func (c Config) GranolaConfig() granolaconfig.Config {
	return granolaconfig.Config{
		Version:         c.Version,
		DBPath:          c.GranolacrawlDB,
		CacheDir:        c.Granola.CacheDir,
		LogDir:          c.Granola.LogDir,
		APIKeyRef:       c.Granola.APIKeyRef,
		SecretsProvider: c.Secrets.Provider,
	}
}

func (c Config) SecretProvider() (secret.Provider, error) {
	return secret.NewProvider(secret.Options{
		Provider:  c.Secrets.Provider,
		OpCommand: c.Secrets.OpCommand,
	})
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
	return EnsureSourceDirs(cfg)
}

// EnsureSourceDirs creates cache/log dirs for each registered source adapter.
func EnsureSourceDirs(cfg Config) error {
	return registry.EnsureSourceArchives(ConfigView(cfg))
}

// ConfigView adapts Config for the adapter registry without an import cycle.
type ConfigView Config

func (c ConfigView) OpenWhisprArchiveDB() string { return c.WhispcrawlDB }
func (c ConfigView) GMeetArchiveDB() string      { return c.GmeetcrawlDB }
func (c ConfigView) ExportFileArchiveDB() string { return c.ExportcrawlDB }
func (c ConfigView) GrainArchiveDB() string      { return c.GraincrawlDB }
func (c ConfigView) GranolaArchiveDB() string    { return c.GranolacrawlDB }
func (c ConfigView) WhispConfig() wconfig.Config { return Config(c).WhispConfig() }
func (c ConfigView) GMeetConfig() gconfig.Config { return Config(c).GMeetConfig() }
func (c ConfigView) ExportFileConfig() econfig.Config {
	return Config(c).ExportFileConfig()
}
func (c ConfigView) GrainConfig() grainconfig.Config {
	return Config(c).GrainConfig()
}
func (c ConfigView) GranolaConfig() granolaconfig.Config {
	return Config(c).GranolaConfig()
}

func MarshalPreview(cfg Config) ([]byte, error) {
	return toml.Marshal(cfg)
}
