package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Default configuration as embedded constant
const defaultConfigYAML = `default_excludes:
  exact:
    - audits
    - sessions
    - cache
    - cache_locks
    - failed_jobs
    - telescope_entries
    - telescope_entries_tags
    - telescope_monitoring
    - pulse_entries
    - pulse_aggregates
  patterns:
    - "telescope_*"
    - "pulse_*"
    - "_cache"
`

// ExcludeConfig represents the exclude configuration
type ExcludeConfig struct {
	Exact    []string `yaml:"exact"`
	Patterns []string `yaml:"patterns"`
}

// Config represents the full configuration
type Config struct {
	Name    string        `yaml:"name"`
	Exclude ExcludeConfig `yaml:"exclude"`
}

// DefaultConfig represents the default excludes
type DefaultConfig struct {
	DefaultExcludes ExcludeConfig `yaml:"default_excludes"`
}

// LoadDefaults loads the default exclude patterns
func LoadDefaults() (*DefaultConfig, error) {
	var config DefaultConfig
	if err := yaml.Unmarshal([]byte(defaultConfigYAML), &config); err != nil {
		return nil, fmt.Errorf("failed to parse defaults: %w", err)
	}

	return &config, nil
}

// LoadConfig loads a project-specific configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// MergeExcludes merges default excludes with project-specific excludes
func MergeExcludes(defaults *DefaultConfig, project *Config) ExcludeConfig {
	merged := ExcludeConfig{
		Exact:    make([]string, 0),
		Patterns: make([]string, 0),
	}

	// Add defaults
	if defaults != nil {
		merged.Exact = append(merged.Exact, defaults.DefaultExcludes.Exact...)
		merged.Patterns = append(merged.Patterns, defaults.DefaultExcludes.Patterns...)
	}

	// Add project-specific (avoiding duplicates)
	if project != nil {
		merged.Exact = append(merged.Exact, project.Exclude.Exact...)
		merged.Patterns = append(merged.Patterns, project.Exclude.Patterns...)
	}

	// Remove duplicates
	merged.Exact = uniqueStrings(merged.Exact)
	merged.Patterns = uniqueStrings(merged.Patterns)

	return merged
}

// uniqueStrings removes duplicate strings from a slice
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, str := range input {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// GetGlobalConfigPath returns the path to the global config file
func GetGlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "dbdump", "config.yaml"), nil
}

// LoadGlobalConfig loads the global user config file if it exists
// Returns nil if the file doesn't exist (which is not an error)
func LoadGlobalConfig() (*Config, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// File doesn't exist, return nil (not an error)
		return nil, nil
	}

	// File exists, load it
	return LoadConfig(configPath)
}
