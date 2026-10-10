package registry

import (
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/exportfile"
	"github.com/TheAngryPit/meetcrawl/internal/adapters/gmeet"
	"github.com/TheAngryPit/meetcrawl/internal/adapters/grain"
	"github.com/TheAngryPit/meetcrawl/internal/adapters/granola"
	"github.com/TheAngryPit/meetcrawl/internal/adapters/openwhispr"
	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	grainconfig "github.com/TheAngryPit/meetcrawl/internal/grain/config"
	granolaconfig "github.com/TheAngryPit/meetcrawl/internal/granola/config"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
)

// Name is the meet sync --source flag value for a registered adapter.
type Name string

func (n Name) String() string { return string(n) }

// SyncOptions are per-invocation flags for meet sync.
type SyncOptions struct {
	SourceDB   string
	FixtureDir string
	SourceDir  string
}

// Deps carries shared runtime dependencies for adapters.
type Deps struct {
	Config         ConfigView
	CrawlerVersion string
	Secrets        secret.Provider
}

type factory func(Deps, SyncOptions) source.Adapter

// Entry describes one registered source adapter.
type Entry struct {
	Name        Name
	Kind        source.Kind
	Scope       string
	ArchivePath func(ConfigView) string
	EnsureDirs  func(ConfigView) error
	NewAdapter  factory
}

var (
	byName   = map[Name]Entry{}
	regOrder []Name
)

// Register adds a source adapter. Shipped adapters register from init; tests may register synthetics.
func Register(e Entry) {
	if e.Name == "" || strings.TrimSpace(string(e.Kind)) == "" || e.NewAdapter == nil || e.ArchivePath == nil {
		panic("registry: invalid Entry")
	}
	if _, dup := byName[e.Name]; dup {
		panic(fmt.Sprintf("registry: duplicate name %q", e.Name))
	}
	for _, existing := range byName {
		if existing.Kind == e.Kind {
			panic(fmt.Sprintf("registry: duplicate kind %q", e.Kind))
		}
	}
	byName[e.Name] = e
	regOrder = append(regOrder, e.Name)
}

// Unregister removes a registration (tests only).
func Unregister(name Name) {
	delete(byName, name)
	filtered := regOrder[:0]
	for _, n := range regOrder {
		if n != name {
			filtered = append(filtered, n)
		}
	}
	regOrder = filtered
}

func init() {
	Register(Entry{
		Name:  "openwhispr",
		Kind:  source.KindOpenWhispr,
		Scope: "openwhispr SQLite archive",
		ArchivePath: func(c ConfigView) string {
			return c.OpenWhisprArchiveDB()
		},
		EnsureDirs: func(c ConfigView) error {
			return wconfig.EnsureDirs(c.WhispConfig())
		},
		NewAdapter: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.WhispConfig()
			return openwhispr.Adapter{
				Config:         cfg,
				SourceDB:       o.SourceDB,
				CrawlerVersion: d.CrawlerVersion,
			}
		},
	})
	Register(Entry{
		Name:  "gmeet",
		Kind:  source.KindGMeetGemini,
		Scope: "gmeet-gemini SQLite archive",
		ArchivePath: func(c ConfigView) string {
			return c.GMeetArchiveDB()
		},
		EnsureDirs: func(c ConfigView) error {
			return gconfig.EnsureDirs(c.GMeetConfig())
		},
		NewAdapter: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.GMeetConfig()
			return gmeet.Adapter{
				Config:         cfg,
				FixtureDir:     o.FixtureDir,
				CrawlerVersion: d.CrawlerVersion,
			}
		},
	})
	Register(Entry{
		Name:  "export-file",
		Kind:  source.KindExportFile,
		Scope: "export-file SQLite archive",
		ArchivePath: func(c ConfigView) string {
			return c.ExportFileArchiveDB()
		},
		EnsureDirs: func(c ConfigView) error {
			return econfig.EnsureDirs(c.ExportFileConfig())
		},
		NewAdapter: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.ExportFileConfig()
			return exportfile.Adapter{
				Config:         cfg,
				SourceDir:      o.SourceDir,
				FixtureDir:     o.FixtureDir,
				CrawlerVersion: d.CrawlerVersion,
			}
		},
	})
	Register(Entry{
		Name:  "grain",
		Kind:  source.KindGrain,
		Scope: "grain SQLite archive",
		ArchivePath: func(c ConfigView) string {
			return c.GrainArchiveDB()
		},
		EnsureDirs: func(c ConfigView) error {
			return grainconfig.EnsureDirs(c.GrainConfig())
		},
		NewAdapter: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.GrainConfig()
			return grain.Adapter{
				Config:         cfg,
				FixtureDir:     o.FixtureDir,
				CrawlerVersion: d.CrawlerVersion,
				Secrets:        d.Secrets,
			}
		},
	})
	Register(Entry{
		Name:  "granola",
		Kind:  source.KindGranola,
		Scope: "granola SQLite archive",
		ArchivePath: func(c ConfigView) string {
			return c.GranolaArchiveDB()
		},
		EnsureDirs: func(c ConfigView) error {
			return granolaconfig.EnsureDirs(c.GranolaConfig())
		},
		NewAdapter: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.GranolaConfig()
			return granola.Adapter{
				Config:         cfg,
				FixtureDir:     o.FixtureDir,
				CrawlerVersion: d.CrawlerVersion,
				Secrets:        d.Secrets,
			}
		},
	})
}

// Names returns registered adapter names in stable registration order.
func Names() []Name {
	out := make([]Name, len(regOrder))
	copy(out, regOrder)
	return out
}

// EntryFor returns one registration.
func EntryFor(name Name) (Entry, bool) {
	e, ok := byName[name]
	return e, ok
}

// ParseName resolves a --source flag value.
func ParseName(raw string) (Name, error) {
	n := Name(strings.TrimSpace(raw))
	if _, ok := byName[n]; ok {
		return n, nil
	}
	return "", fmt.Errorf("registry: unknown source %q", raw)
}

// RegisteredKind reports whether kind belongs to a registered adapter.
func RegisteredKind(kind source.Kind) bool {
	for _, e := range byName {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// LocalOnlyScopes returns crawlbar privacy.local_only_scopes entries.
func LocalOnlyScopes() []string {
	scopes := make([]string, 0, len(regOrder)+1)
	for _, name := range regOrder {
		scopes = append(scopes, byName[name].Scope)
	}
	scopes = append(scopes, "meet index SQLite archive")
	return scopes
}

// EnsureSourceArchives creates cache/log dirs for every registered adapter.
func EnsureSourceArchives(cfg ConfigView) error {
	for _, name := range regOrder {
		e := byName[name]
		if e.EnsureDirs == nil {
			continue
		}
		if err := e.EnsureDirs(cfg); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// SourceArchive is one registered adapter archive passed to meet index.
type SourceArchive struct {
	Kind source.Kind
	Path string
}

// IndexArchives builds the source list for meet index from registrations and optional path overrides.
func IndexArchives(cfg ConfigView, overrides map[Name]string) []SourceArchive {
	out := make([]SourceArchive, 0, len(regOrder))
	for _, name := range regOrder {
		e := byName[name]
		path := e.ArchivePath(cfg)
		if override, ok := overrides[name]; ok && strings.TrimSpace(override) != "" {
			path = strings.TrimSpace(override)
		}
		out = append(out, SourceArchive{Kind: e.Kind, Path: path})
	}
	return out
}

// NewAdapter constructs a source.Adapter for name.
func NewAdapter(name Name, deps Deps, opts SyncOptions) (source.Adapter, error) {
	e, ok := byName[name]
	if !ok {
		return nil, fmt.Errorf("registry: unknown source %q", name)
	}
	return e.NewAdapter(deps, opts), nil
}
