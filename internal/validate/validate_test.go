package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bx-pack/internal/config"
)

func TestRun(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "install")
		if err := os.Mkdir(installDir, 0755); err != nil {
			t.Fatal(err)
		}

		cfg := config.Default()
		cfg.Module.ID = "my.custom.id"
		cfg.Build.SourceDir = tmpDir
		cfg.Module.Install = "install"

		issues := Run(cfg)
		for _, issue := range issues {
			if issue.Severity == Error {
				t.Errorf("unexpected error: %v", issue)
			}
		}
	})

	t.Run("sourceDir not found", func(t *testing.T) {
		cfg := config.Default()
		cfg.Module.ID = "my.custom.id"
		cfg.Build.SourceDir = "./non-existent-dir"

		issues := Run(cfg)
		found := false
		for _, issue := range issues {
			if issue.Code == "BUILD_SOURCE_DIR_NOT_FOUND" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected BUILD_SOURCE_DIR_NOT_FOUND error")
		}
	})

	t.Run("install dir not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := config.Default()
		cfg.Module.ID = "my.custom.id"
		cfg.Build.SourceDir = tmpDir
		cfg.Module.Install = "missing-install"

		issues := Run(cfg)
		found := false
		for _, issue := range issues {
			if issue.Code == "MODULE_INSTALL_NOT_FOUND" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected MODULE_INSTALL_NOT_FOUND error")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.Config{}
		issues := Run(cfg)
		hasError := false
		for _, issue := range issues {
			if issue.Severity == Error {
				hasError = true
				break
			}
		}
		if !hasError {
			t.Error("expected errors for empty config, got none")
		}
	})

	t.Run("missing optional name", func(t *testing.T) {
		cfg := config.Default()
		cfg.Module.Name = ""
		issues := Run(cfg)
		hasWarning := false
		for _, issue := range issues {
			if issue.Severity == Warning && issue.Code == "MODULE_NAME_REQUIRED" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected warning for empty module name, got none")
		}
	})

	t.Run("staging equals output", func(t *testing.T) {
		cfg := config.Default()
		cfg.Build.OutputDir = "./dist"
		cfg.Build.StagingDir = "./dist"
		issues := Run(cfg)
		hasError := false
		for _, issue := range issues {
			if issue.Severity == Error && issue.Code == "STAGING_DIR_EQUALS_OUTPUT_DIR" {
				hasError = true
				break
			}
		}
		if !hasError {
			t.Error("expected error when stagingDir equals outputDir")
		}
	})

	t.Run("empty exclude pattern", func(t *testing.T) {
		cfg := config.Default()
		cfg.Exclude = []string{".git", ""}
		issues := Run(cfg)
		hasWarning := false
		for _, issue := range issues {
			if issue.Severity == Warning && issue.Code == "EXCLUDE_PATTERN_EMPTY" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected warning for empty exclude pattern")
		}
	})

	t.Run("forbidden paths found", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Создаем .git директорию
		if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
		// Создаем .idea директорию
		if err := os.Mkdir(filepath.Join(tmpDir, ".idea"), 0755); err != nil {
			t.Fatal(err)
		}
		// Создаем .log файл
		if err := os.WriteFile(filepath.Join(tmpDir, "error.log"), []byte("some log"), 0644); err != nil {
			t.Fatal(err)
		}
		// Создаем разрешенный файл
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("some text"), 0644); err != nil {
			t.Fatal(err)
		}
		// Создаем разрешенную директорию install
		if err := os.Mkdir(filepath.Join(tmpDir, "install"), 0755); err != nil {
			t.Fatal(err)
		}

		cfg := config.Default()
		cfg.Module.ID = "my.custom.id"
		cfg.Build.SourceDir = tmpDir
		cfg.Module.Install = "install"

		issues := Run(cfg)

		expectedCodes := map[string]bool{
			"FORBIDDEN_PATH_FOUND": false,
		}

		for _, issue := range issues {
			if _, ok := expectedCodes[issue.Code]; ok {
				expectedCodes[issue.Code] = true
			}
		}

		for code, found := range expectedCodes {
			if !found {
				t.Errorf("expected issue code %s not found", code)
			}
		}
	})

	t.Run("invalid module id", func(t *testing.T) {
		cfg := config.Default()
		cfg.Module.ID = "invalid id!"
		issues := Run(cfg)
		found := false
		for _, issue := range issues {
			if issue.Code == "MODULE_ID_INVALID" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected MODULE_ID_INVALID error")
		}
	})

	t.Run("invalid module version", func(t *testing.T) {
		cfg := config.Default()
		cfg.Module.Version = "invalid-version"
		issues := Run(cfg)
		found := false
		for _, issue := range issues {
			if issue.Code == "MODULE_VERSION_INVALID" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected MODULE_VERSION_INVALID error")
		}
	})

	t.Run("forbidden paths excluded", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(tmpDir, "install"), 0755); err != nil {
			t.Fatal(err)
		}

		cfg := config.Default()
		cfg.Module.ID = "my.custom.id"
		cfg.Build.SourceDir = tmpDir
		cfg.Module.Install = "install"
		cfg.Exclude = []string{".git"}

		issues := Run(cfg)
		for _, issue := range issues {
			if issue.Code == "FORBIDDEN_PATH_FOUND" {
				t.Errorf("unexpected issue for excluded path: %v", issue)
			}
		}
	})

	t.Run("forbidden nested path", func(t *testing.T) {
		tmpDir := t.TempDir()
		nested := filepath.Join(tmpDir, "some/nested/dir")
		if err := os.MkdirAll(nested, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(nested, "error.log"), []byte("log"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(tmpDir, "install"), 0755); err != nil {
			t.Fatal(err)
		}

		cfg := config.Default()
		cfg.Module.ID = "my.id"
		cfg.Build.SourceDir = tmpDir
		cfg.Module.Install = "install"

		issues := Run(cfg)
		found := false
		for _, issue := range issues {
			if issue.Code == "FORBIDDEN_PATH_FOUND" && strings.Contains(issue.Message, "error.log") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected FORBIDDEN_PATH_FOUND for nested forbidden file")
		}
	})
}
