package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/adapters/registry"
	calprovider "github.com/TheAngryPit/meetcrawl/internal/calendar/provider"
	"github.com/TheAngryPit/meetcrawl/internal/index/build"
	"github.com/TheAngryPit/meetcrawl/internal/index/reads"
	mcpserver "github.com/TheAngryPit/meetcrawl/internal/mcp"
	"github.com/TheAngryPit/meetcrawl/internal/meet/buildinfo"
	mconfig "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/meetcrawl/doctor"
	mruntime "github.com/TheAngryPit/meetcrawl/internal/meetcrawl/runtime"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

const usage = `meet — local meetings index (source adapters)

Usage:
  meet [--json] [--config <path>] <command> [args]

Commands:
  init       Create config, reads.db schema, and data directories
  doctor     Check source archive paths and index database
  sync       Sync one source adapter (--source openwhispr|gmeet|export-file)
  index      Rebuild meetcrawl.db from source archives (read-only; optional --calendar-fixture)
  status     Show index status
  search     Full-text search indexed meeting content
  metadata   Emit crawlkit.control.v1 manifest (--json)
  mcp        Read-only MCP server on stdio
  auth       Google OAuth for gmeet (optional --client, --manual)

Global flags:
  --json       JSON output (place before the command)
  --config     Path to config.toml (default ~/.config/meetcrawl/config.toml)
  --version    Print version and exit
`

type App struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (a App) Run(ctx context.Context, args []string) error {
	stdout := a.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	flags, rest := parseGlobalFlags(args)
	if flags.Version {
		return a.runVersion(stdout, flags)
	}
	if flags.Help || len(rest) == 0 {
		_, err := io.WriteString(stdout, usage)
		return err
	}
	cmd, cmdArgs := rest[0], rest[1:]
	switch cmd {
	case "init":
		return a.runInit(stdout, flags)
	case "doctor":
		return a.runDoctor(stdout, flags)
	case "sync":
		return a.runSync(ctx, stdout, flags, cmdArgs)
	case "auth":
		return a.runAuth(ctx, stdout, flags, cmdArgs)
	case "index":
		return a.runIndex(ctx, stdout, flags, cmdArgs)
	case "status":
		return a.runStatus(ctx, stdout, flags)
	case "search":
		return a.runSearch(ctx, stdout, flags, cmdArgs)
	case "metadata":
		return a.runMetadata(ctx, stdout, flags)
	case "mcp":
		return mcpserver.Run(ctx, mcpserver.Options{ConfigPath: flags.ConfigPath})
	case "help":
		_, err := io.WriteString(stdout, usage)
		return err
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func (a App) runVersion(w io.Writer, flags GlobalFlags) error {
	info := buildinfo.Current()
	if flags.JSON {
		return writeEnvelope(w, info)
	}
	printKV(w, "name", info.Name)
	printKV(w, "version", info.Version)
	return nil
}

func (a App) runInit(w io.Writer, flags GlobalFlags) error {
	cfg, defaultPath, err := mconfig.Defaults()
	if err != nil {
		return err
	}
	path, err := mconfig.App().ResolveConfigPath(flags.ConfigPath)
	if err != nil {
		return err
	}
	if path == "" {
		path = defaultPath
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists at %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := mconfig.EnsureDirs(cfg); err != nil {
		return err
	}
	if err := reads.EnsureSchema(context.Background(), cfg.ReadsDBPath); err != nil {
		return err
	}
	if err := mconfig.Save(path, cfg); err != nil {
		return err
	}
	result := map[string]string{
		"config_path": path,
		"db_path":     cfg.DBPath,
		"reads_db":    cfg.ReadsDBPath,
	}
	if flags.JSON {
		return writeEnvelope(w, result)
	}
	printKV(w, "config", path)
	printKV(w, "database", cfg.DBPath)
	printKV(w, "reads_db", cfg.ReadsDBPath)
	return nil
}

func (a App) runMetadata(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	_ = ctx
	cfg, configPath, err := mconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	manifest := ControlManifest(configPath, cfg)
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	if flags.JSON {
		return writeJSON(w, manifest)
	}
	printKV(w, "id", manifest.ID)
	printKV(w, "binary", manifest.Binary.Name)
	return nil
}

func (a App) runDoctor(w io.Writer, flags GlobalFlags) error {
	cfg, configPath, err := mconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	report := doctor.Run(cfg, configPath)
	if flags.JSON {
		return writeEnvelope(w, report)
	}
	printKV(w, "config", report.ConfigPath)
	printKV(w, "openwhispr_db", report.Archives.Whispcrawl.Exists)
	printKV(w, "gmeet_db", report.Archives.Gmeetcrawl.Exists)
	printKV(w, "export_file_db", report.Archives.Exportcrawl.Exists)
	printKV(w, "index_db", report.Archives.Index.Exists)
	return nil
}

func (a App) runSync(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	sourceRaw, ok := flagValue(args, "--source")
	if !ok || strings.TrimSpace(sourceRaw) == "" {
		return fmt.Errorf("usage: meet sync --source openwhispr|gmeet|export-file [flags]")
	}
	name, err := registry.ParseName(sourceRaw)
	if err != nil {
		return err
	}
	cfg, _, err := mconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	if err := mconfig.EnsureSourceDirs(cfg); err != nil {
		return err
	}
	opts := registry.SyncOptions{
		SourceDB:   trimFlag(args, "--source-db"),
		FixtureDir: trimFlag(args, "--fixture"),
		SourceDir:  trimFlag(args, "--source-dir"),
	}
	deps := registry.Deps{Config: cfg, CrawlerVersion: buildinfo.Current().Version}
	adapter, err := registry.NewAdapter(name, deps, opts)
	if err != nil {
		return err
	}
	outcome, err := adapter.Sync(ctx)
	payload := syncPayload(name, opts, outcome)
	if err != nil && outcome.Code == source.OutcomeUnsupportedSchema {
		if flags.JSON {
			_ = writeJSON(w, Envelope{OK: false, Result: payload, Error: string(outcome.Code)})
		} else {
			printKV(w, "code", outcome.Code)
			printKV(w, "detail", outcome.Detail)
		}
		return err
	}
	if err != nil {
		if flags.JSON {
			_ = writeErrorJSON(w, err)
		}
		return err
	}
	if flags.JSON {
		return writeEnvelope(w, payload)
	}
	printKV(w, "code", outcome.Code)
	printKV(w, "artifacts", len(outcome.Artifacts))
	printKV(w, "source", name)
	return nil
}

func syncPayload(name registry.Name, opts registry.SyncOptions, outcome source.SyncOutcome) map[string]any {
	payload := map[string]any{
		"code":      outcome.Code,
		"artifacts": len(outcome.Artifacts),
		"source":    name,
	}
	if outcome.Detail != "" {
		payload["detail"] = outcome.Detail
	}
	if opts.SourceDB != "" {
		payload["source_db"] = opts.SourceDB
	}
	if opts.FixtureDir != "" {
		payload["fixture"] = opts.FixtureDir
	}
	if opts.SourceDir != "" {
		payload["source_dir"] = opts.SourceDir
	}
	return payload
}

func trimFlag(args []string, name string) string {
	v, _ := flagValue(args, name)
	return strings.TrimSpace(v)
}

func (a App) runIndex(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	cfg, _, err := mconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	if err := mconfig.EnsureDirs(cfg); err != nil {
		return err
	}
	if err := reads.EnsureSchema(ctx, cfg.ReadsDBPath); err != nil {
		return err
	}
	whispDB := cfg.WhispcrawlDB
	if override, ok := flagValue(args, "--openwhispr-db"); ok {
		whispDB = override
	}
	if override, ok := flagValue(args, "--whispcrawl-db"); ok {
		whispDB = override
	}
	gmeetDB := cfg.GmeetcrawlDB
	if override, ok := flagValue(args, "--gmeet-db"); ok {
		gmeetDB = override
	}
	if override, ok := flagValue(args, "--gmeetcrawl-db"); ok {
		gmeetDB = override
	}
	exportDB := cfg.ExportcrawlDB
	if override, ok := flagValue(args, "--export-file-db"); ok {
		exportDB = override
	}
	if override, ok := flagValue(args, "--exportcrawl-db"); ok {
		exportDB = override
	}
	calendarOpts := calprovider.Options{
		FixtureDir:      cfg.Calendar.FixtureDir,
		OAuthClientPath: cfg.Calendar.OAuthClientPath,
		TokenPath:       cfg.Calendar.TokenPath,
	}
	if override, ok := flagValue(args, "--calendar-fixture"); ok {
		calendarOpts.FixtureDir = override
	}
	result, err := build.Run(ctx, build.Options{
		IndexDBPath: cfg.DBPath,
		Privacy:     cfg.Privacy,
		Calendar:    calendarOpts,
		Sources: []build.SourceArchive{
			{Kind: source.KindOpenWhispr, Path: whispDB},
			{Kind: source.KindGMeetGemini, Path: gmeetDB},
			{Kind: source.KindExportFile, Path: exportDB},
		},
	})
	if err != nil {
		if flags.JSON {
			_ = writeErrorJSON(w, err)
		}
		return err
	}
	if flags.JSON {
		return writeEnvelope(w, result)
	}
	printKV(w, "meetings", result.Meetings)
	printKV(w, "contents", result.Contents)
	printKV(w, "source_links", result.SourceLinks)
	printKV(w, "database", cfg.DBPath)
	return nil
}

func (a App) runStatus(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	rt, err := mruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	status, err := rt.Store.Status(ctx)
	if err != nil {
		return err
	}
	if flags.JSON {
		return writeEnvelope(w, status)
	}
	printKV(w, "database", status.DBPath)
	printKV(w, "meetings", status.Meetings)
	printKV(w, "contents", status.Contents)
	printKV(w, "sources", status.Sources)
	return nil
}

func (a App) runSearch(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	query := strings.TrimSpace(strings.Join(nonFlagArgs(args), " "))
	if query == "" {
		return fmt.Errorf("usage: meet search <query> [--limit <n>]")
	}
	rt, err := mruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	hits, err := rt.Store.Search(ctx, query, parseLimit(args, 50))
	if err != nil {
		return err
	}
	if flags.JSON {
		return writeEnvelope(w, map[string]any{"query": query, "hits": hits})
	}
	for _, hit := range hits {
		fmt.Fprintf(w, "%s\t%s\n", hit.MeetingID, hit.Snippet)
	}
	return nil
}

func nonFlagArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--limit" || args[i] == "--openwhispr-db" || args[i] == "--whispcrawl-db" ||
			args[i] == "--gmeet-db" || args[i] == "--gmeetcrawl-db" ||
			args[i] == "--export-file-db" || args[i] == "--exportcrawl-db" {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func parseLimit(args []string, fallback int) int {
	if limit, ok := flagValue(args, "--limit"); ok {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}
