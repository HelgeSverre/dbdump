package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfilesRejectsUnknownKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "dbdump")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	path := filepath.Join(configDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte("profile:\n  - name: prod\n"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := LoadProfiles(); err == nil {
		t.Fatal("expected unknown YAML key to be rejected")
	}
}

func TestLoadProfilesDoesNotCreateConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	profiles, err := LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles returned error: %v", err)
	}

	if len(profiles.Profiles) != 0 {
		t.Fatalf("expected empty profiles, got %#v", profiles.Profiles)
	}

	configDir := filepath.Join(home, ".config", "dbdump")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatalf("expected config directory to remain absent, stat err=%v", err)
	}
}
