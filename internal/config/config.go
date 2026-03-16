package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadServerConfig reads and parses server_config.yaml, applying defaults.
func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading server config: %w", err)
	}

	var cfg ServerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing server config: %w", err)
	}

	applyServerDefaults(&cfg)

	if err := validateServerConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid server config: %w", err)
	}

	return &cfg, nil
}

// LoadRepos reads and parses repos.yaml.
func LoadRepos(path string) (*ReposConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading repos config: %w", err)
	}

	var cfg ReposConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing repos config: %w", err)
	}

	if err := validateRepos(&cfg); err != nil {
		return nil, fmt.Errorf("invalid repos config: %w", err)
	}

	return &cfg, nil
}

func applyServerDefaults(cfg *ServerConfig) {
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./docs_output"
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = ".dockit_cache"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9090
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Defaults.Branch == "" {
		cfg.Defaults.Branch = "main"
	}
	if len(cfg.Defaults.Patterns) == 0 {
		cfg.Defaults.Patterns = []string{`.*\.md$`}
	}
}

func validateServerConfig(cfg *ServerConfig) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if cfg.Defaults.Auth != nil {
		if err := validateAuth(cfg.Defaults.Auth); err != nil {
			return fmt.Errorf("defaults.auth: %w", err)
		}
	}
	return nil
}

func validateRepos(cfg *ReposConfig) error {
	if len(cfg.Repos) == 0 {
		return fmt.Errorf("repos list must not be empty")
	}
	for i, r := range cfg.Repos {
		if r.URL == "" {
			return fmt.Errorf("repos[%d]: url is required", i)
		}
		if r.Path == "" {
			return fmt.Errorf("repos[%d]: path is required", i)
		}
		if r.Auth != nil {
			if err := validateAuth(r.Auth); err != nil {
				return fmt.Errorf("repos[%d].auth: %w", i, err)
			}
		}
	}
	return nil
}

func validateAuth(auth *AuthConfig) error {
	if auth.Type != "" && auth.Type != "ssh" && auth.Type != "http" {
		return fmt.Errorf("auth.type must be \"ssh\" or \"http\", got %q", auth.Type)
	}
	return nil
}
