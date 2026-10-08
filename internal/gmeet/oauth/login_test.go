package oauth_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
	"golang.org/x/oauth2"
)

type fakeExchanger struct {
	wantCode     string
	wantVerifier string
	gotVerifier  string
}

func (f *fakeExchanger) Exchange(_ context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	f.gotVerifier = codeVerifier
	if code != f.wantCode {
		return nil, fmt.Errorf("unexpected code")
	}
	if f.wantVerifier != "" && codeVerifier != f.wantVerifier {
		return nil, fmt.Errorf("unexpected verifier")
	}
	return &oauth2.Token{AccessToken: "from-exchange", TokenType: "Bearer"}, nil
}

type callbackInfo struct {
	redirect string
	state    string
	authURL  string
}

func runLoopbackLogin(t *testing.T, cfg *oauth2.Config, store oauth.TokenStore, ex *fakeExchanger, mutate func(q url.Values, state string)) (oauth.LoginResult, error) {
	t.Helper()
	infoCh := make(chan callbackInfo, 1)
	resultCh := make(chan struct {
		res oauth.LoginResult
		err error
	}, 1)
	go func() {
		res, err := oauth.Login(context.Background(), cfg, store, oauth.LoginOptions{
			Exchanger:    ex,
			AuthCodeWait: 5 * time.Second,
			Out:          io.Discard,
			OpenBrowser: func(authURL string) error {
				parsed, err := url.Parse(authURL)
				if err != nil {
					return err
				}
				infoCh <- callbackInfo{
					redirect: cfg.RedirectURL,
					state:    parsed.Query().Get("state"),
					authURL:  authURL,
				}
				return nil
			},
		})
		resultCh <- struct {
			res oauth.LoginResult
			err error
		}{res, err}
	}()
	info := <-infoCh
	time.Sleep(30 * time.Millisecond)
	u, err := url.Parse(info.redirect)
	if err != nil {
		return oauth.LoginResult{}, err
	}
	q := u.Query()
	mutate(q, info.state)
	u.RawQuery = q.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		return oauth.LoginResult{}, err
	}
	resp.Body.Close()
	out := <-resultCh
	return out.res, out.err
}

func TestLoginLoopbackPKCEAndEphemeralPort(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{
		ClientID:     "synthetic-client",
		ClientSecret: "synthetic-secret",
		RedirectURL:  "http://localhost",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.test/auth",
			TokenURL: "https://example.test/token",
		},
	}
	store := oauth.TokenStore{FilePath: t.TempDir() + "/token.json", KeyringEnabled: false}
	ex := &fakeExchanger{wantCode: "synthetic-code"}
	var capturedAuthURL string
	infoCh := make(chan callbackInfo, 1)
	resultCh := make(chan struct {
		res oauth.LoginResult
		err error
	}, 1)
	go func() {
		res, err := oauth.Login(context.Background(), cfg, store, oauth.LoginOptions{
			Exchanger:    ex,
			AuthCodeWait: 5 * time.Second,
			Out:          io.Discard,
			OpenBrowser: func(authURL string) error {
				parsed, _ := url.Parse(authURL)
				capturedAuthURL = authURL
				infoCh <- callbackInfo{redirect: cfg.RedirectURL, state: parsed.Query().Get("state")}
				return nil
			},
		})
		resultCh <- struct {
			res oauth.LoginResult
			err error
		}{res, err}
	}()
	info := <-infoCh
	time.Sleep(30 * time.Millisecond)
	if !strings.HasPrefix(info.redirect, "http://127.0.0.1:") {
		t.Fatalf("redirect = %q", info.redirect)
	}
	if strings.Contains(info.redirect, ":80") {
		t.Fatalf("redirect must not bind port 80: %q", info.redirect)
	}
	if info.state == "" {
		t.Fatal("expected random state in auth URL")
	}
	u, _ := url.Parse(info.redirect)
	q := u.Query()
	q.Set("code", "synthetic-code")
	q.Set("state", info.state)
	u.RawQuery = q.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	out := <-resultCh
	if out.err != nil {
		t.Fatal(out.err)
	}
	if !out.res.UsedLocalhost {
		t.Fatal("expected loopback")
	}
	authParsed, err := url.Parse(capturedAuthURL)
	if err != nil {
		t.Fatal(err)
	}
	if authParsed.Query().Get("code_challenge") == "" || authParsed.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("missing PKCE on auth URL: %s", capturedAuthURL)
	}
	if ex.gotVerifier == "" {
		t.Fatal("Exchange missing PKCE verifier")
	}
}

func TestLoginMissingStateRejected(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{
		ClientID:    "synthetic-client",
		RedirectURL: "http://localhost",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://example.test/auth", TokenURL: "https://example.test/token"},
	}
	store := oauth.TokenStore{FilePath: t.TempDir() + "/token.json", KeyringEnabled: false}
	_, err := runLoopbackLogin(t, cfg, store, &fakeExchanger{wantCode: "x"}, func(q url.Values, _ string) {
		q.Set("code", "synthetic-code")
	})
	if err == nil {
		t.Fatal("expected missing state error")
	}
}

func TestLoginWrongStateRejected(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{
		ClientID:    "synthetic-client",
		RedirectURL: "http://localhost",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://example.test/auth", TokenURL: "https://example.test/token"},
	}
	store := oauth.TokenStore{FilePath: t.TempDir() + "/token.json", KeyringEnabled: false}
	_, err := runLoopbackLogin(t, cfg, store, &fakeExchanger{wantCode: "x"}, func(q url.Values, _ string) {
		q.Set("code", "synthetic-code")
		q.Set("state", "not-the-random-state")
	})
	if err == nil {
		t.Fatal("expected wrong state error")
	}
}

func TestLoginManualWithPKCE(t *testing.T) {
	t.Parallel()
	cfg := &oauth2.Config{
		ClientID: "synthetic-client",
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.test/auth", TokenURL: "https://example.test/token"},
	}
	store := oauth.TokenStore{FilePath: t.TempDir() + "/token.json", KeyringEnabled: false}
	ex := &fakeExchanger{wantCode: "typed-code"}
	_, err := oauth.Login(context.Background(), cfg, store, oauth.LoginOptions{
		Manual:    true,
		Exchanger: ex,
		In:        strings.NewReader("typed-code\n"),
		Out:       io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ex.gotVerifier == "" {
		t.Fatal("manual flow must pass PKCE verifier to Exchange")
	}
}
