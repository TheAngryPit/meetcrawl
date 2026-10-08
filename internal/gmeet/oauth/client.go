package oauth

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func LoadConfig(clientPath string) (*oauth2.Config, error) {
	clientJSON, err := os.ReadFile(clientPath)
	if err != nil {
		return nil, fmt.Errorf("read oauth client: %w", err)
	}
	cfg, err := google.ConfigFromJSON(clientJSON, Scopes()...)
	if err != nil {
		return nil, fmt.Errorf("parse oauth client: %w", err)
	}
	return cfg, nil
}

func InstallClient(srcPath, destPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read client json: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(destPath, data, 0o600); err != nil {
		return fmt.Errorf("write client json: %w", err)
	}
	return nil
}
