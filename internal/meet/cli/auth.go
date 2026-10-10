package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/oauth"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
)

func (a App) runAuth(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	cfg, configPath, err := mconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	gcfg := cfg.GMeetConfig()
	clientSrc, _ := flagValue(args, "--client")
	manual := hasFlag(args, "--manual")
	if clientSrc != "" {
		if err := oauth.InstallClient(clientSrc, gcfg.OAuthClientPath); err != nil {
			return err
		}
	}
	if _, err := os.Stat(gcfg.OAuthClientPath); err != nil {
		return fmt.Errorf("oauth client JSON not found at %s (use --client <path> or place the desktop client file there)", gcfg.OAuthClientPath)
	}
	oauthCfg, err := oauth.LoadConfig(gcfg.OAuthClientPath)
	if err != nil {
		return err
	}
	store := oauth.DefaultTokenStore(gcfg.TokenPath)
	stderr := a.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	loginOut := io.Writer(stderr)
	if !flags.JSON {
		loginOut = io.MultiWriter(w, stderr)
	}
	result, err := oauth.Login(ctx, oauthCfg, store, oauth.LoginOptions{
		Manual: manual,
		Out:    loginOut,
		In:     os.Stdin,
	})
	if err != nil {
		return err
	}
	payload := map[string]any{
		"config_path":   configPath,
		"client_path":   gcfg.OAuthClientPath,
		"token_path":    gcfg.TokenPath,
		"token_storage": result.Storage,
		"localhost":     result.UsedLocalhost,
	}
	if flags.JSON {
		return writeEnvelope(w, payload)
	}
	printKV(w, "config", configPath)
	printKV(w, "oauth_client", gcfg.OAuthClientPath)
	printKV(w, "token_storage", result.Storage)
	if gcfg.TokenPath != "" {
		printKV(w, "token_file", gcfg.TokenPath)
	}
	return nil
}
