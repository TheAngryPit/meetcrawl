package registry_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/registry"
	"github.com/TheAngryPit/meetcrawl/internal/index/archive"
	"github.com/TheAngryPit/meetcrawl/internal/index/build"
	"github.com/TheAngryPit/meetcrawl/internal/index/reads"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	warchive "github.com/TheAngryPit/meetcrawl/internal/whisp/archive"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type syntheticAdapter struct {
	kind source.Kind
}

func (s syntheticAdapter) Kind() source.Kind { return s.kind }

func (s syntheticAdapter) Sync(context.Context) (source.SyncOutcome, error) {
	return source.SyncOutcome{Code: source.OutcomeOK}, nil
}

func TestRegisterSyntheticAdapter(t *testing.T) {
	const testName registry.Name = "synthetic-test-only"
	const testKind source.Kind = "synthetic-test-only"
	registry.Register(registry.Entry{
		Name:  testName,
		Kind:  testKind,
		Scope: "synthetic test archive",
		ArchivePath: func(c registry.ConfigView) string {
			return filepath.Join(filepath.Dir(c.OpenWhisprArchiveDB()), "synthetic-test.db")
		},
		EnsureDirs: func(registry.ConfigView) error { return nil },
		NewAdapter: func(_ registry.Deps, _ registry.SyncOptions) source.Adapter {
			return syntheticAdapter{kind: testKind}
		},
	})
	t.Cleanup(func() { registry.Unregister(testName) })

	cfg, _, err := mconfig.Defaults()
	if err != nil {
		t.Fatalf("Defaults() = %v", err)
	}
	adp, err := registry.NewAdapter(testName, registry.Deps{Config: mconfig.ConfigView(cfg), CrawlerVersion: "test"}, registry.SyncOptions{})
	if err != nil {
		t.Fatalf("NewAdapter() = %v", err)
	}
	if _, err := adp.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() = %v", err)
	}
	if _, err := registry.ParseName(string(testName)); err != nil {
		t.Fatalf("ParseName() = %v", err)
	}
	if !registry.RegisteredKind(testKind) {
		t.Fatal("RegisteredKind(synthetic) = false")
	}
}

func TestBuiltinAdaptersImplementContract(t *testing.T) {
	cfg, _, err := mconfig.Defaults()
	if err != nil {
		t.Fatalf("Defaults() = %v", err)
	}
	deps := registry.Deps{
		Config:         mconfig.ConfigView(cfg),
		CrawlerVersion: "test",
		Secrets:        secret.MapProvider{},
	}
	for _, name := range []registry.Name{"openwhispr", "gmeet", "export-file", "grain", "granola"} {
		name := name
		t.Run(string(name), func(t *testing.T) {
			adp, err := registry.NewAdapter(name, deps, registry.SyncOptions{})
			if err != nil {
				t.Fatalf("NewAdapter() = %v", err)
			}
			var _ source.Adapter = adp
		})
	}
}

func TestSyntheticAdapterIndexAndMCP(t *testing.T) {
	const (
		testName registry.Name = "registry-proof-synthetic"
		testKind source.Kind   = "registry-proof-synthetic"
		marker                 = "REGISTRY_SYNTHETIC_MARKER"
	)
	home := t.TempDir()
	buildHome, _ := os.UserHomeDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"GOMODCACHE", "GOPATH", "GOCACHE"} {
		if v := os.Getenv(key); v != "" {
			t.Setenv(key, v)
		} else if key == "GOMODCACHE" {
			t.Setenv(key, filepath.Join(buildHome, "go/pkg/mod"))
		} else if key == "GOPATH" {
			t.Setenv(key, filepath.Join(buildHome, "go"))
		}
	}
	t.Setenv("GOTELEMETRY", "off")

	cfg, configPath, err := mconfig.Defaults()
	if err != nil {
		t.Fatalf("Defaults() = %v", err)
	}
	cfg.DBPath = filepath.Join(home, "index", "meetcrawl.db")
	cfg.ReadsDBPath = filepath.Join(home, "index", "reads.db")
	syntheticDB := filepath.Join(home, "synthetic", "archive.db")

	emptyArchive(t, cfg.WhispcrawlDB)
	emptyArchive(t, cfg.GmeetcrawlDB)
	emptyArchive(t, cfg.ExportcrawlDB)
	emptyArchive(t, cfg.GraincrawlDB)
	emptyArchive(t, cfg.GranolacrawlDB)
	if err := seedSyntheticArchive(t, syntheticDB, testKind, marker); err != nil {
		t.Fatal(err)
	}

	registry.Register(registry.Entry{
		Name:        testName,
		Kind:        testKind,
		Scope:       "registry proof synthetic archive",
		ArchivePath: func(registry.ConfigView) string { return syntheticDB },
		EnsureDirs:  func(registry.ConfigView) error { return nil },
		NewAdapter: func(_ registry.Deps, _ registry.SyncOptions) source.Adapter {
			return syntheticAdapter{kind: testKind}
		},
	})
	t.Cleanup(func() { registry.Unregister(testName) })

	if err := mconfig.Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	if err := reads.EnsureSchema(context.Background(), cfg.ReadsDBPath); err != nil {
		t.Fatal(err)
	}

	regSources := registry.IndexArchives(mconfig.ConfigView(cfg), nil)
	buildSources := make([]build.SourceArchive, len(regSources))
	for i, s := range regSources {
		buildSources[i] = build.SourceArchive{Kind: s.Kind, Path: s.Path}
	}
	_, err = build.Run(context.Background(), build.Options{
		IndexDBPath: cfg.DBPath,
		Sources:     buildSources,
	})
	if err != nil {
		t.Fatalf("build.Run: %v", err)
	}

	ctx := context.Background()
	index, err := archive.OpenReadOnly(ctx, cfg.DBPath)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer index.Close()
	hits, err := index.Search(ctx, marker, 5, archive.NewRestrictedAllowlist(nil))
	if err != nil {
		t.Fatalf("index search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("index ignored synthetic adapter kind (no hits)")
	}

	repoRoot := moduleRoot(t)
	meetBin := filepath.Join(home, "meet")
	buildCmd := exec.Command("go", "build", "-o", meetBin, "./cmd/meet")
	buildCmd.Dir = repoRoot
	buildCmd.Env = append(os.Environ(), "HOME="+buildHome, "GOTELEMETRY=off")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build meet: %v\n%s", err, out)
	}
	cmd := exec.Command(meetBin, "--config", configPath, "mcp")
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "registry-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("mcp connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "search_meetings",
		Arguments: map[string]any{"query": marker, "limit": 5},
	})
	if err != nil {
		t.Fatalf("mcp search_meetings: %v", err)
	}
	text := toolText(res)
	if !strings.Contains(text, marker) {
		t.Fatalf("mcp ignored synthetic adapter: %q", text)
	}
}

func emptyArchive(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := warchive.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceArtifacts(context.Background(), nil, "test", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
}

func seedSyntheticArchive(t *testing.T, path string, kind source.Kind, marker string) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	store, err := warchive.Open(context.Background(), path)
	if err != nil {
		return err
	}
	defer store.Close()
	start := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	artifact := source.Artifact{
		Identity: source.Identity{
			Kind:     kind,
			SourceID: "registry-synthetic-001",
		},
		SourceRevision: "rev-synthetic",
		Fidelity:       source.FidelityNotes,
		Privacy:        source.PrivacyPrivate,
		NormalizedText: marker + " synthetic adapter proof.",
		Window:         source.Window{Start: start, End: start.Add(time.Hour)},
		Language:       "en",
	}
	if err := artifact.Validate(); err != nil {
		return err
	}
	return store.ReplaceArtifacts(context.Background(), []warchive.StoredArtifact{{Artifact: artifact}}, "registry-test", time.Now().UTC())
}

func toolText(res *sdkmcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*sdkmcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
