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
	shareableID := os.Getenv("PROOF_MCP_SHAREABLE_MEETING_ID")
	if cfg == "" || readsPath == "" || bin == "" || shareableID == "" {
		fmt.Fprintln(os.Stderr, "missing PROOF_MCP_CFG, PROOF_MCP_READS, PROOF_MCP_BIN, or PROOF_MCP_SHAREABLE_MEETING_ID")
		os.Exit(2)
	}

	restrictedID := calendarMeetingID("cal-synthetic-001", "2026-01-15T14:00:00Z")
	const wantReadLogDelta = 4

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

	call := func(tool string, args map[string]any, hideRestricted bool) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", tool, err)
			os.Exit(1)
		}
		if res.IsError {
			fmt.Fprintf(os.Stderr, "%s returned error: %s\n", tool, textFromResult(res))
			os.Exit(1)
		}
		text := textFromResult(res)
		if !strings.HasPrefix(text, mcpserver.UntrustedPrefix) {
			fmt.Fprintf(os.Stderr, "%s missing untrusted prefix:\n%s\n", tool, text)
			os.Exit(1)
		}
		if hideRestricted && strings.Contains(text, restrictedID) {
			fmt.Fprintf(os.Stderr, "%s restricted meeting leaked:\n%s\n", tool, text)
			os.Exit(1)
		}
	}

	call("search_meetings", map[string]any{"query": "reuniao", "limit": 10}, false)
	call("search_meetings", map[string]any{"query": "Synthetic standup transcript", "limit": 10}, true)
	call("get_meeting", map[string]any{"meeting_id": shareableID}, false)
	call("list_meetings", map[string]any{"limit": 50}, true)

	after, err := reads.CountRows(ctx, readsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if after-before != wantReadLogDelta {
		fmt.Fprintf(os.Stderr, "read_log delta = %d, want %d (one row per tool call)\n", after-before, wantReadLogDelta)
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
