package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
)

const keyringService = "gmeetcrawl"
const keyringAccount = "oauth-token"

// TokenStore persists OAuth tokens outside the archive (keychain or 0600 file).
type TokenStore struct {
	FilePath       string
	KeyringEnabled bool
	Keyring        KeyringBackend
}

func DefaultTokenStore(filePath string) TokenStore {
	return TokenStore{FilePath: filePath, KeyringEnabled: true}
}

func (s TokenStore) HasToken() bool {
	_, err := s.Load()
	return err == nil
}

func (s TokenStore) Load() (*oauth2.Token, error) {
	if s.KeyringEnabled {
		if raw, err := s.keyring().Get(keyringService, keyringAccount); err == nil && strings.TrimSpace(raw) != "" {
			var tok oauth2.Token
			if err := json.Unmarshal([]byte(raw), &tok); err != nil {
				return nil, fmt.Errorf("parse keychain token: %w", err)
			}
			return &tok, nil
		}
	}
	if strings.TrimSpace(s.FilePath) == "" {
		return nil, errors.New("oauth token not found")
	}
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("oauth token not found")
		}
		return nil, fmt.Errorf("read oauth token: %w", err)
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse oauth token: %w", err)
	}
	return &tok, nil
}

func (s TokenStore) Save(tok *oauth2.Token) (string, error) {
	if tok == nil {
		return "", fmt.Errorf("token is nil")
	}
	payload, err := json.Marshal(tok)
	if err != nil {
		return "", err
	}
	if s.KeyringEnabled {
		if err := s.keyring().Set(keyringService, keyringAccount, string(payload)); err == nil {
			if err := s.removeTokenFile(); err != nil {
				return "", err
			}
			return "keychain", nil
		}
	}
	if strings.TrimSpace(s.FilePath) == "" {
		return "", fmt.Errorf("token_path is required when keychain is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(s.FilePath), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(s.FilePath, payload, 0o600); err != nil {
		return "", fmt.Errorf("write oauth token: %w", err)
	}
	return "file", nil
}

func (s TokenStore) removeTokenFile() error {
	if strings.TrimSpace(s.FilePath) == "" {
		return nil
	}
	err := os.Remove(s.FilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale oauth token file: %w", err)
	}
	return nil
}

// StorageInUse reports where a token would be loaded from ("keychain", "file", or "").
func (s TokenStore) StorageInUse() string {
	if s.KeyringEnabled {
		if raw, err := s.keyring().Get(keyringService, keyringAccount); err == nil && strings.TrimSpace(raw) != "" {
			return "keychain"
		}
	}
	if strings.TrimSpace(s.FilePath) == "" {
		return ""
	}
	if _, err := os.Stat(s.FilePath); err != nil {
		return ""
	}
	if _, err := s.loadFromFile(); err == nil {
		return "file"
	}
	return ""
}

func (s TokenStore) loadFromFile() (*oauth2.Token, error) {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func (s TokenStore) FileState() (exists bool, bytes int64) {
	if strings.TrimSpace(s.FilePath) == "" {
		return false, 0
	}
	info, err := os.Stat(s.FilePath)
	if err != nil {
		return false, 0
	}
	return true, info.Size()
}

func (s TokenStore) KeychainPresent() bool {
	if !s.KeyringEnabled {
		return false
	}
	_, err := s.keyring().Get(keyringService, keyringAccount)
	return err == nil
}
