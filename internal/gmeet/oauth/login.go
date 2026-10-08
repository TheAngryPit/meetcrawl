package oauth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// CodeExchanger exchanges an authorization code for a token (injectable in tests).
type CodeExchanger interface {
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
}

type configExchanger struct {
	cfg *oauth2.Config
}

func (c configExchanger) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.cfg.Exchange(ctx, code)
}

type LoginOptions struct {
	Config       *oauth2.Config
	Exchanger    CodeExchanger
	OpenBrowser  func(url string) error
	Listen       func(network, address string) (net.Listener, error)
	AuthCodeWait time.Duration
	Out          io.Writer
	In           io.Reader
}

type LoginResult struct {
	Token         *oauth2.Token
	Storage       string
	RedirectUsed  string
	UsedLocalhost bool
}

func Login(ctx context.Context, cfg *oauth2.Config, store TokenStore, opts LoginOptions) (LoginResult, error) {
	if cfg == nil {
		return LoginResult{}, fmt.Errorf("oauth config is nil")
	}
	if opts.Exchanger == nil {
		opts.Exchanger = configExchanger{cfg: cfg}
	}
	if opts.Listen == nil {
		opts.Listen = net.Listen
	}
	if opts.AuthCodeWait <= 0 {
		opts.AuthCodeWait = 2 * time.Minute
	}
	tok, redirect, usedLocalhost, err := obtainToken(ctx, cfg, opts)
	if err != nil {
		return LoginResult{}, err
	}
	storage, err := store.Save(tok)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Token:         tok,
		Storage:       storage,
		RedirectUsed:  redirect,
		UsedLocalhost: usedLocalhost,
	}, nil
}

func obtainToken(ctx context.Context, cfg *oauth2.Config, opts LoginOptions) (*oauth2.Token, string, bool, error) {
	if redirect := pickLocalhostRedirect(cfg.RedirectURL); redirect != "" {
		tok, err := loginViaLocalhost(ctx, cfg, redirect, opts)
		return tok, redirect, true, err
	}
	return loginViaManualCode(ctx, cfg, opts)
}

func pickLocalhostRedirect(redirect string) string {
	raw := strings.TrimSpace(redirect)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost") {
		return raw
	}
	return ""
}

func loginViaLocalhost(ctx context.Context, cfg *oauth2.Config, redirect string, opts LoginOptions) (*oauth2.Token, error) {
	u, err := url.Parse(redirect)
	if err != nil {
		return nil, err
	}
	port := u.Port()
	if port == "" {
		port = "80"
	}
	ln, err := opts.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		return nil, fmt.Errorf("listen for oauth redirect: %w", err)
	}
	defer ln.Close()

	state := "gmeetcrawl-oauth"
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	if opts.OpenBrowser != nil {
		_ = opts.OpenBrowser(authURL)
	}
	if opts.Out != nil {
		fmt.Fprintf(opts.Out, "Open this URL to authorize gmeetcrawl:\n%s\n\n", authURL)
	}

	type result struct {
		code string
		err  error
	}
	ch := make(chan result, 1)
	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if q.Get("state") != "" && q.Get("state") != state {
				http.Error(w, "state mismatch", http.StatusBadRequest)
				ch <- result{err: fmt.Errorf("oauth state mismatch")}
				return
			}
			if errMsg := q.Get("error"); errMsg != "" {
				http.Error(w, errMsg, http.StatusBadRequest)
				ch <- result{err: fmt.Errorf("oauth error: %s", errMsg)}
				return
			}
			code := q.Get("code")
			if code == "" {
				http.Error(w, "missing code", http.StatusBadRequest)
				ch <- result{err: fmt.Errorf("oauth redirect missing code")}
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, "Authorization complete. You can close this tab and return to the terminal.\n")
			ch <- result{code: code}
		}),
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	waitCtx, cancel := context.WithTimeout(ctx, opts.AuthCodeWait)
	defer cancel()
	var res result
	select {
	case <-waitCtx.Done():
		return nil, fmt.Errorf("timed out waiting for oauth redirect")
	case res = <-ch:
	}
	if res.err != nil {
		return nil, res.err
	}
	return opts.Exchanger.Exchange(ctx, res.code)
}

func loginViaManualCode(ctx context.Context, cfg *oauth2.Config, opts LoginOptions) (*oauth2.Token, string, bool, error) {
	state := "gmeetcrawl-oauth"
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	if opts.OpenBrowser != nil {
		_ = opts.OpenBrowser(authURL)
	}
	if opts.Out != nil {
		fmt.Fprintf(opts.Out, "Open this URL to authorize gmeetcrawl:\n%s\n\nPaste the authorization code: ", authURL)
	}
	in := opts.In
	if in == nil {
		return nil, "", false, fmt.Errorf("authorization code input required")
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		return nil, "", false, fmt.Errorf("read authorization code: %w", err)
	}
	code := strings.TrimSpace(line)
	if code == "" {
		return nil, "", false, fmt.Errorf("authorization code is empty")
	}
	tok, err := opts.Exchanger.Exchange(ctx, code)
	return tok, "", false, err
}
