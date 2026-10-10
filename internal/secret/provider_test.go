package secret_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/secret"
	"github.com/zalando/go-keyring"
)

type mapKeychain map[string]string

func (m mapKeychain) Get(service, account string) (string, error) {
	key := service + "/" + account
	v, ok := m[key]
	if !ok || v == "" {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func TestKeychainProviderResolve(t *testing.T) {
	t.Parallel()
	p := secret.KeychainProvider{
		Service: secret.KeychainService,
		Backend: mapKeychain{secret.KeychainService + "/" + secret.AccountGrainPAT: " pat "},
	}
	got, err := p.Resolve(context.Background(), secret.AccountGrainPAT)
	if err != nil {
		t.Fatalf("Resolve() = %v", err)
	}
	if got != "pat" {
		t.Fatalf("value = %q", got)
	}
}

func TestKeychainMissingFailsClosed(t *testing.T) {
	t.Parallel()
	p := secret.KeychainProvider{Service: secret.KeychainService, Backend: mapKeychain{}}
	_, err := p.Resolve(context.Background(), secret.AccountGrainPAT)
	if !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestOpProviderRequiresOpReference(t *testing.T) {
	t.Parallel()
	p, err := secret.NewProvider(secret.Options{Provider: secret.ProviderOp})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Resolve(context.Background(), "grain-pat")
	if !errors.Is(err, secret.ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestOpProviderReadOnlyRunner(t *testing.T) {
	t.Parallel()
	var seenCmd, seenRef string
	p, err := secret.NewProvider(secret.Options{
		Provider: secret.ProviderOp,
		OpRunner: func(_ context.Context, command, ref string) (string, error) {
			seenCmd = command
			seenRef = ref
			return "token-from-op", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := "op://Synthetic Vault/Grain/credential"
	got, err := p.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatalf("Resolve() = %v", err)
	}
	if got != "token-from-op" || seenRef != ref || seenCmd != "op" {
		t.Fatalf("got=%q cmd=%q ref=%q", got, seenCmd, seenRef)
	}
}

func TestValidateRefProviderMismatch(t *testing.T) {
	t.Parallel()
	if err := secret.ValidateRef(secret.ProviderKeychain, "op://x/y/z"); err == nil {
		t.Fatal("expected invalid keychain ref")
	}
	if err := secret.ValidateRef(secret.ProviderOp, "grain-pat"); err == nil {
		t.Fatal("expected invalid op ref")
	}
}
