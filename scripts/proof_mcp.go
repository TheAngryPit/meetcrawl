//go:build ignore

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/index/reads"
	mcpserver "github.com/TheAngryPit/meetcrawl/internal/mcp"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	cfg := os.Getenv("PROOF_MCP_CFG")
	readsPath := os.Getenv("PROOF_MCP_READS")
	bin := os.Getenv("PROOF_MCP_BIN")
	if cfg == "" || readsPath == "" || bin == "" {
		fmt.Fprintln(os.Stderr, "missing PROOF_MCP_CFG, PROOF_MCP_READS, or PROOF_MCP_BIN")
		os.Exit(2)
	}

	restrictedID := calendarMeetingID("cal-synthetic-001", "2026-01-15T14:00:00Z")

	ctx := context.Background()
	before, err := reads.CountRows(ctx, readsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := exec.Command(bin, "--config", cfg, "mcp")
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "proof-client", Version: "proof"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tools/list:", err)
		os.Exit(1)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"search_meetings", "get_meeting", "list_meetings"} {
		if !names[want] {
			fmt.Fprintf(os.Stderr, "tools/list missing %q\n", want)
			os.Exit(1)
		}
	}

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "search_meetings",
		Arguments: map[string]any{"query": "reuniao", "limit": 10},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "search_meetings:", err)
		os.Exit(1)
	}
	text := textFromResult(res)
	if !strings.HasPrefix(text, mcpserver.UntrustedPrefix) {
		fmt.Fprintf(os.Stderr, "missing untrusted prefix:\n%s\n", text)
		os.Exit(1)
	}

	hidden, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "search_meetings",
		Arguments: map[string]any{"query": "Synthetic standup transcript", "limit": 10},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "search restricted:", err)
		os.Exit(1)
	}
	hiddenText := textFromResult(hidden)
	if strings.Contains(hiddenText, restrictedID) {
		fmt.Fprintf(os.Stderr, "restricted meeting leaked:\n%s\n", hiddenText)
		os.Exit(1)
	}

	after, err := reads.CountRows(ctx, readsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if after-before != 2 {
		fmt.Fprintf(os.Stderr, "read_log delta = %d, want 2\n", after-before)
		os.Exit(1)
	}
}

func textFromResult(res *sdkmcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*sdkmcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func calendarMeetingID(icalUID, start string) string {
	ts, err := time.Parse(time.RFC3339, start)
	if err != nil {
		panic(err)
	}
	payload := icalUID + "|" + ts.UTC().Format(time.RFC3339)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
