package sync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	gosync "sync"
	"time"

	"dockit/internal/config"
	"dockit/internal/git"
)

// RepoResult holds the sync result for a single repository.
type RepoResult struct {
	Name       string `json:"name"`
	FilesCount int    `json:"files_count"`
	Error      string `json:"error,omitempty"`
}

// SyncStatus holds the overall sync state.
type SyncStatus struct {
	Running  bool         `json:"running"`
	LastSync *time.Time   `json:"last_sync,omitempty"`
	Results  []RepoResult `json:"results,omitempty"`
	Error    string       `json:"error,omitempty"`
}

// Syncer manages concurrent repository synchronization.
type Syncer struct {
	cfg   *config.ServerConfig
	repos []config.ResolvedRepo

	mu     gosync.RWMutex
	status SyncStatus
}

// NewSyncer creates a new Syncer.
func NewSyncer(cfg *config.ServerConfig, repos []config.ResolvedRepo) *Syncer {
	return &Syncer{
		cfg:   cfg,
		repos: repos,
	}
}

// Run executes a full sync: clone/pull all repos, scan, extract assets, and copy.
func (s *Syncer) Run(ctx context.Context) error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	s.status = SyncStatus{Running: true}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.status.Running = false
		now := time.Now()
		s.status.LastSync = &now
		s.mu.Unlock()
	}()

	if err := os.MkdirAll(s.cfg.CacheDir, 0o755); err != nil {
		s.setError(fmt.Sprintf("creating cache dir: %v", err))
		return err
	}

	if err := os.RemoveAll(s.cfg.OutputDir); err != nil {
		s.setError(fmt.Sprintf("cleaning output dir: %v", err))
		return err
	}
	if err := os.MkdirAll(s.cfg.OutputDir, 0o755); err != nil {
		s.setError(fmt.Sprintf("creating output dir: %v", err))
		return err
	}

	const maxWorkers = 4
	jobs := make(chan config.ResolvedRepo, len(s.repos))
	resultsCh := make(chan RepoResult, len(s.repos))

	workers := maxWorkers
	if len(s.repos) < workers {
		workers = len(s.repos)
	}

	var wg gosync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range jobs {
				resultsCh <- s.syncRepo(ctx, repo)
			}
		}()
	}

	for _, repo := range s.repos {
		jobs <- repo
	}
	close(jobs)

	wg.Wait()
	close(resultsCh)

	var results []RepoResult
	var failed, succeeded int
	for r := range resultsCh {
		results = append(results, r)
		if r.Error != "" {
			failed++
		} else {
			succeeded++
		}
	}

	s.mu.Lock()
	s.status.Results = results
	if failed > 0 {
		s.status.Error = fmt.Sprintf("%d/%d repos failed to sync", failed, failed+succeeded)
	}
	s.mu.Unlock()

	slog.Info("sync completed", "total", len(s.repos), "succeeded", succeeded, "failed", failed)

	if failed > 0 {
		return fmt.Errorf("%d repos failed to sync", failed)
	}
	return nil
}

func (s *Syncer) syncRepo(ctx context.Context, repo config.ResolvedRepo) RepoResult {
	result := RepoResult{Name: repo.Name}

	// clone or pull
	if err := git.CloneOrPull(ctx, repo, s.cfg.CacheDir); err != nil {
		result.Error = err.Error()
		slog.Error("sync failed", "repo", repo.Name, "error", err)
		return result
	}

	repoDir := filepath.Join(s.cfg.CacheDir, repo.Name)

	// scan for matching document files
	docFiles, err := ScanRepo(repoDir, repo.StartPath, repo.Patterns)
	if err != nil {
		result.Error = err.Error()
		slog.Error("scan failed", "repo", repo.Name, "error", err)
		return result
	}

	// extract referenced assets from matched markdown files
	assets := ExtractAssets(repoDir, docFiles)
	slog.Info("extracted assets", "repo", repo.Name, "docs", len(docFiles), "assets", len(assets))

	// merge doc files and assets, deduplicate
	allFiles := dedup(append(docFiles, assets...))

	// copy to output
	if err := CopyFiles(repo.Name, repoDir, allFiles, s.cfg.OutputDir); err != nil {
		result.Error = err.Error()
		slog.Error("copy failed", "repo", repo.Name, "error", err)
		return result
	}

	result.FilesCount = len(allFiles)
	slog.Info("synced repository", "repo", repo.Name, "files", len(allFiles))
	return result
}

func (s *Syncer) setError(msg string) {
	s.mu.Lock()
	s.status.Error = msg
	s.mu.Unlock()
}

// Status returns the current sync status.
func (s *Syncer) Status() SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func dedup(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
