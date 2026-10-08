package adapter

import (
	"context"

	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/sync"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Adapter struct {
	Config         econfig.Config
	SourceDir      string
	FixtureDir     string
	CrawlerVersion string
}

func (a Adapter) Kind() source.Kind {
	return source.KindExportFile
}

func (a Adapter) Sync(ctx context.Context) (source.SyncOutcome, error) {
	result, err := sync.Run(ctx, a.Config, sync.Options{
		SourceDir:      a.SourceDir,
		FixtureDir:     a.FixtureDir,
		CrawlerVersion: a.CrawlerVersion,
	})
	if sync.IsUnsupportedSchema(err) {
		return source.SyncOutcome{
			Code:   source.OutcomeUnsupportedSchema,
			Detail: result.Detail,
		}, err
	}
	if err != nil {
		return source.SyncOutcome{
			Code:   source.OutcomeFailed,
			Detail: err.Error(),
		}, err
	}
	return result.Outcome, nil
}
