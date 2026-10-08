package runtime

import (
	"context"

	"github.com/TheAngryPit/meetcrawl/internal/index/archive"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
)

type Runtime struct {
	ConfigPath string
	Config     mconfig.Config
	Store      *archive.Store
}

func Open(ctx context.Context, configPath string) (*Runtime, error) {
	cfg, resolved, err := mconfig.Load(configPath)
	if err != nil {
		return nil, err
	}
	store, err := archive.Open(ctx, cfg.DBPath)
	if err != nil {
		return nil, err
	}
	return &Runtime{ConfigPath: resolved, Config: cfg, Store: store}, nil
}

func (rt *Runtime) Close() error {
	if rt == nil || rt.Store == nil {
		return nil
	}
	return rt.Store.Close()
}
