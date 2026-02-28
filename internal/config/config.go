package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RepoConfig represents a single tracked repository.
type RepoConfig struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
	Notes string `yaml:"notes,omitempty"`
}

// Config is the root config file structure.
type Config struct {
	Repos []RepoConfig `yaml:"repos"`
}

// DefaultPath returns the default config file path (~/.config/reprac/repos.yaml).
func DefaultPath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = os.Getenv("HOME")
	}
	return filepath.Join(cfgDir, "reprac", "repos.yaml")
}

// Load reads and parses a config YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s\n\nCreate it with:\n\nrepos:\n  - owner: your-org\n    repo: your-repo\n    notes: \"Optional description\"", path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config back to disk.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// InitExample creates a sample config file.
func InitExample(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	content := `# reprac config — list of repos to track
# Each entry must have owner and repo. notes is optional.
repos:
  - owner: your-org
    repo: your-app
    notes: "Production app"
  - owner: your-org
    repo: your-api
    notes: "Backend API"
`
	return os.WriteFile(path, []byte(content), 0644)
}
