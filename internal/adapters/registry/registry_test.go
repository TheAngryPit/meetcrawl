package registry_test

import (
	"context"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/registry"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type syntheticAdapter struct {
	kind source.Kind
}

func (s syntheticAdapter) Kind() source.Kind { return s.kind }

func (s syntheticAdapter) Sync(context.Context) (source.SyncOutcome, error) {
	return source.SyncOutcome{Code: source.OutcomeOK}, nil
}

func TestRegisterSyntheticAdapter(t *testing.T) {
	t.Parallel()
	const testName registry.Name = "synthetic-test-only"
	registry.Register(testName, func(_ registry.Deps, _ registry.SyncOptions) source.Adapter {
		return syntheticAdapter{kind: source.Kind("synthetic-test-only")}
	})
	t.Cleanup(func() { registry.Unregister(testName) })

	cfg, _, err := mconfig.Defaults()
	if err != nil {
		t.Fatalf("Defaults() = %v", err)
	}
	adp, err := registry.NewAdapter(testName, registry.Deps{Config: cfg, CrawlerVersion: "test"}, registry.SyncOptions{})
	if err != nil {
		t.Fatalf("NewAdapter() = %v", err)
	}
	if _, err := adp.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() = %v", err)
	}
	if _, err := registry.ParseName(string(testName)); err != nil {
		t.Fatalf("ParseName() = %v", err)
	}
}

func TestBuiltinAdaptersImplementContract(t *testing.T) {
	t.Parallel()
	cfg, _, err := mconfig.Defaults()
	if err != nil {
		t.Fatalf("Defaults() = %v", err)
	}
	deps := registry.Deps{Config: cfg, CrawlerVersion: "test"}
	for _, name := range registry.BuiltinNames() {
		name := name
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()
			adp, err := registry.NewAdapter(name, deps, registry.SyncOptions{})
			if err != nil {
				t.Fatalf("NewAdapter() = %v", err)
			}
			var _ source.Adapter = adp
		})
	}
}
