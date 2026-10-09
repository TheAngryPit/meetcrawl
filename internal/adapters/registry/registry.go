package registry

import (
	"fmt"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/exportfile"
	"github.com/TheAngryPit/meetcrawl/internal/adapters/gmeet"
	"github.com/TheAngryPit/meetcrawl/internal/adapters/openwhispr"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

// Name identifies a registered source adapter CLI flag value.
type Name string

const (
	NameOpenWhispr Name = "openwhispr"
	NameGMeet      Name = "gmeet"
	NameExportFile Name = "export-file"
)

func (n Name) String() string { return string(n) }

// SyncOptions are per-invocation flags for meet sync.
type SyncOptions struct {
	SourceDB   string
	FixtureDir string
	SourceDir  string
}

// Deps carries shared runtime dependencies for adapters.
type Deps struct {
	Config         mconfig.Config
	CrawlerVersion string
}

type factory func(Deps, SyncOptions) source.Adapter

var (
	builtin = map[Name]factory{
		NameOpenWhispr: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.WhispConfig()
			return openwhispr.Adapter{
				Config:         cfg,
				SourceDB:       o.SourceDB,
				CrawlerVersion: d.CrawlerVersion,
			}
		},
		NameGMeet: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.GMeetConfig()
			return gmeet.Adapter{
				Config:         cfg,
				FixtureDir:     o.FixtureDir,
				CrawlerVersion: d.CrawlerVersion,
			}
		},
		NameExportFile: func(d Deps, o SyncOptions) source.Adapter {
			cfg := d.Config.ExportFileConfig()
			return exportfile.Adapter{
				Config:         cfg,
				SourceDir:      o.SourceDir,
				FixtureDir:     o.FixtureDir,
				CrawlerVersion: d.CrawlerVersion,
			}
		},
	}
	extra = map[Name]factory{}
)

// Register adds a source adapter for tests or optional plugins. Not used by shipped commands.
func Register(name Name, f factory) {
	if f == nil {
		panic("registry: nil factory")
	}
	extra[name] = f
}

// Unregister removes a test-only registration.
func Unregister(name Name) {
	delete(extra, name)
}

// ParseName normalizes a --source flag value.
func ParseName(raw string) (Name, error) {
	n := Name(strings.TrimSpace(raw))
	switch n {
	case NameOpenWhispr, NameGMeet, NameExportFile:
		return n, nil
	default:
		if _, ok := extra[n]; ok {
			return n, nil
		}
		return "", fmt.Errorf("registry: unknown source %q (want openwhispr, gmeet, or export-file)", raw)
	}
}

// BuiltinNames returns shipped adapter ids in stable order.
func BuiltinNames() []Name {
	return []Name{NameOpenWhispr, NameGMeet, NameExportFile}
}

// NewAdapter constructs a source.Adapter for name.
func NewAdapter(name Name, deps Deps, opts SyncOptions) (source.Adapter, error) {
	if f, ok := extra[name]; ok {
		return f(deps, opts), nil
	}
	f, ok := builtin[name]
	if !ok {
		return nil, fmt.Errorf("registry: unknown source %q", name)
	}
	return f(deps, opts), nil
}
