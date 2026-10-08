package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/archive"
	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/ingest"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Options struct {
	SourceDir      string
	FixtureDir     string
	CrawlerVersion string
}

type Result struct {
	Code      source.OutcomeCode `json:"code"`
	Detail    string             `json:"detail,omitempty"`
	Artifacts int                `json:"artifacts"`
	SourceDir string             `json:"source_dir,omitempty"`
	Fixture   string             `json:"fixture_dir,omitempty"`
	Outcome   source.SyncOutcome `json:"-"`
}

type UnsupportedSchemaError struct {
	Detail string
}

func (e *UnsupportedSchemaError) Error() string {
	if e == nil || e.Detail == "" {
		return string(source.OutcomeUnsupportedSchema)
	}
	return e.Detail
}

func Run(ctx context.Context, cfg econfig.Config, opts Options) (Result, error) {
	privacy, err := cfg.Privacy()
	if err != nil {
		return Result{}, err
	}

	fixtureDir := strings.TrimSpace(opts.FixtureDir)
	sourceDir := strings.TrimSpace(opts.SourceDir)
	if sourceDir == "" {
		sourceDir = cfg.SourceDir
	}
	if fixtureDir != "" {
		fixtureDir = filepath.Clean(fixtureDir)
	}
	if sourceDir != "" {
		sourceDir = filepath.Clean(sourceDir)
	}

	var rows []ingest.StoredRow
	switch {
	case fixtureDir != "":
		manifest, err := fixture.LoadManifest(fixtureDir)
		if err != nil {
			return failSchema(fixtureDir, err)
		}
		version := opts.CrawlerVersion
		if version == "" {
			version = "exportcrawl-dev"
		}
		rows, err = ingest.BuildFromManifest(fixtureDir, manifest, privacy, version)
		if err != nil {
			return failSchema(fixtureDir, err)
		}
	case sourceDir != "":
		if _, err := os.Stat(sourceDir); err != nil {
			return Result{}, fmt.Errorf("stat source dir: %w", err)
		}
		version := opts.CrawlerVersion
		if version == "" {
			version = "exportcrawl-dev"
		}
		rows, err = ingest.BuildFromSourceDir(sourceDir, privacy, version)
		if err != nil {
			return failSchema(sourceDir, err)
		}
	default:
		return Result{}, fmt.Errorf("source dir is required (--source-dir, config source_dir, or --fixture)")
	}

	version := opts.CrawlerVersion
	if version == "" {
		version = "exportcrawl-dev"
	}

	var artifacts []source.Artifact
	var stored []archive.StoredArtifact
	for _, row := range rows {
		artifacts = append(artifacts, row.Artifact)
		stored = append(stored, archive.StoredArtifact{Artifact: row.Artifact})
	}
	outcome := source.SyncOutcome{Code: source.OutcomeOK, Artifacts: artifacts}
	if err := outcome.Validate(); err != nil {
		return Result{}, err
	}

	finishedAt := time.Now().UTC()
	archiveStore, err := archive.Open(ctx, cfg.DBPath)
	if err != nil {
		return Result{}, err
	}
	defer archiveStore.Close()
	if err := archiveStore.ReplaceArtifacts(ctx, stored, version, finishedAt); err != nil {
		return Result{}, err
	}
	if err := archiveStore.RecordSyncRun(ctx, string(source.OutcomeOK), "", len(artifacts), finishedAt); err != nil {
		return Result{}, err
	}

	return Result{
		Code:      source.OutcomeOK,
		Artifacts: len(artifacts),
		SourceDir: sourceDir,
		Fixture:   fixtureDir,
		Outcome:   outcome,
	}, nil
}

func failSchema(location string, err error) (Result, error) {
	return Result{
		Code:    source.OutcomeUnsupportedSchema,
		Detail:  err.Error(),
		Fixture: location,
	}, &UnsupportedSchemaError{Detail: err.Error()}
}

func IsUnsupportedSchema(err error) bool {
	var target *UnsupportedSchemaError
	return errors.As(err, &target)
}
