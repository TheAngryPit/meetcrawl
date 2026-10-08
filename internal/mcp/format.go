package mcp

import (
	"encoding/json"
	"strings"
)

func clientNameFromInitialize(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "untrusted:"
	}
	return "untrusted:" + name
}

func cappedText(body string) string {
	if len(body) <= MaxTextBytes {
		return body
	}
	truncated := body[:MaxTextBytes]
	return truncated + "\n\n[truncated: response size cap]"
}

func prefixedJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return UntrustedPrefix + "\n\n" + cappedText(string(b)), nil
}
