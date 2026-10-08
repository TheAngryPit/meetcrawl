package detect

import (
	"path/filepath"
	"regexp"
	"strings"
)

var ptGeminiTitle = regexp.MustCompile(`(?i)^Reunião iniciada a \d{4}/\d{2}/\d{2} \d{2}:\d{2} \S+ [-–] Notas do Gemini$`)

// IsGeminiDocTitle reports whether a Drive file name matches Meet/Gemini notes doc titles.
func IsGeminiDocTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	if ptGeminiTitle.MatchString(title) {
		return true
	}
	lower := strings.ToLower(title)
	return strings.HasSuffix(lower, "notes by gemini")
}

// InFolderScope returns true when parentPath is under any configured root (ID or path segment).
func InFolderScope(parentPath string, roots []string) bool {
	if len(roots) == 0 {
		roots = []string{"Google Meet"}
	}
	parentPath = normalizePath(parentPath)
	if parentPath == "" {
		return false
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if LooksLikeFolderID(root) {
			if parentPath == root || strings.HasPrefix(parentPath, root+"/") {
				return true
			}
			continue
		}
		rootNorm := normalizePath(root)
		if parentPath == rootNorm || strings.HasPrefix(parentPath, rootNorm+"/") {
			return true
		}
	}
	return false
}

// LooksLikeFolderID reports whether root looks like a Drive folder id.
func LooksLikeFolderID(root string) bool {
	if len(root) < 20 {
		return false
	}
	for _, r := range root {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	path = filepath.ToSlash(path)
	return path
}
