package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestApplyDefaults(t *testing.T) {
	cfg := Config{}
	cfg = ApplyDefaults(cfg)

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

	cfg := Default()
	cfg.Module.ID = "test.module"

	if err := Save(cfg, cfgPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Module.ID != "test.module" {
		t.Errorf("expected module id 'test.module', got %q", loaded.Module.ID)
	}
}

func TestConfig_Load_Error(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(tmpDir, "missing.yml"))
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.yml")
		if err := os.WriteFile(path, []byte("invalid: yaml: :"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Load(path)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}

func TestGenerateTemplate(t *testing.T) {
	template := GenerateTemplate()
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
	var cfg Config
	err := yaml.Unmarshal([]byte(template), &cfg)
	if err != nil {
		t.Errorf("GenerateTemplate() output is not valid YAML: %v", err)
	}

	if cfg.Module.ID == "" {
		t.Error("GenerateTemplate() output did not contain valid module.id")
	}
}
