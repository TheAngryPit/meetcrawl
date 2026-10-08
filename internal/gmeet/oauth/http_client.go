package oauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

func HTTPClient(ctx context.Context, clientPath string, store TokenStore) (*http.Client, error) {
	if strings.TrimSpace(clientPath) == "" {
		return nil, fmt.Errorf("oauth_client_path is required for live sync (or use --fixture)")
	}
	cfg, err := LoadConfig(clientPath)
	if err != nil {
		return nil, err
	}
	tok, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("oauth token: %w", err)
	}
	return oauth2.NewClient(ctx, cfg.TokenSource(ctx, tok)), nil
}
