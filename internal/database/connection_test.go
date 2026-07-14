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

	dsn, err := connection.DSN()
	if err != nil {
		t.Fatalf("DSN returned error: %v", err)
	}

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

	if strings.Contains(dsn, "tls=") {
		t.Fatalf("expected no tls parameter when TLS is unconfigured, got %q", dsn)
	}
}

func TestConnectionDSNAddsTLSParam(t *testing.T) {
	t.Parallel()

	connection := &Connection{
		Host: "db.example.com", Port: 3306, User: "app", Database: "testdb",
		TLS: TLSConfig{Mode: TLSRequire},
	}

	dsn, err := connection.DSN()
	if err != nil {
		t.Fatalf("DSN returned error: %v", err)
	}
	if !strings.Contains(dsn, "tls=skip-verify") {
		t.Fatalf("expected require mode to map to tls=skip-verify, got %q", dsn)
	}
}
