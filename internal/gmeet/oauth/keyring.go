package oauth

import (
	"github.com/zalando/go-keyring"
)

// KeyringBackend stores tokens in the OS keychain (injectable in tests).
type KeyringBackend interface {
	Get(service, account string) (string, error)
	Set(service, account, password string) error
}

type zalandoKeyring struct{}

func (zalandoKeyring) Get(service, account string) (string, error) {
	return keyring.Get(service, account)
}

func (zalandoKeyring) Set(service, account, password string) error {
	return keyring.Set(service, account, password)
}

func (s TokenStore) keyring() KeyringBackend {
	if s.Keyring != nil {
		return s.Keyring
	}
	return zalandoKeyring{}
}
