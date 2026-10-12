package sync

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/fireflies/api"
	farchive "github.com/TheAngryPit/meetcrawl/internal/fireflies/archive"
	fconfig "github.com/TheAngryPit/meetcrawl/internal/fireflies/config"
	"github.com/TheAngryPit/meetcrawl/internal/fireflies/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/fireflies/ingest"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Options struct {
	FixtureDir     string
	CrawlerVersion string
	Secrets        secret.Provider
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

func Run(ctx context.Context, cfg fconfig.Config, opts Options) (Result, error) {
	fixtureDir := strings.TrimSpace(opts.FixtureDir)
	if fixtureDir != "" {
		fixtureDir = filepath.Clean(fixtureDir)
	}
	secrets := opts.Secrets
	if secrets == nil {
		return failClosed(fmt.Errorf("fireflies sync requires a secret provider"))
	}

	client, err := api.New(ctx, cfg, fixtureDir, api.Deps{Secrets: secrets, APIKeyRef: cfg.APIKeyRef})
	if err != nil {
		if errors.Is(err, secret.ErrNotFound) || errors.Is(err, secret.ErrInvalid) {
			return failClosed(err)
		}
		return Result{}, err
	}

	version := opts.CrawlerVersion
	if version == "" {
		version = "fireflyescrawl-dev"
	}

	stored, err := ingest.BuildArtifacts(ctx, client, ingest.Options{CrawlerVersion: version})
	if err != nil {
		if isUnsupported(err) || isUnsupportedPayload(err) {
			return failSchema(fixtureDir, err)
		}
		var auth *api.AuthError
		if errors.As(err, &auth) {
			return failClosed(auth)
		}
		if errors.Is(err, secret.ErrNotFound) || errors.Is(err, secret.ErrInvalid) {
			return failClosed(err)
		}
		return Result{}, err
	}

	artifacts := make([]source.Artifact, 0, len(stored))
	for _, row := range stored {
		artifacts = append(artifacts, row.Artifact)
	}
	outcome := source.SyncOutcome{Code: source.OutcomeOK, Artifacts: artifacts}
	if err := outcome.Validate(); err != nil {
		return Result{}, err
	}

	finishedAt := time.Now().UTC()
	archiveStore, err := farchive.Open(ctx, cfg.DBPath)
	if err != nil {
		return Result{}, err
	}
	defer archiveStore.Close()
	if err := archiveStore.ReplaceArtifacts(ctx, stored, version, finishedAt); err != nil {
		return Result{}, err
	}
	if err := archiveStore.RecordSyncRun(ctx, string(outcome.Code), "", len(artifacts), finishedAt); err != nil {
		return Result{}, err
	}

	res := Result{
		Code:      outcome.Code,
		Artifacts: len(artifacts),
		Fixture:   fixtureDir,
		Outcome:   outcome,
	}
	return res, nil
}

func failSchema(fixtureDir string, err error) (Result, error) {
	detail := err.Error()
	if fixtureDir != "" {
		if _, loadErr := fixture.LoadDir(fixtureDir); loadErr != nil {
			detail = loadErr.Error()
		}
	}
	outcome := source.SyncOutcome{Code: source.OutcomeUnsupportedSchema, Detail: detail}
	return Result{Code: outcome.Code, Detail: detail, Fixture: fixtureDir, Outcome: outcome}, &UnsupportedSchemaError{Detail: detail}
}

func failClosed(err error) (Result, error) {
	detail := err.Error()
	outcome := source.SyncOutcome{Code: source.OutcomeFailed, Detail: detail}
	return Result{Code: outcome.Code, Detail: detail, Outcome: outcome}, fmt.Errorf("%s", detail)
}

func IsUnsupportedSchema(err error) bool {
	var u *UnsupportedSchemaError
	return errors.As(err, &u)
}

func isUnsupported(err error) bool {
	if IsUnsupportedSchema(err) {
		return true
	}
	_, ok := err.(*UnsupportedSchemaError)
	return ok
}

func isUnsupportedPayload(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "sentence[") ||
		strings.Contains(msg, "fixture layout") ||
		strings.Contains(msg, "fixture manifest") ||
		strings.Contains(msg, "transcript missing") ||
		strings.Contains(msg, "missing \"text\"")
}
