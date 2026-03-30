package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
