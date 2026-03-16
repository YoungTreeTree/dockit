package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dockit/internal/config"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// BuildAuth constructs the appropriate go-git AuthMethod from config.
func BuildAuth(auth config.AuthConfig) (transport.AuthMethod, error) {
	switch auth.Type {
	case "ssh":
		return buildSSHAuth(auth)
	case "http":
		return buildHTTPAuth(auth), nil
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown auth type: %q", auth.Type)
	}
}

func buildSSHAuth(auth config.AuthConfig) (transport.AuthMethod, error) {
	keyPath := expandHome(auth.SSHKeyPath)
	if keyPath == "" {
		keyPath = expandHome("~/.ssh/id_rsa")
	}

	if _, err := os.Stat(keyPath); err != nil {
		return nil, fmt.Errorf("SSH key not found at %s: %w", keyPath, err)
	}

	publicKeys, err := ssh.NewPublicKeysFromFile("git", keyPath, auth.SSHKeyPassphrase)
	if err != nil {
		return nil, fmt.Errorf("creating SSH auth: %w", err)
	}

	return publicKeys, nil
}

func buildHTTPAuth(auth config.AuthConfig) transport.AuthMethod {
	username := auth.Username
	if username == "" {
		username = "git"
	}
	return &http.BasicAuth{
		Username: username,
		Password: auth.Token,
	}
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
