package runtime

import (
	"context"

	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/archive"
	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
)

type Runtime struct {
	Config     econfig.Config
	ConfigPath string
	Store      *archive.Store
}

func Open(ctx context.Context, configPath string) (Runtime, error) {
	cfg, resolved, err := econfig.Load(configPath)
	if err != nil {
		return Runtime{}, err
	}
	store, err := archive.Open(ctx, cfg.DBPath)
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{Config: cfg, ConfigPath: resolved, Store: store}, nil
}

func (r Runtime) Close() error {
	if r.Store == nil {
		return nil
	}
	return r.Store.Close()
}
