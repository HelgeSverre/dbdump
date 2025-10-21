package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConnectionProfile represents a saved database connection
type ConnectionProfile struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password,omitempty"`
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
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "profiles.yaml"), nil
}

// LoadProfiles loads saved connection profiles
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

// SaveProfiles saves connection profiles
func SaveProfiles(config *ProfilesConfig) error {
	path, err := GetProfilesPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write profiles: %w", err)
	}

	return nil
}

// GetProfile retrieves a profile by name
func (pc *ProfilesConfig) GetProfile(name string) (*ConnectionProfile, error) {
	for _, profile := range pc.Profiles {
		if profile.Name == name {
			return &profile, nil
		}
	}
	return nil, fmt.Errorf("profile '%s' not found", name)
}

// AddProfile adds or updates a profile
func (pc *ProfilesConfig) AddProfile(profile ConnectionProfile) {
	// Check if profile already exists and update it
	for i, p := range pc.Profiles {
		if p.Name == profile.Name {
			pc.Profiles[i] = profile
			return
		}
	}

	// Add new profile
	pc.Profiles = append(pc.Profiles, profile)
}

// RemoveProfile removes a profile by name
func (pc *ProfilesConfig) RemoveProfile(name string) error {
	for i, p := range pc.Profiles {
		if p.Name == name {
			pc.Profiles = append(pc.Profiles[:i], pc.Profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile '%s' not found", name)
}
