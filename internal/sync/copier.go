package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFiles copies the listed files from repoDir to outputDir/<repoName>/,
// preserving the relative directory structure.
func CopyFiles(repoName, repoDir string, files []string, outputDir string) error {
	for _, relPath := range files {
		src := filepath.Join(repoDir, filepath.FromSlash(relPath))
		dst := filepath.Join(outputDir, repoName, filepath.FromSlash(relPath))

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copying %s: %w", relPath, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
