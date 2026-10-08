package sync

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/api"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/archive"
	gconfig "github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/ingest"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Options struct {
	FixtureDir     string
	CrawlerVersion string
}

type Result struct {
	Code      source.OutcomeCode `json:"code"`
	Detail    string             `json:"detail,omitempty"`
	Artifacts int                `json:"artifacts"`
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

func Run(ctx context.Context, cfg gconfig.Config, opts Options) (Result, error) {
	fixtureDir := strings.TrimSpace(opts.FixtureDir)
	if fixtureDir != "" {
		fixtureDir = filepath.Clean(fixtureDir)
	}

	client, err := api.New(ctx, cfg, fixtureDir)
	if err != nil {
		if isFixtureLayoutError(err) {
			return failSchema(fixtureDir, err)
		}
		return Result{}, err
	}

	version := opts.CrawlerVersion
	if version == "" {
		version = "gmeetcrawl-dev"
	}

	rows, driveRows, err := ingest.BuildArtifacts(ctx, client, cfg, version)
	if err != nil {
		if isDocLayoutError(err) {
			return failSchema(fixtureDir, err)
		}
		return Result{}, err
	}

	var artifacts []source.Artifact
	var stored []archive.StoredArtifact
	for _, row := range rows {
		artifacts = append(artifacts, row.Artifact)
		stored = append(stored, archive.StoredArtifact{
			Artifact:        row.Artifact,
			CalendarEventID: row.CalendarEventID,
			Participants:    row.Participants,
			IngestFlags:     row.IngestFlags,
		})
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
	var driveFiles []archive.DriveFileRecord
	for _, d := range driveRows {
		driveFiles = append(driveFiles, archive.DriveFileRecord{
			FileID:       d.FileID,
			RevisionID:   d.RevisionID,
			ModifiedTime: d.ModifiedTime,
		})
	}
	if err := archiveStore.ReplaceDriveFiles(ctx, driveFiles, finishedAt); err != nil {
		return Result{}, err
	}
	if err := archiveStore.RecordSyncRun(ctx, string(source.OutcomeOK), "", len(artifacts), finishedAt); err != nil {
		return Result{}, err
	}

	return Result{
		Code:      source.OutcomeOK,
		Artifacts: len(artifacts),
		Fixture:   fixtureDir,
		Outcome:   outcome,
	}, nil
}

func failSchema(fixtureDir string, err error) (Result, error) {
	return Result{
		Code:    source.OutcomeUnsupportedSchema,
		Detail:  err.Error(),
		Fixture: fixtureDir,
	}, &UnsupportedSchemaError{Detail: err.Error()}
}

func isFixtureLayoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "fixture layout version") ||
		strings.Contains(msg, "fixture manifest") ||
		strings.Contains(msg, "drive_files")
}

func isDocLayoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unrecognized Meet/Gemini doc title") ||
		strings.Contains(msg, "unsupported_schema") ||
		strings.Contains(msg, "gemini export") ||
		strings.Contains(msg, "missing notes tab marker")
}

func IsUnsupportedSchema(err error) bool {
	var target *UnsupportedSchemaError
	return errors.As(err, &target)
}
