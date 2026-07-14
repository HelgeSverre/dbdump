package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfilesIgnoresLegacyPasswordKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "dbdump")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	path := filepath.Join(configDir, "profiles.yaml")
	// A pre-existing profile may still carry the now-removed password key; it
	// should load fine with that key simply ignored.
	if err := os.WriteFile(path, []byte("profiles:\n  - name: prod\n    host: db.example.com\n    port: 3306\n    user: readonly\n    password: legacy-secret\n"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	profiles, err := LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles returned error: %v", err)
	}
	if len(profiles.Profiles) != 1 || profiles.Profiles[0].Name != "prod" {
		t.Fatalf("unexpected profiles: %#v", profiles.Profiles)
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
