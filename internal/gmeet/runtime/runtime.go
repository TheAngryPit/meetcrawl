package runtime

import (
	"context"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/archive"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
)

type Runtime struct {
	Config     wconfig.Config
	ConfigPath string
	Store      *archive.Store
}

func Open(ctx context.Context, configPath string) (Runtime, error) {
	cfg, resolved, err := wconfig.Load(configPath)
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
