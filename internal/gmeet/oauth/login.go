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
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// CodeExchanger exchanges an authorization code for a token (injectable in tests).
type CodeExchanger interface {
	Exchange(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error)
}

type configExchanger struct {
	cfg *oauth2.Config
}

func (c configExchanger) Exchange(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	return c.cfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
}

type LoginOptions struct {
	Config       *oauth2.Config
	Exchanger    CodeExchanger
	Manual       bool
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
	if opts.Listen == nil {
		opts.Listen = net.Listen
	}
	if opts.AuthCodeWait <= 0 {
		opts.AuthCodeWait = 2 * time.Minute
	}
	verifier := oauth2.GenerateVerifier()
	state, err := randomOAuthState()
	if err != nil {
		return LoginResult{}, err
	}
	if opts.Exchanger == nil {
		opts.Exchanger = configExchanger{cfg: cfg}
	}
	var tok *oauth2.Token
	var redirect string
	var usedLocalhost bool
	if opts.Manual {
		tok, err = loginViaManualCode(ctx, cfg, state, verifier, opts)
	} else {
		tok, redirect, err = loginViaLoopback(ctx, cfg, state, verifier, opts)
		usedLocalhost = true
	}
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

func loginViaLoopback(ctx context.Context, cfg *oauth2.Config, state, verifier string, opts LoginOptions) (*oauth2.Token, string, error) {
	callbackPath := loopbackCallbackPath(cfg.RedirectURL)
	ln, err := opts.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("listen for oauth redirect: %w", err)
	}
	defer ln.Close()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return nil, "", err
	}
	redirectUsed := fmt.Sprintf("http://127.0.0.1:%s%s", port, callbackPath)
	cfg.RedirectURL = redirectUsed

	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.S256ChallengeOption(verifier),
	)
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
	var deliverOnce sync.Once
	deliver := func(res result) {
		deliverOnce.Do(func() { ch <- res })
	}

	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != callbackPath {
				http.NotFound(w, r)
				return
			}
			q := r.URL.Query()
			gotState := q.Get("state")
			if gotState == "" {
				http.Error(w, "missing state", http.StatusBadRequest)
				deliver(result{err: fmt.Errorf("oauth redirect missing state")})
				return
			}
			if !oauthStateMatches(gotState, state) {
				http.Error(w, "state mismatch", http.StatusBadRequest)
				deliver(result{err: fmt.Errorf("oauth state mismatch")})
				return
			}
			if errMsg := q.Get("error"); errMsg != "" {
				http.Error(w, "authorization failed", http.StatusBadRequest)
				deliver(result{err: fmt.Errorf("oauth authorization failed")})
				return
			}
			code := q.Get("code")
			if code == "" {
				http.Error(w, "missing code", http.StatusBadRequest)
				deliver(result{err: fmt.Errorf("oauth redirect missing code")})
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, "Authorization complete. You can close this tab and return to the terminal.\n")
			deliver(result{code: code})
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
		return nil, "", fmt.Errorf("timed out waiting for oauth redirect")
	case res = <-ch:
	}
	if res.err != nil {
		return nil, "", res.err
	}
	tok, err := opts.Exchanger.Exchange(ctx, res.code, verifier)
	return tok, redirectUsed, err
}

func loginViaManualCode(ctx context.Context, cfg *oauth2.Config, state, verifier string, opts LoginOptions) (*oauth2.Token, error) {
	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.S256ChallengeOption(verifier),
	)
	if opts.OpenBrowser != nil {
		_ = opts.OpenBrowser(authURL)
	}
	if opts.Out != nil {
		fmt.Fprintf(opts.Out, "Manual OAuth is deprecated by Google and may not work. Prefer the default loopback login.\n\nOpen this URL:\n%s\n\nPaste the authorization code: ", authURL)
	}
	in := opts.In
	if in == nil {
		return nil, fmt.Errorf("authorization code input required for --manual")
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read authorization code: %w", err)
	}
	code := strings.TrimSpace(line)
	if code == "" {
		return nil, fmt.Errorf("authorization code is empty")
	}
	return opts.Exchanger.Exchange(ctx, code, verifier)
}

func loopbackCallbackPath(redirect string) string {
	raw := strings.TrimSpace(redirect)
	if raw == "" {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Path == "" {
		return "/"
	}
	return u.Path
}
