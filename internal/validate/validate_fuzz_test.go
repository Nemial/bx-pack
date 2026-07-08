package validate

import (
	"testing"

	"github.com/Nemial/bx-pack/internal/config"
)

func FuzzValidateModuleID(f *testing.F) {
	// Seed corpus with valid and invalid module IDs
	f.Add("vendor.module")
	f.Add("my.custom.id")
	f.Add("a.b")
	f.Add("test.module.123")
	f.Add("")
	f.Add("invalid id!")
	f.Add("example.module")
	f.Add("UPPERCASE.module")
	f.Add("module.")
	f.Add(".module")

	f.Fuzz(func(t *testing.T, moduleID string) {
		cfg := config.Default()
		cfg.Module.ID = moduleID

		// ValidateModuleID should not panic
		issues := ValidateModuleID(cfg)

		// Verify issues are well-formed
		for _, issue := range issues {
			if issue.Code == "" {
				t.Errorf("issue has empty code")
			}
			if issue.Message == "" {
				t.Errorf("issue has empty message")
			}
			if issue.Severity != Error && issue.Severity != Warning && issue.Severity != Info {
				t.Errorf("invalid severity: %v", issue.Severity)
			}
		}
	})
}

func FuzzValidateModuleVersionScheme(f *testing.F) {
	// Seed corpus with valid and invalid schemes
	f.Add("semver")
	f.Add("calver")
	f.Add("year-semver")
	f.Add("custom")
	f.Add("")
	f.Add("invalid")
	f.Add("SEMVER")
	f.Add("SemVer")

	f.Fuzz(func(t *testing.T, scheme string) {
		cfg := config.Default()
		cfg.Module.VersionScheme = scheme

		// ValidateModuleVersionScheme should not panic
		issues := ValidateModuleVersionScheme(cfg)

		// Verify issues are well-formed
		for _, issue := range issues {
			if issue.Code == "" {
				t.Errorf("issue has empty code")
			}
			if issue.Message == "" {
				t.Errorf("issue has empty message")
			}
		}
	})
}

func FuzzValidateExcludePatterns(f *testing.F) {
	// Seed corpus with various exclude patterns
	f.Add(".git")
	f.Add("*.log")
	f.Add("tests")
	f.Add("")
	f.Add("node_modules")
	f.Add(".bxpack")
	f.Add("dist")

	f.Fuzz(func(t *testing.T, pattern string) {
		cfg := config.Default()
		cfg.Exclude = []string{pattern}

		// ValidateExcludePatterns should not panic
		issues := ValidateExcludePatterns(cfg)

		// Verify issues are well-formed
		for _, issue := range issues {
			if issue.Code == "" {
				t.Errorf("issue has empty code")
			}
			if issue.Message == "" {
				t.Errorf("issue has empty message")
			}
		}
	})
}
