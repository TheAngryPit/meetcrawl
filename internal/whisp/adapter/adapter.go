package adapter

import (
	"context"

	"github.com/TheAngryPit/meetcrawl/internal/source"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/sync"
)

type Adapter struct {
	Config         wconfig.Config
	SourceDB       string
	CrawlerVersion string
}

func (a Adapter) Kind() source.Kind {
	return source.KindOpenWhispr
}

func (a Adapter) Sync(ctx context.Context) (source.SyncOutcome, error) {
	result, err := sync.Run(ctx, a.Config, sync.Options{
		SourceDB:       a.SourceDB,
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
