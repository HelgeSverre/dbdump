package patterns

import (
	"reflect"
	"testing"

	"github.com/helgesverre/dbdump/internal/config"
)

func TestMatcherMatchesSuffixPattern(t *testing.T) {
	t.Parallel()

	matcher := NewMatcher(config.ExcludeConfig{
		Patterns: []string{"*_cache"},
	})

	if !matcher.Matches("redis_cache") {
		t.Fatal("expected suffix glob to match cache table")
	}

	if matcher.Matches("cache_table") {
		t.Fatal("did not expect suffix glob to match non-suffix table")
	}
}

func TestFilterIncludedExcludesConfiguredTables(t *testing.T) {
	t.Parallel()

	matcher := NewMatcher(config.ExcludeConfig{
		Exact:    []string{"audits"},
		Patterns: []string{"temp_*"},
	})

	got := matcher.FilterIncluded([]string{"users", "audits", "temp_jobs"})
	want := []string{"users"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected included tables: got %v want %v", got, want)
	}
}
