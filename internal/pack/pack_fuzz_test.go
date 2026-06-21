package pack_test

import (
	"testing"

	"bx-pack/internal/pack"
)

func FuzzIsExcluded(f *testing.F) {
	// Seed corpus with typical patterns
	f.Add("config.php", "config.php")
	f.Add("tests/unit/test.php", "tests")
	f.Add("error.log", "*.log")
	f.Add("logs/error.log", "*.log")
	f.Add("temp/cache/file.txt", "temp/*")
	f.Add("install/index.php", "tests")
	f.Add("", "")
	f.Add("file.txt", "")
	f.Add("file.txt", "nonexistent")

	f.Fuzz(func(t *testing.T, relPath, pattern string) {
		// IsExcluded should not panic
		_, _ = pack.IsExcluded(relPath, []string{pattern})
	})
}

func FuzzIsExcludedMultiplePatterns(f *testing.F) {
	// Seed corpus with multiple patterns
	f.Add("config.php", "config.php,*.log,tests")
	f.Add("error.log", "*.log,*.tmp,*.bak")
	f.Add("tests/test.php", "tests,*.log,.git")
	f.Add("clean.txt", "tests,*.log,.git")

	f.Fuzz(func(t *testing.T, relPath, patternsStr string) {
		// Split patterns by comma for testing
		patterns := splitPatterns(patternsStr)

		// IsExcluded should not panic
		_, _ = pack.IsExcluded(relPath, patterns)
	})
}

// splitPatterns splits a comma-separated string into patterns
func splitPatterns(s string) []string {
	if s == "" {
		return nil
	}

	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			pattern := s[start:i]
			if pattern != "" {
				result = append(result, pattern)
			}
			start = i + 1
		}
	}
	return result
}
