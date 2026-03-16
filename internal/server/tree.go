package server

import (
	"regexp"
	"strings"

	"dockit/internal/config"
	"dockit/internal/sync"
)

// NavTree represents a node in the navigation tree.
type NavTree struct {
	Name     string    `json:"name"`
	Children []NavTree `json:"children,omitempty"`
	Repos    []NavRepo `json:"repos,omitempty"`
}

// NavRepo represents a repository in the navigation tree.
type NavRepo struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

// BuildNavTree constructs the navigation tree from resolved repos,
// using each repo's Path field to determine its position.
// Only document files (matched by patterns) are listed, not asset files.
func BuildNavTree(repos []config.ResolvedRepo, outputDir string) NavTree {
	root := NavTree{Name: "root"}

	for _, repo := range repos {
		parts := strings.Split(repo.Path, "/")
		node := &root

		// The last segment of path is the repo name, only use
		// preceding segments as group nodes to avoid duplication.
		groupParts := parts
		if len(parts) > 1 {
			groupParts = parts[:len(parts)-1]
		}

		for _, part := range groupParts {
			found := false
			for i := range node.Children {
				if node.Children[i].Name == part {
					node = &node.Children[i]
					found = true
					break
				}
			}
			if !found {
				node.Children = append(node.Children, NavTree{Name: part})
				node = &node.Children[len(node.Children)-1]
			}
		}

		allFiles, _ := sync.ListOutputFiles(outputDir, repo.Name)
		docFiles := filterDocFiles(allFiles, repo.Patterns)
		if docFiles == nil {
			docFiles = []string{}
		}

		node.Repos = append(node.Repos, NavRepo{
			Name:  repo.Name,
			Files: docFiles,
		})
	}

	return root
}

// filterDocFiles returns only files that match any of the given regex patterns.
func filterDocFiles(files []string, patterns []string) []string {
	if len(files) == 0 || len(patterns) == 0 {
		return nil
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}

	var result []string
	for _, f := range files {
		for _, re := range compiled {
			if re.MatchString(f) {
				result = append(result, f)
				break
			}
		}
	}
	return result
}
