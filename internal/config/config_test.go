package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nemial/bx-pack/internal/config"
	"gopkg.in/yaml.v3"
)

func TestApplyDefaults(t *testing.T) {
	cfg := config.Config{}
	cfg = config.ApplyDefaults(cfg)

	if cfg.Module.ID != "example.module" {
		t.Errorf("expected default module id 'example.module', got %q", cfg.Module.ID)
	}
	if cfg.Build.OutputDir != "./dist" {
		t.Errorf("expected default output dir './dist', got %q", cfg.Build.OutputDir)
	}
	if len(cfg.Exclude) == 0 {
		t.Error("expected default exclusions, got empty slice")
	}
}

func TestConfig_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".bxpack.yml")

	cfg := config.Default()
	cfg.Module.ID = "test.module"

	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Module.ID != "test.module" {
		t.Errorf("expected module id 'test.module', got %q", loaded.Module.ID)
	}
}

func TestLoadAndPrepare(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".bxpack.yml")

	cfg := config.Config{}
	cfg.Module.ID = "test.module"
	cfg.Build.SourceDir = "."
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := config.LoadAndPrepare(cfgPath)
	if err != nil {
		t.Fatalf("failed to load and prepare config: %v", err)
	}

	// 1. Проверка применения дефолтов
	if loaded.Module.Name != "Example Module" {
		t.Errorf("expected default module name, got %q", loaded.Module.Name)
	}

	// 2. Проверка нормализации путей
	if !filepath.IsAbs(loaded.Build.SourceDir) {
		t.Errorf("expected absolute sourceDir, got %q", loaded.Build.SourceDir)
	}

	if !filepath.IsAbs(loaded.Build.OutputDir) {
		t.Errorf("expected absolute outputDir, got %q", loaded.Build.OutputDir)
	}
}

func TestConfig_Load_Error(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		_, err := config.Load(filepath.Join(tmpDir, "missing.yml"))
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.yml")
		if err := os.WriteFile(path, []byte("invalid: yaml: :"), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := config.Load(path)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}

func TestGenerateTemplate(t *testing.T) {
	template := config.GenerateTemplate()
	if template == "" {
		t.Error("GenerateTemplate() returned empty string")
	}

	// Простая проверка на наличие ключевых слов и комментариев
	if !strings.Contains(template, "module:") || !strings.Contains(template, "build:") || !strings.Contains(template, "#") {
		t.Error("GenerateTemplate() output does not look like a commented YAML template")
	}

	// Проверка на наличие конкретных пояснений на русском
	russianKeywords := []string{
		"Уникальный идентификатор",
		"установочными скриптами",
		"Паттерны для исключения",
		"Служебные файлы Git",
	}

	for _, kw := range russianKeywords {
		if !strings.Contains(template, kw) {
			t.Errorf("GenerateTemplate() output does not contain expected Russian text: %q", kw)
		}
	}

	// Проверка корректности YAML структуры (должна парситься)
	var cfg config.Config
	err := yaml.Unmarshal([]byte(template), &cfg)
	if err != nil {
		t.Errorf("GenerateTemplate() output is not valid YAML: %v", err)
	}

	if cfg.Module.ID == "" {
		t.Error("GenerateTemplate() output did not contain valid module.id")
	}
}

func TestUpdateModuleVersion(t *testing.T) {
	t.Run("updates only version and preserves comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, ".bxpack.yml")

		content := `# header comment
module:
  id: "vendor.module"
  version: "1.2.3" # keep me
  versionScheme: "semver"
  name: "Модуль"

build:
  sourceDir: "."
  outputDir: "./dist"

exclude:
  - ".git"
`
		if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := config.UpdateModuleVersion(cfgPath, "1.2.4"); err != nil {
			t.Fatalf("UpdateModuleVersion failed: %v", err)
		}

		//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
		updated, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		got := string(updated)

		if !strings.Contains(got, `version: "1.2.4" # keep me`) {
			t.Fatalf("expected updated version with preserved comment, got:\n%s", got)
		}
		if !strings.Contains(got, `# header comment`) {
			t.Fatalf("expected header comment to be preserved, got:\n%s", got)
		}
		if !strings.Contains(got, "build:\n  sourceDir: \".\"") {
			t.Fatalf("expected build section to stay untouched, got:\n%s", got)
		}
		if strings.Contains(got, `version: "1.2.3" # keep me`) {
			t.Fatalf("old version still present, got:\n%s", got)
		}
	})

	t.Run("preserves single quotes", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, ".bxpack.yml")

		content := `module:
 id: "vendor.module"
 version: '1.2.3'
 versionScheme: "semver"
`
		if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := config.UpdateModuleVersion(cfgPath, "1.2.4"); err != nil {
			t.Fatalf("UpdateModuleVersion failed: %v", err)
		}

		//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
		updated, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(updated), `version: '1.2.4'`) {
			t.Fatalf("expected single quotes to be preserved, got:\n%s", string(updated))
		}
	})

	t.Run("returns error when version key is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, ".bxpack.yml")

		content := `module:
 id: "vendor.module"
 versionScheme: "semver"
`
		if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		err := config.UpdateModuleVersion(cfgPath, "1.2.4")
		if err == nil {
			t.Fatal("expected error when module.version is missing")
		}
	})
}
