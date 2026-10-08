package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/sync"
	"github.com/TheAngryPit/meetcrawl/internal/index/build"
	"github.com/TheAngryPit/meetcrawl/internal/index/privacy"
	"github.com/TheAngryPit/meetcrawl/internal/index/reads"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	wsync "github.com/TheAngryPit/meetcrawl/internal/whisp/sync"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/testutil"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pelletier/go-toml/v2"
)

const runAsMCPServer = "MEETCRAWL_RUN_MCP_SERVER"

func TestMain(m *testing.M) {
	if os.Getenv(runAsMCPServer) != "" {
		cfgPath := os.Getenv("MEETCRAWL_TEST_CONFIG")
		if cfgPath == "" {
			os.Exit(2)
		}
		if err := Run(context.Background(), Options{ConfigPath: cfgPath}); err != nil {
			os.Exit(1)
		}
		return
	}
	os.Exit(m.Run())
}

func TestStdioMCPSearchAndReadLog(t *testing.T) {
	if testing.Short() {
		t.Skip("stdio MCP subprocess test")
	}
	cfgPath, readsPath, restrictedID := setupIndexedFixtures(t)
	before, err := reads.CountRows(context.Background(), readsPath)
	if err != nil {
		t.Fatalf("count read_log before: %v", err)
	}

	cmd := mcpServerCommand(t, cfgPath)
	ctx := context.Background()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "meetcrawl-test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"search_meetings", "get_meeting", "list_meetings"} {
		if !names[want] {
			t.Fatalf("tools/list missing %q; got %v", want, names)
		}
	}

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "search_meetings",
		Arguments: map[string]any{"query": "reuniao", "limit": 10},
	})
	if err != nil {
		t.Fatalf("search_meetings: %v", err)
	}
	text := textFromResult(t, res)
	if !strings.HasPrefix(text, UntrustedPrefix) {
		t.Fatalf("missing untrusted prefix in:\n%s", text)
	}

	hidden, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "search_meetings",
		Arguments: map[string]any{"query": "Synthetic standup transcript", "limit": 10},
	})
	if err != nil {
		t.Fatalf("search restricted: %v", err)
	}
	hiddenText := textFromResult(t, hidden)
	if strings.Contains(hiddenText, restrictedID) {
		t.Fatalf("restricted meeting leaked in search:\n%s", hiddenText)
	}

	after, err := reads.CountRows(context.Background(), readsPath)
	if err != nil {
		t.Fatalf("count read_log after: %v", err)
	}
	if after-before != 2 {
		t.Fatalf("read_log rows added = %d, want 2 (one per search_meetings call)", after-before)
	}
}

func textFromResult(t *testing.T, res *sdkmcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty tool content")
	}
	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("content type %T, want TextContent", res.Content[0])
	}
	return tc.Text
}

func mcpServerCommand(t *testing.T, configPath string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$", "-test.v")
	cmd.Env = append(os.Environ(),
		runAsMCPServer+"=1",
		"MEETCRAWL_TEST_CONFIG="+configPath,
	)
	return cmd
}

func setupIndexedFixtures(t *testing.T) (configPath, readsPath, restrictedMeetingID string) {
	t.Helper()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)

	whispDB := filepath.Join(root, "whispcrawl.db")
	sourceDB := filepath.Join(root, "openwhispr", "transcriptions.db")
	if err := testutil.WriteSupportedDB(sourceDB); err != nil {
		t.Fatalf("WriteSupportedDB: %v", err)
	}
	whispCfg := wconfig.Config{
		Version:  1,
		DBPath:   whispDB,
		CacheDir: filepath.Join(root, "whisp-cache"),
		LogDir:   filepath.Join(root, "whisp-logs"),
	}
	if _, err := wsync.Run(context.Background(), whispCfg, wsync.Options{
		SourceDB:       sourceDB,
		CrawlerVersion: "whispcrawl-test",
	}); err != nil {
		t.Fatalf("whisp sync: %v", err)
	}

	gmeetDB := filepath.Join(root, "gmeetcrawl.db")
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "gdrive", "supported")
	gmeetCfg := gconfig.Config{
		Version:  1,
		DBPath:   gmeetDB,
		CacheDir: filepath.Join(root, "gmeet-cache"),
		LogDir:   filepath.Join(root, "gmeet-logs"),
	}
	if _, err := sync.Run(context.Background(), gmeetCfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "gmeetcrawl-test",
	}); err != nil {
		t.Fatalf("gmeet sync: %v", err)
	}

	indexDB := filepath.Join(root, "meetcrawl.db")
	readsPath = filepath.Join(root, "reads.db")
	if err := reads.EnsureSchema(context.Background(), readsPath); err != nil {
		t.Fatalf("reads schema: %v", err)
	}
	if _, err := build.Run(context.Background(), build.Options{
		IndexDBPath: indexDB,
		Privacy: privacy.Config{
			Rules: []privacy.Rule{
				{CalendarICal: "cal-synthetic-001", Class: "restricted"},
			},
		},
		Sources: []build.SourceArchive{
			{Kind: source.KindOpenWhispr, Path: whispDB},
			{Kind: source.KindGMeetGemini, Path: gmeetDB},
		},
	}); err != nil {
		t.Fatalf("index: %v", err)
	}

	restrictedMeetingID = calendarMeetingID(t, "cal-synthetic-001", "2026-01-15T14:00:00Z")

	cfg := mconfig.Config{
		Version:      1,
		DBPath:       indexDB,
		ReadsDBPath:  readsPath,
		WhispcrawlDB: whispDB,
		GmeetcrawlDB: gmeetDB,
	}
	configPath = filepath.Join(root, "config.toml")
	body, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath, readsPath, restrictedMeetingID
}

func calendarMeetingID(t *testing.T, icalUID, start string) string {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, start)
	if err != nil {
		t.Fatal(err)
	}
	payload := icalUID + "|" + ts.UTC().Format(time.RFC3339)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
