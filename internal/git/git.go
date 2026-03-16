package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"dockit/internal/config"

	gogit "github.com/go-git/go-git/v5"
	gogitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// CloneOrPull clones a repo into cacheDir/<name>/ or pulls if it already exists.
// If the cached repo is corrupted, it removes the directory and re-clones.
func CloneOrPull(ctx context.Context, repo config.ResolvedRepo, cacheDir string) error {
	repoDir := filepath.Join(cacheDir, repo.Name)
	auth, err := BuildAuth(repo.Auth)
	if err != nil {
		return fmt.Errorf("building auth for %s: %w", repo.Name, err)
	}

	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return cloneRepo(ctx, repo, repoDir, auth)
	}

	if err := pullRepo(ctx, repo, repoDir, auth); err != nil {
		slog.Warn("pull failed, re-cloning", "repo", repo.Name, "error", err)
		if removeErr := os.RemoveAll(repoDir); removeErr != nil {
			return fmt.Errorf("removing corrupted cache for %s: %w", repo.Name, removeErr)
		}
		return cloneRepo(ctx, repo, repoDir, auth)
	}
	return nil
}

func cloneRepo(ctx context.Context, repo config.ResolvedRepo, repoDir string, auth transport.AuthMethod) error {
	slog.Info("cloning repository", "repo", repo.Name, "url", repo.URL, "branch", repo.Branch)

	opts := &gogit.CloneOptions{
		URL:           repo.URL,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(repo.Branch),
		Auth:          auth,
	}

	_, err := gogit.PlainCloneContext(ctx, repoDir, false, opts)
	if err != nil {
		return fmt.Errorf("cloning %s: %w", repo.Name, err)
	}

	slog.Info("cloned repository", "repo", repo.Name)
	return nil
}

func pullRepo(ctx context.Context, repo config.ResolvedRepo, repoDir string, auth transport.AuthMethod) error {
	slog.Info("updating repository", "repo", repo.Name)

	r, err := gogit.PlainOpen(repoDir)
	if err != nil {
		return fmt.Errorf("opening repo %s: %w", repo.Name, err)
	}

	fetchOpts := &gogit.FetchOptions{
		RemoteName: "origin",
		Depth:      1,
		Auth:       auth,
		RefSpecs: []gogitconfig.RefSpec{
			gogitconfig.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", repo.Branch, repo.Branch)),
		},
	}

	err = r.FetchContext(ctx, fetchOpts)
	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return fmt.Errorf("fetching %s: %w", repo.Name, err)
	}

	wt, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree for %s: %w", repo.Name, err)
	}

	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewRemoteReferenceName("origin", repo.Branch),
		Force:  true,
	})
	if err != nil {
		return fmt.Errorf("checking out %s: %w", repo.Name, err)
	}

	slog.Info("updated repository", "repo", repo.Name)
	return nil
}
