package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func queryHash(tool string, args any) string {
	payload := struct {
		Tool string `json:"tool"`
		Args any    `json:"args"`
	}{Tool: tool, Args: args}
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte(tool)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
