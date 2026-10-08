package reads

import (
	"context"

	"github.com/openclaw/crawlkit/state"
	ckstore "github.com/openclaw/crawlkit/store"
)

// EnsureSchema creates reads.db with the append-only read_log table when missing.
func EnsureSchema(ctx context.Context, path string) error {
	store, err := ckstore.Open(ctx, ckstore.Options{
		Path:          path,
		Schema:        Schema,
		SchemaVersion: SchemaVersion,
		MaxOpenConns:  1,
		MaxIdleConns:  1,
	})
	if err != nil {
		return err
	}
	defer store.Close()
	return state.EnsureSchema(ctx, store.DB())
}
