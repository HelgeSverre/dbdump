package database

import (
	"strings"
	"testing"
)

func TestConnectionDSNIncludesConfiguredTimeouts(t *testing.T) {
	t.Parallel()

	connection := &Connection{
		Host:     "db.example.com",
		Port:     3307,
		User:     "app",
		Password: "secret",
		Database: "testdb",
	}

	dsn := connection.DSN()

	for _, want := range []string{
		"timeout=5s",
		"readTimeout=30s",
		"writeTimeout=30s",
		"parseTime=true",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("expected DSN %q to contain %q", dsn, want)
		}
	}
}
