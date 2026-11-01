package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/meet/cli"
)

func TestInitAndSyncCreateOnlyTheDirsTheyNeed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "MEETCRAWL_CONFIG"} {
		t.Setenv(key, "")
	}
	app := cli.App{Stdout: io.Discard, Stderr: io.Discard}
	run := func(args ...string) {
		t.Helper()
		if err := app.Run(context.Background(), args); err != nil {
			t.Fatalf("meetcrawl %v: %v", args, err)
		}
	}
	appDirs := func() map[string]bool {
		t.Helper()
		found := map[string]bool{}
		err := filepath.WalkDir(home, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				found[d.Name()] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return found
	}

	run("init")
	got := appDirs()
	if !got["meetcrawl"] {
		t.Fatal("init did not create the meetcrawl dir")
	}
	for _, name := range []string{"whispcrawl", "gmeetcrawl", "exportcrawl", "granolacrawl", "pocketcrawl"} {
		if got[name] {
			t.Fatalf("init created %s, which no command has needed yet", name)
		}
	}

	fixture := filepath.Join(moduleRoot(t), "testdata", "fixtures", "openwhispr", "supported", "transcriptions.db")
	run("sync", "--source", "openwhispr", "--source-db", fixture)
	got = appDirs()
	if !got["whispcrawl"] {
		t.Fatal("sync --source openwhispr did not create the whispcrawl dir")
	}
	for _, name := range []string{"gmeetcrawl", "exportcrawl", "granolacrawl", "pocketcrawl"} {
		if got[name] {
			t.Fatalf("sync --source openwhispr created %s", name)
		}
	}
}
