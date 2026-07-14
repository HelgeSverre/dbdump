package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveProfilesRoundTripsAndEnforces0600(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	profiles := &ProfilesConfig{}
	profiles.Upsert(ConnectionProfile{
		Name: "prod", Host: "db.example.com", Port: 3306,
		User: "readonly", Password: "s3cret", Database: "mydb",
	})

	if err := SaveProfiles(profiles); err != nil {
		t.Fatalf("SaveProfiles returned error: %v", err)
	}

	path, err := GetProfilesPath()
	if err != nil {
		t.Fatalf("GetProfilesPath returned error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("expected profiles file mode 0600, got %o", perm)
	}

	reloaded, err := LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles returned error: %v", err)
	}
	got, ok := reloaded.Find("prod")
	if !ok {
		t.Fatal("expected saved profile to be found")
	}
	if got.Password != "s3cret" || got.Host != "db.example.com" || got.Database != "mydb" {
		t.Fatalf("unexpected reloaded profile: %#v", got)
	}
}

func TestProfilesUpsertReplacesExistingByName(t *testing.T) {
	t.Parallel()

	profiles := &ProfilesConfig{}
	profiles.Upsert(ConnectionProfile{Name: "prod", Host: "old"})
	profiles.Upsert(ConnectionProfile{Name: "prod", Host: "new"})

	if len(profiles.Profiles) != 1 {
		t.Fatalf("expected upsert to replace, got %d profiles", len(profiles.Profiles))
	}
	if profiles.Profiles[0].Host != "new" {
		t.Fatalf("expected updated host, got %q", profiles.Profiles[0].Host)
	}
}

func TestProfilesRemove(t *testing.T) {
	t.Parallel()

	profiles := &ProfilesConfig{Profiles: []ConnectionProfile{{Name: "a"}, {Name: "b"}}}

	if !profiles.Remove("a") {
		t.Fatal("expected Remove to report an existing profile")
	}
	if _, ok := profiles.Find("a"); ok {
		t.Fatal("expected profile to be gone after Remove")
	}
	if profiles.Remove("missing") {
		t.Fatal("expected Remove to report false for a missing profile")
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
