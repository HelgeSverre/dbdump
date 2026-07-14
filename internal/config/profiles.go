package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConnectionProfile represents a saved database connection that can be selected
// for a dump or listing with `--profile`. The optional password is stored in the
// profiles file; that file is written with 0600 permissions, but treat it like any
// other credential store and keep it off shared machines.
type ConnectionProfile struct {
	Name          string `yaml:"name"`
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	User          string `yaml:"user"`
	Password      string `yaml:"password,omitempty"`
	Database      string `yaml:"database,omitempty"`
	TLSMode       string `yaml:"tls_mode,omitempty"`
	TLSCAFile     string `yaml:"tls_ca,omitempty"`
	TLSCertFile   string `yaml:"tls_cert,omitempty"`
	TLSKeyFile    string `yaml:"tls_key,omitempty"`
	TLSSkipVerify bool   `yaml:"tls_skip_verify,omitempty"`
	TLSServerName string `yaml:"tls_server_name,omitempty"`
}

// ProfilesConfig represents the profiles configuration file
type ProfilesConfig struct {
	Profiles []ConnectionProfile `yaml:"profiles"`
}

// Find returns the profile with the given name, if present.
func (p *ProfilesConfig) Find(name string) (ConnectionProfile, bool) {
	for _, profile := range p.Profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return ConnectionProfile{}, false
}

// Upsert adds the profile, replacing any existing profile with the same name.
func (p *ProfilesConfig) Upsert(profile ConnectionProfile) {
	for i, existing := range p.Profiles {
		if existing.Name == profile.Name {
			p.Profiles[i] = profile
			return
		}
	}
	p.Profiles = append(p.Profiles, profile)
}

// Remove deletes the named profile, reporting whether it existed.
func (p *ProfilesConfig) Remove(name string) bool {
	for i, profile := range p.Profiles {
		if profile.Name == name {
			p.Profiles = append(p.Profiles[:i], p.Profiles[i+1:]...)
			return true
		}
	}
	return false
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
// so profiles.yaml files that carry unrecognised keys keep loading; extra keys are
// ignored.
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

// SaveProfiles writes the profiles file, creating the config directory as needed
// and enforcing 0600 permissions on the file since it may contain credentials.
func SaveProfiles(profiles *ProfilesConfig) error {
	path, err := GetProfilesPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(profiles)
	if err != nil {
		return fmt.Errorf("failed to encode profiles: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write profiles: %w", err)
	}

	// WriteFile only applies the mode when creating the file; enforce 0600 even
	// when overwriting an existing profiles file with looser permissions.
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to secure profiles file: %w", err)
	}

	return nil
}
