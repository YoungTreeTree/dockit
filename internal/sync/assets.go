package sync

import (
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Regex patterns to extract local asset references from Markdown.
var (
	// ![alt](path) or [text](path)
	mdLinkRe = regexp.MustCompile(`\[(?:[^\]]*)\]\(([^)]+)\)`)
	// <img src="path"> or <img src='path'>
	htmlImgRe = regexp.MustCompile(`<img\s[^>]*src=["']([^"']+)["']`)
)

// ExtractAssets parses matched Markdown files and returns a deduplicated list
// of referenced local asset paths (relative to repo root).
func ExtractAssets(repoDir string, mdFiles []string) []string {
	seen := make(map[string]bool)
	var assets []string

	for _, mdFile := range mdFiles {
		absPath := filepath.Join(repoDir, filepath.FromSlash(mdFile))
		data, err := os.ReadFile(absPath)
		if err != nil {
			slog.Warn("cannot read markdown for asset extraction", "file", mdFile, "error", err)
			continue
		}

		refs := extractRefs(string(data))
		mdDir := path.Dir(mdFile) // use forward-slash path

		for _, ref := range refs {
			if isExternalRef(ref) {
				continue
			}
			// resolve relative to the markdown file's directory
			resolved := path.Clean(path.Join(mdDir, ref))
			if seen[resolved] {
				continue
			}

			// verify the file exists in the repo
			absResolved := filepath.Join(repoDir, filepath.FromSlash(resolved))
			if _, err := os.Stat(absResolved); err != nil {
				slog.Warn("referenced asset not found", "markdown", mdFile, "asset", resolved)
				continue
			}

			seen[resolved] = true
			assets = append(assets, resolved)
		}
	}

	return assets
}

func extractRefs(content string) []string {
	var refs []string
	seen := make(map[string]bool)

	for _, matches := range mdLinkRe.FindAllStringSubmatch(content, -1) {
		ref := strings.TrimSpace(matches[1])
		// strip title if present: path "title"
		if idx := strings.Index(ref, " "); idx > 0 {
			ref = ref[:idx]
		}
		if !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}

	for _, matches := range htmlImgRe.FindAllStringSubmatch(content, -1) {
		ref := strings.TrimSpace(matches[1])
		if !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}

	return refs
}

func isExternalRef(ref string) bool {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "//") {
		return true
	}
	if strings.HasPrefix(ref, "#") {
		return true
	}
	if strings.Contains(ref, "://") {
		return true
	}
	if strings.HasPrefix(ref, "mailto:") || strings.HasPrefix(ref, "data:") {
		return true
	}
	return false
}
