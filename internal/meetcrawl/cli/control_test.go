package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/meetcrawl/cli"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/openclaw/crawlkit/control"
)

func TestMetadataJSONControlSchema(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	manifest := mustManifest(t)
	if err := validateManifestExported(manifest); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(manifest.Paths.DefaultConfig, "~/") {
		t.Fatalf("default_config = %q, want ~/… portable path", manifest.Paths.DefaultConfig)
	}
}

func TestMetadataCommandMatchesManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(key, "")
	}

	expected := mustManifest(t)
	out := runMetadataApp(t)
	var got control.Manifest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("parse metadata json: %v\n%s", err, out)
	}
	if err := validateManifestExported(got); err != nil {
		t.Fatal(err)
	}
	if !manifestsEqual(expected, got) {
		t.Fatalf("metadata mismatch:\nwant %#v\ngot  %#v", expected, got)
	}
}

func TestShippedCrawlbarManifestMatchesCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("contrib/crawlbar/meetcrawl.json ships Linux XDG ~/ defaults")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(key, "")
	}

	root := moduleRoot(t)
	shippedPath := filepath.Join(root, "contrib", "crawlbar", "meetcrawl.json")
	shipped, err := os.ReadFile(shippedPath)
	if err != nil {
		t.Fatalf("read shipped manifest: %v", err)
	}
	out := runMetadataApp(t)
	if !jsonEqual(shipped, out) {
		t.Fatalf("shipped manifest differs from meetcrawl metadata --json\nshipped:\n%s\ncommand:\n%s", shipped, out)
	}
	var manifest control.Manifest
	if err := json.Unmarshal(shipped, &manifest); err != nil {
		t.Fatalf("parse shipped manifest: %v", err)
	}
	if err := validateManifestExported(manifest); err != nil {
		t.Fatal(err)
	}
}

func mustManifest(t *testing.T) control.Manifest {
	t.Helper()
	cfg, configPath, err := mconfig.Load("")
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	return cli.ControlManifest(configPath, cfg)
}

func runMetadataApp(t *testing.T) []byte {
	t.Helper()
	var stdout bytes.Buffer
	app := cli.App{Stdout: &stdout}
	if err := app.Run(context.Background(), []string{"--json", "metadata"}); err != nil {
		t.Fatalf("meetcrawl metadata --json: %v", err)
	}
	return bytes.TrimSpace(stdout.Bytes())
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -m: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func validateManifestExported(m control.Manifest) error {
	return cli.ValidateManifest(m)
}

func manifestsEqual(a, b control.Manifest) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return bytes.Equal(aj, bj)
}

func jsonEqual(a, b []byte) bool {
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		return false
	}
	aj, _ := json.Marshal(av)
	bj, _ := json.Marshal(bv)
	return bytes.Equal(aj, bj)
}

func TestMetadataAppRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var buf bytes.Buffer
	app := cli.App{Stdout: &buf}
	if err := app.Run(context.Background(), []string{"--json", "metadata"}); err != nil {
		t.Fatal(err)
	}
	var manifest control.Manifest
	if err := json.Unmarshal(buf.Bytes(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != control.SchemaVersion {
		t.Fatalf("schema = %q", manifest.SchemaVersion)
	}
}
