package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/meet/cli"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/testutil"
	"github.com/zalando/go-keyring"
)

type syncAllResult struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error"`
	Result struct {
		Sources []struct {
			Source    string `json:"source"`
			Status    string `json:"status"`
			Artifacts int    `json:"artifacts"`
			Reason    string `json:"reason"`
		} `json:"sources"`
	} `json:"result"`
}

func syncAllHome(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "MEETCRAWL_CONFIG"} {
		t.Setenv(key, "")
	}
	app := cli.App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init: %v", err)
	}
}

func runSyncAll(t *testing.T, args ...string) (syncAllResult, error) {
	t.Helper()
	var out bytes.Buffer
	app := cli.App{Stdout: &out, Stderr: &bytes.Buffer{}}
	err := app.Run(context.Background(), append([]string{"--json", "sync", "--source", "all"}, args...))
	var res syncAllResult
	if jsonErr := json.Unmarshal(out.Bytes(), &res); jsonErr != nil {
		t.Fatalf("sync --source all output is not JSON: %v\n%s", jsonErr, out.String())
	}
	return res, err
}

func statuses(res syncAllResult) map[string]string {
	out := map[string]string{}
	for _, s := range res.Result.Sources {
		out[s.Source] = s.Status
	}
	return out
}

func TestSyncAllSkipsEverySourceThatIsNotSetUp(t *testing.T) {
	syncAllHome(t)
	res, err := runSyncAll(t)
	if err != nil {
		t.Fatalf("sync --source all with nothing set up = %v, want success", err)
	}
	if len(res.Result.Sources) == 0 {
		t.Fatal("no sources reported")
	}
	for name, status := range statuses(res) {
		if status != "skipped" {
			t.Fatalf("source %s status = %q, want skipped", name, status)
		}
	}
	err = filepath.WalkDir(os.Getenv("HOME"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && d.Name() == "whispcrawl" {
			t.Fatalf("a skipped source got its directory created: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSyncAllRunsOnlyConfiguredSources(t *testing.T) {
	syncAllHome(t)
	if err := testutil.WriteSupportedDB(wconfig.DefaultOpenWhisprDB()); err != nil {
		t.Fatal(err)
	}
	res, err := runSyncAll(t)
	if err != nil {
		t.Fatalf("sync --source all = %v", err)
	}
	got := statuses(res)
	if got["openwhispr"] != "ok" {
		t.Fatalf("openwhispr status = %q, want ok; all: %v", got["openwhispr"], got)
	}
	for _, s := range res.Result.Sources {
		if s.Source == "openwhispr" && s.Artifacts != 3 {
			t.Fatalf("openwhispr artifacts = %d, want 3", s.Artifacts)
		}
		if s.Source != "openwhispr" && s.Status != "skipped" {
			t.Fatalf("source %s status = %q, want skipped", s.Source, s.Status)
		}
	}
}

func TestSyncAllFailsWhenASourceThatIsSetUpFails(t *testing.T) {
	syncAllHome(t)
	if err := testutil.WriteUnsupportedDB(wconfig.DefaultOpenWhisprDB()); err != nil {
		t.Fatal(err)
	}
	res, err := runSyncAll(t)
	if err == nil {
		t.Fatal("sync --source all = nil, want an error when a configured source fails")
	}
	if res.OK {
		t.Fatal("envelope ok = true for a failed run")
	}
	if got := statuses(res)["openwhispr"]; got != "failed" {
		t.Fatalf("openwhispr status = %q, want failed", got)
	}
	if !strings.Contains(res.Error, "1 of") {
		t.Fatalf("error = %q, want a failed-source count", res.Error)
	}
}

func TestSyncAllRejectsPerSourceFlags(t *testing.T) {
	syncAllHome(t)
	app := cli.App{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	err := app.Run(context.Background(), []string{"sync", "--source", "all", "--source-db", "x.db"})
	if err == nil {
		t.Fatal("expected --source-db to be rejected with --source all")
	}
}
