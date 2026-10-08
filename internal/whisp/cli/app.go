package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/TheAngryPit/meetcrawl/internal/whisp/buildinfo"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/doctor"
	wruntime "github.com/TheAngryPit/meetcrawl/internal/whisp/runtime"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/sync"
)

const usage = `whispcrawl — read-only OpenWhispr meeting archive

Usage:
  whispcrawl [--json] [--config <path>] <command> [args]

Commands:
  init       Create config and data directories
  doctor     Check OpenWhispr source and archive paths
  sync       Snapshot OpenWhispr DB and ingest meeting artifacts
  status     Show archive status
  search     Full-text search archived artifacts
  metadata   Emit crawlkit.control.v1 manifest
  sql        Run read-only SQL against the archive

Global flags:
  --json       JSON output (place before the command, or after for most commands)
  --config     Path to config.toml
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
		return a.runDoctor(ctx, stdout, flags, cmdArgs)
	case "metadata":
		return a.runMetadata(ctx, stdout, flags)
	case "sync":
		return a.runSync(ctx, stdout, flags, cmdArgs)
	case "status":
		return a.runStatus(ctx, stdout, flags)
	case "search":
		return a.runSearch(ctx, stdout, flags, cmdArgs)
	case "sql":
		return a.runSQL(ctx, stdout, flags, cmdArgs)
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
	cfg, defaultPath, err := wconfig.Defaults()
	if err != nil {
		return err
	}
	path, err := wconfig.App().ResolveConfigPath(flags.ConfigPath)
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
	if err := wconfig.EnsureDirs(cfg); err != nil {
		return err
	}
	if err := wconfig.Save(path, cfg); err != nil {
		return err
	}
	result := map[string]string{"config_path": path, "db_path": cfg.DBPath}
	if flags.JSON {
		return writeEnvelope(w, result)
	}
	printKV(w, "config", path)
	printKV(w, "database", cfg.DBPath)
	return nil
}

func (a App) runDoctor(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	_ = ctx
	cfg, configPath, err := wconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	sourceDB, _ := flagValue(args, "--source-db")
	report := doctor.Run(cfg, configPath, sourceDB)
	if flags.JSON {
		return writeEnvelope(w, report)
	}
	printKV(w, "config", report.ConfigPath)
	printKV(w, "source_db", report.SourceDB)
	printKV(w, "source_exists", report.Source.Exists)
	printKV(w, "archive", report.DBPath)
	return nil
}

func (a App) runMetadata(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	_ = ctx
	cfg, configPath, err := wconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	manifest := controlManifest(configPath, cfg)
	if flags.JSON {
		return writeJSON(w, manifest)
	}
	printKV(w, "id", manifest.ID)
	printKV(w, "binary", manifest.Binary.Name)
	return nil
}

func (a App) runSync(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	cfg, _, err := wconfig.Load(flags.ConfigPath)
	if err != nil {
		return err
	}
	sourceDB, _ := flagValue(args, "--source-db")
	result, err := sync.Run(ctx, cfg, sync.Options{
		SourceDB:       sourceDB,
		CrawlerVersion: buildinfo.Current().Version,
	})
	if sync.IsUnsupportedSchema(err) {
		if flags.JSON {
			_ = writeJSON(w, Envelope{OK: false, Result: result, Error: string(result.Code)})
		} else {
			printKV(w, "code", result.Code)
			printKV(w, "detail", result.Detail)
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
		return writeEnvelope(w, result)
	}
	printKV(w, "code", result.Code)
	printKV(w, "artifacts", result.Artifacts)
	printKV(w, "source_db", result.SourceDB)
	return nil
}

func (a App) runStatus(ctx context.Context, w io.Writer, flags GlobalFlags) error {
	rt, err := wruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	status, err := rt.Store.Status(ctx)
	if err != nil {
		return err
	}
	if flags.JSON {
		return writeJSON(w, controlStatus(rt.ConfigPath, rt.Config, status))
	}
	printKV(w, "database", rt.Store.Path())
	printKV(w, "artifacts", status.Artifacts)
	printKV(w, "sync_runs", status.SyncRuns)
	return nil
}

func (a App) runSearch(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	query := strings.TrimSpace(strings.Join(nonFlagArgs(args), " "))
	if query == "" {
		return fmt.Errorf("usage: whispcrawl search <query> [--limit <n>]")
	}
	rt, err := wruntime.Open(ctx, flags.ConfigPath)
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
		fmt.Fprintf(w, "%s\t%s\t%s\n", hit.SourceID, hit.Fidelity, hit.Snippet)
	}
	return nil
}

func (a App) runSQL(ctx context.Context, w io.Writer, flags GlobalFlags, args []string) error {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" || query == "-" {
		data, err := io.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			return err
		}
		query = strings.TrimSpace(string(data))
	}
	if query == "" {
		return fmt.Errorf("sql query required")
	}
	rt, err := wruntime.Open(ctx, flags.ConfigPath)
	if err != nil {
		return err
	}
	defer rt.Close()
	result, err := rt.Store.QueryReadOnly(ctx, query)
	if err != nil {
		return err
	}
	if flags.JSON {
		return writeEnvelope(w, result)
	}
	for _, row := range result.Values {
		parts := make([]string, 0, len(result.Columns))
		for _, col := range result.Columns {
			parts = append(parts, fmt.Sprint(row[col]))
		}
		fmt.Fprintln(w, strings.Join(parts, "\t"))
	}
	return nil
}

func nonFlagArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--limit" || args[i] == "--source-db" {
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
