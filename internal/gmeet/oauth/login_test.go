package oauth_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
	"golang.org/x/oauth2"
)

type fakeExchanger struct {
	code string
}

func (f fakeExchanger) Exchange(_ context.Context, code string) (*oauth2.Token, error) {
	if code != f.code {
		return nil, fmt.Errorf("unexpected code %q", code)
	}
	return &oauth2.Token{AccessToken: "from-exchange", TokenType: "Bearer"}, nil
}

func TestLoginViaLocalhostRedirect(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	redirect := fmt.Sprintf("http://%s:%s/", host, port)
	cfg := &oauth2.Config{
		ClientID:     "synthetic-client",
		ClientSecret: "synthetic-secret",
		RedirectURL:  redirect,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.test/auth",
			TokenURL: "https://example.test/token",
		},
	}
	store := oauth.TokenStore{FilePath: t.TempDir() + "/token.json", KeyringEnabled: false}
	go func() {
		time.Sleep(50 * time.Millisecond)
		u, _ := url.Parse(redirect)
		u.RawQuery = "code=synthetic-code&state=gmeetcrawl-oauth"
		resp, err := http.Get(u.String())
		if err != nil {
			t.Errorf("redirect GET: %v", err)
			return
		}
		resp.Body.Close()
	}()
	result, err := oauth.Login(context.Background(), cfg, store, oauth.LoginOptions{
		Exchanger:    fakeExchanger{code: "synthetic-code"},
		AuthCodeWait: 5 * time.Second,
		Listen: func(network, address string) (net.Listener, error) {
			return net.Listen(network, net.JoinHostPort(host, port))
		},
		Out: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.UsedLocalhost {
		t.Fatal("expected localhost redirect flow")
	}
	if result.Storage != "file" {
		t.Fatalf("storage = %q", result.Storage)
	}
}

func TestLoginViaManualCode(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{
		ClientID: "synthetic-client",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.test/auth",
			TokenURL: "https://example.test/token",
		},
	}
	store := oauth.TokenStore{FilePath: t.TempDir() + "/token.json", KeyringEnabled: false}
	result, err := oauth.Login(context.Background(), cfg, store, oauth.LoginOptions{
		Exchanger: fakeExchanger{code: "typed-code"},
		In:        strings.NewReader("typed-code\n"),
		Out:       io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.UsedLocalhost {
		t.Fatal("expected manual code flow")
	}
}
