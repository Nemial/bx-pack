package config

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed templates/config.yml.tmpl
var templatesFS embed.FS

const DefaultConfigPath = ".bxpack.yml"

type Config struct {
	Module  Module   `yaml:"module" omitzero`
	Build   Build    `yaml:"build" omitzero`
	Exclude []string `yaml:"exclude" omitzero`
}

type Module struct {
	ID            string `yaml:"id" omitzero`
	Version       string `yaml:"version" omitzero`
	VersionScheme string `yaml:"versionScheme" omitzero`
	Name          string `yaml:"name" omitzero`
	Install       string `yaml:"install" omitzero`
}

type Build struct {
	SourceDir   string `yaml:"sourceDir" omitzero`
	OutputDir   string `yaml:"outputDir" omitzero`
	StagingDir  string `yaml:"stagingDir" omitzero`
	ArchiveName string `yaml:"archiveName" omitzero`
}

func Default() Config {
	return ApplyDefaults(Config{})
}

func ApplyDefaults(cfg Config) Config {
	if cfg.Module.ID == "" {
		cfg.Module.ID = "example.module"
	}
	if cfg.Module.VersionScheme == "" {
		cfg.Module.VersionScheme = "semver"
	}
	if cfg.Module.Name == "" {
		cfg.Module.Name = "Example Module"
	}
	if cfg.Module.Install == "" {
		cfg.Module.Install = "install"
	}

	if cfg.Build.SourceDir == "" {
		cfg.Build.SourceDir = "."
	}
	if cfg.Build.OutputDir == "" {
		cfg.Build.OutputDir = "./dist"
	}
	if cfg.Build.StagingDir == "" {
		cfg.Build.StagingDir = "./.bxpack/staging"
	}
	if cfg.Build.ArchiveName == "" {
		cfg.Build.ArchiveName = "{module.id}-{module.version}.zip"
	}

	if len(cfg.Exclude) == 0 {
		cfg.Exclude = []string{
			".git",
			"node_modules",
			".bxpack",
			"dist",
		}
	}

	return cfg
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config %q: %w", path, err)
	}

	return cfg, nil
}

// LoadAndPrepare загружает конфигурацию, применяет значения по умолчанию и нормализует пути.
func LoadAndPrepare(path string) (Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return Config{}, err
	}

	cfg = ApplyDefaults(cfg)

	if err := cfg.NormalizePaths(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Save(cfg Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file %q: %w", path, err)
	}

	return nil
}

// NormalizePaths нормализует пути в конфиге, делая их абсолютными.
func (cfg *Config) NormalizePaths() error {
	var err error
	cfg.Build.SourceDir, err = filepath.Abs(cfg.Build.SourceDir)
	if err != nil {
		return fmt.Errorf("normalize sourceDir %q: %w", cfg.Build.SourceDir, err)
	}

	cfg.Build.OutputDir, err = filepath.Abs(cfg.Build.OutputDir)
	if err != nil {
		return fmt.Errorf("normalize outputDir %q: %w", cfg.Build.OutputDir, err)
	}

	cfg.Build.StagingDir, err = filepath.Abs(cfg.Build.StagingDir)
	if err != nil {
		return fmt.Errorf("normalize stagingDir %q: %w", cfg.Build.StagingDir, err)
	}

	return nil
}

func GenerateTemplate() string {
	data, err := templatesFS.ReadFile("templates/config.yml.tmpl")
	if err != nil {
		// В случае ошибки возвращаем пустую строку или паникуем,
		// так как это встроенный ресурс, который обязан быть.
		return ""
	}
	// Мы возвращаем "сырой" шаблон, так как Init его сохраняет как есть.
	// Scaffold же сделает замену moduleID сам, либо мы добавим параметры.
	return string(data)
}

// GenerateForModuleID генерирует конфиг с подставленным moduleID.
func GenerateForModuleID(moduleID string) (string, error) {
	tmplContent := GenerateTemplate()
	if tmplContent == "" {
		return "", fmt.Errorf("config template not found")
	}

	tmpl, err := template.New("config").Parse(tmplContent)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]string{"ModuleID": moduleID})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
