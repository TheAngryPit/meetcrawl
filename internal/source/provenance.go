package source

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ContentHash returns the normalized-content digest used for dedup and provenance.
func ContentHash(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// ProvenanceInput is the tuple hashed into provenance_hash (docs/SPEC.md section 6).
type ProvenanceInput struct {
	Source         string
	SourceID       string
	SourceRevision string
	ContentHash    string
	CrawlerVersion string
}

func (p ProvenanceInput) validate() error {
	if strings.TrimSpace(p.Source) == "" {
		return fmt.Errorf("source: empty provenance source")
	}
	if strings.TrimSpace(p.SourceID) == "" {
		return fmt.Errorf("source: empty provenance source_id")
	}
	if strings.TrimSpace(p.ContentHash) == "" {
		return fmt.Errorf("source: empty content_hash")
	}
	if strings.TrimSpace(p.CrawlerVersion) == "" {
		return fmt.Errorf("source: empty crawler version")
	}
	return nil
}

// ProvenanceHash returns sha256(source, source_id, source_revision, content_hash, crawler version).
func ProvenanceHash(p ProvenanceInput) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	payload := strings.Join([]string{
		p.Source,
		p.SourceID,
		p.SourceRevision,
		p.ContentHash,
		p.CrawlerVersion,
	}, "\x1e")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}

// ProvenanceForArtifact builds provenance for a validated artifact.
func ProvenanceForArtifact(a Artifact, crawlerVersion string) (string, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	return ProvenanceHash(ProvenanceInput{
		Source:         a.Kind.String(),
		SourceID:       a.SourceID,
		SourceRevision: a.SourceRevision,
		ContentHash:    ContentHash(a.NormalizedText),
		CrawlerVersion: crawlerVersion,
	})
}
