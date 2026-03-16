package sync

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
)

// ScanRepo walks the repository directory (starting from startPath) and returns
// relative paths (from repo root) of files matching any pattern.
func ScanRepo(repoDir string, startPath string, patterns []string) ([]string, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("compiling pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}

	scanRoot := repoDir
	if startPath != "" {
		scanRoot = filepath.Join(repoDir, startPath)
	}

	var matched []string
	err := filepath.WalkDir(scanRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		// relative path from repo root (preserving start_path prefix)
		rel, err := filepath.Rel(repoDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		for _, re := range compiled {
			if re.MatchString(rel) {
				matched = append(matched, rel)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	return matched, nil
}

// ListOutputFiles lists all files under outputDir/<repoName>/ and returns
// their relative paths from the repo output root.
func ListOutputFiles(outputDir, repoName string) ([]string, error) {
	repoOut := filepath.Join(outputDir, repoName)
	var files []string

	err := filepath.WalkDir(repoOut, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // directory might not exist
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(repoOut, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, nil // not found is ok
	}

	return files, nil
}
