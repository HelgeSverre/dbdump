package patterns

import (
	"reflect"
	"testing"

	"github.com/helgesverre/dbdump/internal/config"
)

func TestMatcherMatchesSuffixPattern(t *testing.T) {
	t.Parallel()

	matcher, err := NewMatcher(config.ExcludeConfig{
		Patterns: []string{"*_cache"},
	})
	if err != nil {
		t.Fatalf("NewMatcher returned error: %v", err)
	}

	if !matcher.Matches("redis_cache") {
		t.Fatal("expected suffix glob to match cache table")
	}

	if matcher.Matches("cache_table") {
		t.Fatal("did not expect suffix glob to match non-suffix table")
	}
}

func TestNewMatcherRejectsInvalidPattern(t *testing.T) {
	t.Parallel()

	if _, err := NewMatcher(config.ExcludeConfig{Patterns: []string{"secrets[0-9"}}); err == nil {
		t.Fatal("expected malformed glob pattern to be rejected")
	}
}

func TestFilterTablesExcludesConfiguredTables(t *testing.T) {
	t.Parallel()

	matcher, err := NewMatcher(config.ExcludeConfig{
		Exact:    []string{"audits"},
		Patterns: []string{"temp_*"},
	})
	if err != nil {
		t.Fatalf("NewMatcher returned error: %v", err)
	}

	got := matcher.FilterTables([]string{"users", "audits", "temp_jobs"})
	want := []string{"audits", "temp_jobs"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected excluded tables: got %v want %v", got, want)
	}
}
