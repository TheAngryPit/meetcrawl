package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/fixture"
	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/parse"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type FileRef struct {
	ID           string
	Path         string
	ModifiedTime time.Time
}

type StoredRow struct {
	Artifact source.Artifact
}

func BuildFromManifest(dir string, manifest fixture.Manifest, privacy source.PrivacyClass, crawlerVersion string) ([]StoredRow, error) {
	var refs []FileRef
	for _, f := range manifest.Files {
		mod, err := time.Parse(time.RFC3339, f.ModifiedTime)
		if err != nil {
			return nil, err
		}
		refs = append(refs, FileRef{
			ID:           f.ID,
			Path:         fixture.ResolvePath(dir, f.Path),
			ModifiedTime: mod.UTC(),
		})
	}
	return buildRows(refs, privacy, crawlerVersion)
}

func BuildFromSourceDir(dir string, privacy source.PrivacyClass, crawlerVersion string) ([]StoredRow, error) {
	dir = filepath.Clean(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read source dir: %w", err)
	}
	var refs []FileRef
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if strings.EqualFold(name, "manifest.json") {
			continue
		}
		path := filepath.Join(dir, name)
		if _, err := parse.FormatFromPath(path); err != nil {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			return nil, err
		}
		id := sourceIDForPath(name)
		refs = append(refs, FileRef{
			ID:           id,
			Path:         path,
			ModifiedTime: info.ModTime().UTC(),
		})
	}
	if len(refs) == 0 {
		return nil, fmt.Errorf("no supported export files found in source dir")
	}
	return buildRows(refs, privacy, crawlerVersion)
}

func buildRows(refs []FileRef, privacy source.PrivacyClass, crawlerVersion string) ([]StoredRow, error) {
	var rows []StoredRow
	for _, ref := range refs {
		row, err := artifactForFile(ref, privacy, crawlerVersion)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no export artifacts matched ingest rules")
	}
	return rows, nil
}

func artifactForFile(ref FileRef, privacy source.PrivacyClass, crawlerVersion string) (StoredRow, error) {
	format, err := parse.FormatFromPath(ref.Path)
	if err != nil {
		return StoredRow{}, err
	}
	raw, err := readFileReadOnly(ref.Path)
	if err != nil {
		return StoredRow{}, err
	}
	normalized, fidelity, err := parse.Normalize(format, raw)
	if err != nil {
		return StoredRow{}, err
	}
	artifact := source.Artifact{
		Identity: source.Identity{
			Kind:     source.KindExportFile,
			SourceID: ref.ID,
		},
		SourceRevision: ref.ModifiedTime.UTC().Format(time.RFC3339Nano),
		Fidelity:       fidelity,
		Privacy:        privacy,
		NormalizedText: normalized,
		Window: source.Window{
			Start: ref.ModifiedTime,
		},
		Language: "und",
	}
	if err := artifact.Validate(); err != nil {
		return StoredRow{}, err
	}
	if _, err := source.ProvenanceForArtifact(artifact, crawlerVersion); err != nil {
		return StoredRow{}, err
	}
	return StoredRow{Artifact: artifact}, nil
}

func readFileReadOnly(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open export file %s: %w", path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read export file %s: %w", path, err)
	}
	return data, nil
}

func sourceIDForPath(name string) string {
	sum := sha256.Sum256([]byte(filepath.Base(name)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
