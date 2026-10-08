package fixture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const LayoutVersion = 1

type Manifest struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`
}

type File struct {
	ID           string `json:"id"`
	Path         string `json:"path"`
	ModifiedTime string `json:"modifiedTime"`
}

func LoadManifest(dir string) (Manifest, error) {
	dir = filepath.Clean(dir)
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Manifest{}, fmt.Errorf("read fixture manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse fixture manifest: %w", err)
	}
	if manifest.Version != LayoutVersion {
		return Manifest{}, fmt.Errorf("fixture layout version %d is unsupported (want %d)", manifest.Version, LayoutVersion)
	}
	if len(manifest.Files) == 0 {
		return Manifest{}, fmt.Errorf("fixture manifest files list is empty")
	}
	for i, f := range manifest.Files {
		if strings.TrimSpace(f.ID) == "" {
			return Manifest{}, fmt.Errorf("fixture files[%d] missing id", i)
		}
		if strings.TrimSpace(f.Path) == "" {
			return Manifest{}, fmt.Errorf("fixture files[%d] missing path", i)
		}
		if strings.TrimSpace(f.ModifiedTime) == "" {
			return Manifest{}, fmt.Errorf("fixture files[%d] missing modifiedTime", i)
		}
		if _, err := time.Parse(time.RFC3339, f.ModifiedTime); err != nil {
			return Manifest{}, fmt.Errorf("fixture files[%d] modifiedTime: %w", i, err)
		}
	}
	return manifest, nil
}

func ResolvePath(dir, rel string) string {
	return filepath.Join(dir, filepath.Clean(rel))
}
