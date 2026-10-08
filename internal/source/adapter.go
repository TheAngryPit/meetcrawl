package source

import "context"

// Adapter is the single seam each source crawler implements (whispcrawl, gmeetcrawl, export-file, …).
// Commands own config, archive paths, and crawlkit capture; Sync returns identity, outcome, fidelity,
// privacy class, and provenance-ready artifacts without the index knowing provider details.
type Adapter interface {
	Kind() Kind
	Sync(ctx context.Context) (SyncOutcome, error)
}
