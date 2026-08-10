package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// strictUnmarshal decodes YAML into out, rejecting unknown or misplaced keys so a
// typo (e.g. "excludes" instead of "exclude") surfaces as an error instead of
// silently producing an empty value. An empty document leaves out at its zero value.
func strictUnmarshal(data []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

// defaultConfigYAML is the built-in exclusion configuration.
const defaultConfigYAML = `default_excludes:
  exact:
    - activity_log
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
    - "*_cache"
`

// ExcludeConfig lists exact table names and glob patterns whose data is omitted.
type ExcludeConfig struct {
	Exact    []string `yaml:"exact"`
	Patterns []string `yaml:"patterns"`
}

// Config is the on-disk YAML configuration schema.
type Config struct {
	Name    string        `yaml:"name"`
	Exclude ExcludeConfig `yaml:"exclude"`
}

// DefaultConfig is the built-in configuration document.
type DefaultConfig struct {
	DefaultExcludes ExcludeConfig `yaml:"default_excludes"`
}

// LoadDefaults parses the embedded default exclusions.
func LoadDefaults() (*DefaultConfig, error) {
	var config DefaultConfig
	if err := yaml.Unmarshal([]byte(defaultConfigYAML), &config); err != nil {
		return nil, fmt.Errorf("failed to parse defaults: %w", err)
	}

	return &config, nil
}

// LoadConfig reads a project-specific configuration file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := strictUnmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// MergeExcludes combines exclusion layers and removes duplicates.
func MergeExcludes(defaults *DefaultConfig, project *Config) ExcludeConfig {
	merged := ExcludeConfig{
		Exact:    make([]string, 0),
		Patterns: make([]string, 0),
	}

	if defaults != nil {
		merged.Exact = append(merged.Exact, defaults.DefaultExcludes.Exact...)
		merged.Patterns = append(merged.Patterns, defaults.DefaultExcludes.Patterns...)
	}

	if project != nil {
		merged.Exact = append(merged.Exact, project.Exclude.Exact...)
		merged.Patterns = append(merged.Patterns, project.Exclude.Patterns...)
	}

	merged.Exact = UniqueStrings(merged.Exact)
	merged.Patterns = UniqueStrings(merged.Patterns)

	return merged
}

// UniqueStrings removes duplicate strings from a slice while preserving order.
func UniqueStrings(input []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, str := range input {
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}

	return result
}

// GetGlobalConfigPath returns the conventional global configuration path.
func GetGlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".dbdump.yaml"), nil
}

// LoadGlobalConfig loads the global user configuration. A missing file returns
// (nil, nil) because global configuration is optional.
func LoadGlobalConfig() (*Config, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	return LoadConfig(configPath)
}
