package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConnectionProfile represents a saved database connection. Profiles are
// display-only: `dbdump config list` shows them, but there is currently no way to
// select a profile for a dump, so no credentials are stored or used here.
type ConnectionProfile struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Database string `yaml:"database,omitempty"`
}

// ProfilesConfig represents the profiles configuration file
type ProfilesConfig struct {
	Profiles []ConnectionProfile `yaml:"profiles"`
}

// GetProfilesPath returns the path to the profiles config file
func GetProfilesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "dbdump")
	return filepath.Join(configDir, "profiles.yaml"), nil
}

// LoadProfiles loads saved connection profiles. Decoding is intentionally lenient
// so profiles.yaml files that still carry legacy keys (such as an unused password)
// keep loading; the extra keys are ignored.
func LoadProfiles() (*ProfilesConfig, error) {
	path, err := GetProfilesPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ProfilesConfig{Profiles: []ConnectionProfile{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles: %w", err)
	}

	var config ProfilesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse profiles: %w", err)
	}

	return &config, nil
}
