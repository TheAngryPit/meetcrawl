package registry

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"

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

// ReadyFor reports whether name can sync unattended. An entry with no check is ready.
func ReadyFor(ctx context.Context, name Name, d Deps) Readiness {
	e, ok := byName[name]
	if !ok || e.Ready == nil {
		return ready()
	}
	return e.Ready(ctx, d)
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
	if oauth.DefaultTokenStore(tokenPath).HasToken() {
		return ready()
	}
	return skip("no Google OAuth token (run meet auth)")
}

// secretReady skips whenever the secret cannot be read, including a missing
// keychain or a locked 1Password, so an unattended run on a machine that does
// not use a source does not fail. The reason names the cause. A direct
// meet sync --source <name> still reports the error itself.
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
