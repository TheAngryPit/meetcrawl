package cli

import (
	"github.com/TheAngryPit/meetcrawl/internal/adapters/registry"
	"github.com/TheAngryPit/meetcrawl/internal/index/build"
)

// parseArchiveOverrides maps meet index CLI flags to registered source archive paths.
// Legacy crawler binary names (--whispcrawl-db, etc.) remain as migration aliases (docs/SPEC.md §3).
func parseArchiveOverrides(args []string) map[registry.Name]string {
	out := map[registry.Name]string{}
	set := func(name registry.Name, flag string) {
		if v, ok := flagValue(args, flag); ok && v != "" {
			out[name] = v
		}
	}
	set("openwhispr", "--openwhispr-db")
	set("openwhispr", "--whispcrawl-db")
	set("gmeet", "--gmeet-db")
	set("gmeet", "--gmeetcrawl-db")
	set("export-file", "--export-file-db")
	set("export-file", "--exportcrawl-db")
	set("grain", "--grain-db")
	set("grain", "--graincrawl-db")
	set("granola", "--granola-db")
	set("granola", "--granolacrawl-db")
	return out
}

func toBuildSources(srcs []registry.SourceArchive) []build.SourceArchive {
	out := make([]build.SourceArchive, len(srcs))
	for i, s := range srcs {
		out[i] = build.SourceArchive{Kind: s.Kind, Path: s.Path}
	}
	return out
}
