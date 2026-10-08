package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
)

func (a App) runAuth(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	cfg, configPath, err := gconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	clientSrc, _ := flagValue(args, "--client")
	if clientSrc != "" {
		if err := oauth.InstallClient(clientSrc, cfg.OAuthClientPath); err != nil {
			return err
		}
	}
	if _, err := os.Stat(cfg.OAuthClientPath); err != nil {
		return fmt.Errorf("oauth client JSON not found at %s (use --client <path> or place the desktop client file there)", cfg.OAuthClientPath)
	}
	oauthCfg, err := oauth.LoadConfig(cfg.OAuthClientPath)
	if err != nil {
		return err
	}
	store := oauth.DefaultTokenStore(cfg.TokenPath)
	stderr := a.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	loginOut := io.Writer(stderr)
	if !flags.JSON {
		loginOut = io.MultiWriter(w, stderr)
	}
	result, err := oauth.Login(ctx, oauthCfg, store, oauth.LoginOptions{
		Out: loginOut,
		In:  os.Stdin,
	})
	if err != nil {
		return err
	}
	payload := map[string]any{
		"config_path":   configPath,
		"client_path":   cfg.OAuthClientPath,
		"token_path":    cfg.TokenPath,
		"token_storage": result.Storage,
		"localhost":     result.UsedLocalhost,
	}
	if flags.JSON {
		return writeEnvelope(w, payload)
	}
	printKV(w, "config", configPath)
	printKV(w, "oauth_client", cfg.OAuthClientPath)
	printKV(w, "token_storage", result.Storage)
	if cfg.TokenPath != "" {
		printKV(w, "token_file", cfg.TokenPath)
	}
	return nil
}
