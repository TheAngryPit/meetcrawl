package registry

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
)

// Readiness says whether a source can sync without operator input. A source
// that was never set up is skipped with a reason, not failed.
type Readiness struct {
	Ready  bool
	Reason string
}

func ready() Readiness { return Readiness{Ready: true} }

func skip(reason string) Readiness { return Readiness{Reason: reason} }

// readyTimeout bounds one readiness check. A locked keychain or 1Password can
// wait for a person, which would stall an unattended run before later sources.
// The deadline abandons a blocked read and cannot cancel it: go-keyring starts
// the macOS security tool without a context. A real sync of that source makes
// the same read with no deadline at all.
var readyTimeout = 10 * time.Second

// ReadyFor reports whether name can sync unattended. An entry with no check is
// ready. A check that does not finish within readyTimeout skips the source.
func ReadyFor(ctx context.Context, name Name, d Deps) Readiness {
	e, ok := byName[name]
	if !ok || e.Ready == nil {
		return ready()
	}
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	done := make(chan Readiness, 1)
	go func() { done <- e.Ready(ctx, d) }()
	select {
	case r := <-done:
		return r
	case <-ctx.Done():
		return skip("readiness check timed out after " + readyTimeout.String())
	}
}

// fileReady skips only when the file is definitely absent, so any other
// problem surfaces from the sync itself.
func fileReady(path, what string) Readiness {
	path = strings.TrimSpace(path)
	if path == "" {
		return skip(what + " is not configured")
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return skip(what + " not found at " + path)
	}
	return ready()
}

func tokenReady(tokenPath string) Readiness {
	_, err := oauth.DefaultTokenStore(tokenPath).Load()
	switch {
	case err == nil:
		return ready()
	case errors.Is(err, oauth.ErrTokenNotFound):
		return skip("no Google OAuth token (run meetcrawl auth)")
	default:
		return skip("Google OAuth token unreadable: " + err.Error())
	}
}

// secretReady skips whenever the secret cannot be read, including a missing
// keychain or a locked 1Password, so an unattended run on a machine that does
// not use a source does not fail. The reason names the cause. A direct
// meetcrawl sync --source <name> still reports the error itself.
func secretReady(ctx context.Context, d Deps, ref, fallback string) Readiness {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = fallback
	}
	if d.Secrets == nil {
		return ready()
	}
	_, err := d.Secrets.Resolve(ctx, ref)
	switch {
	case err == nil:
		return ready()
	case errors.Is(err, secret.ErrNotFound):
		return skip("no secret " + ref)
	default:
		return skip("secret " + ref + " unavailable: " + err.Error())
	}
}
