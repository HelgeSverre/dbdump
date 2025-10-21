package patterns

import (
	"path/filepath"
	"strings"

	"github.com/helgesverre/dbdump/internal/config"
)

// Matcher handles table name pattern matching
type Matcher struct {
	exactMatches map[string]bool
	patterns     []string
}

// NewMatcher creates a new Matcher from exclude config
func NewMatcher(excludes config.ExcludeConfig) *Matcher {
	exactMap := make(map[string]bool)
	for _, exact := range excludes.Exact {
		exactMap[exact] = true
	}

	return &Matcher{
		exactMatches: exactMap,
		patterns:     excludes.Patterns,
	}
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

// matchPattern matches a glob-style pattern against a string
// Supports * wildcard (matches any sequence of characters)
func matchPattern(pattern, str string) bool {
	// Use filepath.Match for glob-style matching
	// This supports * and ? wildcards
	matched, err := filepath.Match(pattern, str)
	if err != nil {
		// If pattern is invalid, fall back to simple contains check
		return strings.Contains(str, strings.Trim(pattern, "*"))
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
