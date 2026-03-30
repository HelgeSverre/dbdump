package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaultsIncludesCacheSuffixPattern(t *testing.T) {
	t.Parallel()

	defaults, err := LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults returned error: %v", err)
	}

	if !reflect.DeepEqual(defaults.DefaultExcludes.Patterns, []string{"telescope_*", "pulse_*", "*_cache"}) {
		t.Fatalf("unexpected default patterns: %v", defaults.DefaultExcludes.Patterns)
	}
}

func TestMergeExcludesDeduplicatesAndSupplements(t *testing.T) {
	t.Parallel()

	defaults := &DefaultConfig{
		DefaultExcludes: ExcludeConfig{
			Exact:    []string{"audits", "sessions"},
			Patterns: []string{"temp_*"},
		},
	}
	project := &Config{
		Exclude: ExcludeConfig{
			Exact:    []string{"sessions", "jobs"},
			Patterns: []string{"temp_*", "*_cache"},
		},
	}

	got := MergeExcludes(defaults, project)
	want := ExcludeConfig{
		Exact:    []string{"audits", "sessions", "jobs"},
		Patterns: []string{"temp_*", "*_cache"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected merged config: got %#v want %#v", got, want)
	}
}

func TestLoadConfigReturnsErrorForInvalidYAML(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "broken.yaml")
	if err := os.WriteFile(path, []byte("exclude: [:\n"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected invalid YAML to fail")
	}
}

func TestLoadGlobalConfigReturnsReadErrorWhenPathIsDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".dbdump.yaml")
	if err := os.Mkdir(configPath, 0700); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}

	if _, err := LoadGlobalConfig(); err == nil {
		t.Fatal("expected directory path to fail global config loading")
	}
}
