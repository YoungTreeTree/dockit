package config

// ServerConfig is the top-level structure for server_config.yaml.
type ServerConfig struct {
	OutputDir string   `yaml:"output_dir"`
	CacheDir  string   `yaml:"cache_dir"`
	Server    Server   `yaml:"server"`
	Defaults  Defaults `yaml:"defaults"`
}

// Server defines HTTP server settings.
type Server struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// Defaults defines default values inherited by all repos.
type Defaults struct {
	Branch    string      `yaml:"branch,omitempty"`
	StartPath string      `yaml:"start_path,omitempty"`
	Patterns  []string    `yaml:"patterns,omitempty"`
	Auth      *AuthConfig `yaml:"auth,omitempty"`
}

// AuthConfig defines authentication for git operations.
type AuthConfig struct {
	Type             string `yaml:"type"`
	SSHKeyPath       string `yaml:"ssh_key_path,omitempty"`
	SSHKeyPassphrase string `yaml:"ssh_key_passphrase,omitempty"`
	Token            string `yaml:"token,omitempty"`
	Username         string `yaml:"username,omitempty"`
}

// ReposConfig is the top-level structure for repos.yaml.
type ReposConfig struct {
	Repos []RepoEntry `yaml:"repos"`
}

// RepoEntry defines a single repository in repos.yaml.
type RepoEntry struct {
	Path      string      `yaml:"path"`
	URL       string      `yaml:"url"`
	Name      string      `yaml:"name,omitempty"`
	Branch    string      `yaml:"branch,omitempty"`
	StartPath string      `yaml:"start_path,omitempty"`
	Patterns  []string    `yaml:"patterns,omitempty"`
	Auth      *AuthConfig `yaml:"auth,omitempty"`
}

// ResolvedRepo is the result after inheritance resolution.
type ResolvedRepo struct {
	Name      string
	URL       string
	Branch    string
	StartPath string
	Auth      AuthConfig
	Patterns  []string
	Path      string // navigation tree path, e.g. "Backend/user-service"
}
