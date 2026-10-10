package secret

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	ProviderKeychain = "keychain"
	ProviderOp       = "op"
)

var (
	ErrNotFound = errors.New("secret: not found")
	ErrInvalid  = errors.New("secret: invalid reference")
)

type Provider interface {
	Resolve(ctx context.Context, ref string) (string, error)
}

type Options struct {
	Provider        string
	KeychainService string
	OpCommand       string
	Keychain        KeychainBackend
	OpRunner        OpRunner
}

func NewProvider(opts Options) (Provider, error) {
	kind := strings.ToLower(strings.TrimSpace(opts.Provider))
	if kind == "" {
		kind = ProviderKeychain
	}
	switch kind {
	case ProviderKeychain:
		service := strings.TrimSpace(opts.KeychainService)
		if service == "" {
			service = KeychainService
		}
		return KeychainProvider{Service: service, Backend: opts.Keychain}, nil
	case ProviderOp:
		cmd := strings.TrimSpace(opts.OpCommand)
		if cmd == "" {
			cmd = "op"
		}
		runner := opts.OpRunner
		if runner == nil {
			runner = execOpRead
		}
		return OpProvider{Command: cmd, Runner: runner}, nil
	default:
		return nil, fmt.Errorf("secret: unknown provider %q", opts.Provider)
	}
}

func ValidateRef(providerKind, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("%w: empty reference", ErrInvalid)
	}
	switch strings.ToLower(strings.TrimSpace(providerKind)) {
	case "", ProviderKeychain:
		if strings.HasPrefix(ref, "op://") {
			return fmt.Errorf("%w: keychain provider cannot use op:// reference", ErrInvalid)
		}
	case ProviderOp:
		if !strings.HasPrefix(ref, "op://") {
			return fmt.Errorf("%w: op provider requires op:// reference", ErrInvalid)
		}
	}
	return nil
}
