package secret

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const KeychainService = "meetcrawl"

const AccountGrainPAT = "grain-pat"

type KeychainBackend interface {
	Get(service, account string) (string, error)
}

type zalandoKeychain struct{}

func (zalandoKeychain) Get(service, account string) (string, error) {
	return keyring.Get(service, account)
}

type KeychainProvider struct {
	Service string
	Backend KeychainBackend
}

func (p KeychainProvider) backend() KeychainBackend {
	if p.Backend != nil {
		return p.Backend
	}
	return zalandoKeychain{}
}

func (p KeychainProvider) serviceName() string {
	if strings.TrimSpace(p.Service) != "" {
		return p.Service
	}
	return KeychainService
}

func (p KeychainProvider) Resolve(_ context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("%w: empty keychain account", ErrInvalid)
	}
	if strings.HasPrefix(ref, "op://") {
		return "", fmt.Errorf("%w: keychain provider cannot use op:// reference", ErrInvalid)
	}
	raw, err := p.backend().Get(p.serviceName(), ref)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("secret keychain: %w", err)
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ErrNotFound
	}
	return value, nil
}
