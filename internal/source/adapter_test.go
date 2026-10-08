package source_test

import (
	"context"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type syntheticAdapter struct {
	kind    source.Kind
	outcome source.SyncOutcome
	err     error
}

func (s syntheticAdapter) Kind() source.Kind { return s.kind }

func (s syntheticAdapter) Sync(context.Context) (source.SyncOutcome, error) {
	return s.outcome, s.err
}

func TestSyntheticAdapterImplementsContract(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC)
	artifact := source.Artifact{
		Identity: source.Identity{
			Kind:     source.KindOpenWhispr,
			SourceID: "note-synthetic-001",
		},
		SourceRevision: "rev-1",
		Fidelity:       source.FidelityTranscript,
		Privacy:        source.PrivacyPrivate,
		NormalizedText: "Synthetic standup transcript for contract test.",
		Window: source.Window{
			Start: start,
			End:   start.Add(30 * time.Minute),
		},
		Language: "en",
	}
	outcome := source.SyncOutcome{
		Code:      source.OutcomeOK,
		Artifacts: []source.Artifact{artifact},
	}
	if err := outcome.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	var _ source.Adapter = syntheticAdapter{kind: source.KindOpenWhispr, outcome: outcome}

	got, err := syntheticAdapter{kind: source.KindOpenWhispr, outcome: outcome}.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() err = %v", err)
	}
	if got.Code != source.OutcomeOK {
		t.Fatalf("Code = %q, want ok", got.Code)
	}
	hash, err := source.ProvenanceForArtifact(got.Artifacts[0], "meetcrawl-test-0.0.0")
	if err != nil {
		t.Fatalf("ProvenanceForArtifact() = %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty provenance hash")
	}
}

func TestExportFileAdapterShape(t *testing.T) {
	t.Parallel()
	// Future export-file adapter: VTT maps to transcript fidelity without contract changes.
	artifact := source.Artifact{
		Identity: source.Identity{
			Kind:     source.KindExportFile,
			SourceID: "sha256:synthetic-vtt-id",
		},
		SourceRevision: "mtime-20260115T140000Z",
		Fidelity:       source.FidelityTranscript,
		Privacy:        source.PrivacyPrivate,
		NormalizedText: "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\nSynthetic caption line.",
	}
	if err := artifact.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if artifact.Kind != source.KindExportFile {
		t.Fatalf("Kind = %q", artifact.Kind)
	}
}

func TestBestFidelity(t *testing.T) {
	t.Parallel()
	got := source.BestFidelity(source.FidelityNotes, source.FidelityTranscript)
	if got != source.FidelityTranscript {
		t.Fatalf("BestFidelity() = %v", got)
	}
}
