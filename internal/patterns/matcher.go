package patterns

import (
	"fmt"
	"path/filepath"

	"github.com/helgesverre/dbdump/internal/config"
)

// Matcher handles table name pattern matching
type Matcher struct {
	exactMatches map[string]bool
	patterns     []string
}

// NewMatcher creates a new Matcher from exclude config. It rejects malformed glob
// patterns up front so a typo (e.g. "secrets[0-9") fails loudly instead of silently
// un-excluding a table the user meant to exclude.
func NewMatcher(excludes config.ExcludeConfig) (*Matcher, error) {
	exactMap := make(map[string]bool)
	for _, exact := range excludes.Exact {
		exactMap[exact] = true
	}

	for _, pattern := range excludes.Patterns {
		if _, err := filepath.Match(pattern, ""); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
	}

	return &Matcher{
		exactMatches: exactMap,
		patterns:     excludes.Patterns,
	}, nil
}

// Matches checks if a table name should be excluded
func (m *Matcher) Matches(tableName string) bool {
	// Check exact matches first (faster)
	if m.exactMatches[tableName] {
		return true
	}

	// Check pattern matches
	for _, pattern := range m.patterns {
		if matchPattern(pattern, tableName) {
			return true
		}
	}

	return false
}

// matchPattern matches a glob-style pattern against a string.
// Supports * and ? wildcards. Patterns are validated in NewMatcher, so an
// ErrBadPattern here is treated as a non-match rather than silently changing
// semantics to a substring check.
func matchPattern(pattern, str string) bool {
	matched, err := filepath.Match(pattern, str)
	if err != nil {
		return false
	}
	return matched
}

// FilterTables returns only tables that should be excluded
func (m *Matcher) FilterTables(tables []string) []string {
	var excluded []string
	for _, table := range tables {
		if m.Matches(table) {
			excluded = append(excluded, table)
		}
	}
	return excluded
}

// FilterIncluded returns only tables that should NOT be excluded (data should be dumped)
func (m *Matcher) FilterIncluded(tables []string) []string {
	var included []string
	for _, table := range tables {
		if !m.Matches(table) {
			included = append(included, table)
		}
	}
	return included
}
