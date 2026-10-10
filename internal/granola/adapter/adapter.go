package adapter

import (
	"context"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/granola/config"
	"github.com/TheAngryPit/meetcrawl/internal/granola/sync"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Adapter struct {
	Config         gconfig.Config
	FixtureDir     string
	CrawlerVersion string
	Secrets        secret.Provider
}

func (a Adapter) Kind() source.Kind {
	return source.KindGranola
}

func (a Adapter) Sync(ctx context.Context) (source.SyncOutcome, error) {
	result, err := sync.Run(ctx, a.Config, sync.Options{
		FixtureDir:     a.FixtureDir,
		CrawlerVersion: a.CrawlerVersion,
		Secrets:        a.Secrets,
	})
	if sync.IsUnsupportedSchema(err) {
		return source.SyncOutcome{
			Code:   source.OutcomeUnsupportedSchema,
			Detail: result.Detail,
		}, err
	}
	if err != nil {
		return source.SyncOutcome{
			Code:   result.Code,
			Detail: result.Detail,
		}, err
	}
	return result.Outcome, nil
}
