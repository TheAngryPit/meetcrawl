package source_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestNoCrawlkitRemoteDependency(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "list", "-deps", "./...")
	cmd.Dir = mustModuleRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "github.com/openclaw/crawlkit/remote") {
			t.Fatalf("forbidden dependency: %s", line)
		}
	}
}

func mustModuleRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -m: %v", err)
	}
	return strings.TrimSpace(string(out))
}
