package config

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// Resolve merges defaults into each repo entry and returns a flat list of ResolvedRepo.
func Resolve(serverCfg *ServerConfig, reposCfg *ReposConfig) ([]ResolvedRepo, error) {
	var repos []ResolvedRepo

	for _, entry := range reposCfg.Repos {
		r, err := resolveEntry(entry, serverCfg.Defaults)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}

	// check for duplicate names
	seen := make(map[string]bool)
	for _, r := range repos {
		if seen[r.Name] {
			return nil, fmt.Errorf("duplicate repo name: %q", r.Name)
		}
		seen[r.Name] = true
	}

	return repos, nil
}

func resolveEntry(entry RepoEntry, defaults Defaults) (ResolvedRepo, error) {
	name := entry.Name
	if name == "" {
		var err error
		name, err = repoNameFromURL(entry.URL)
		if err != nil {
			return ResolvedRepo{}, fmt.Errorf("cannot derive repo name from URL %q: %w", entry.URL, err)
		}
	}

	branch := defaults.Branch
	if entry.Branch != "" {
		branch = entry.Branch
	}

	startPath := defaults.StartPath
	if entry.StartPath != "" {
		startPath = entry.StartPath
	}

	patterns := defaults.Patterns
	if len(entry.Patterns) > 0 {
		patterns = entry.Patterns
	}

	auth := AuthConfig{}
	if entry.Auth != nil {
		auth = *entry.Auth
	} else if defaults.Auth != nil {
		auth = *defaults.Auth
	}

	return ResolvedRepo{
		Name:      name,
		URL:       entry.URL,
		Branch:    branch,
		StartPath: startPath,
		Auth:      auth,
		Patterns:  patterns,
		Path:      entry.Path,
	}, nil
}

// repoNameFromURL extracts the repository name from a git URL.
func repoNameFromURL(rawURL string) (string, error) {
	// SSH URL: git@host:org/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		if idx := strings.LastIndex(rawURL, "/"); idx >= 0 {
			name := rawURL[idx+1:]
			return strings.TrimSuffix(name, ".git"), nil
		}
		if idx := strings.LastIndex(rawURL, ":"); idx >= 0 {
			name := rawURL[idx+1:]
			return strings.TrimSuffix(name, ".git"), nil
		}
	}

	// HTTP(S) URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return "", fmt.Errorf("cannot extract name from URL path %q", u.Path)
	}
	return strings.TrimSuffix(name, ".git"), nil
}
