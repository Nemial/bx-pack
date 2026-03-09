package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = ".bxpack.yml"

type Config struct {
	Module  Module   `yaml:"module" omitzero`
	Build   Build    `yaml:"build" omitzero`
	Exclude []string `yaml:"exclude" omitzero`
}

type Module struct {
	ID      string `yaml:"id" omitzero`
	Version string `yaml:"version" omitzero`
	Name    string `yaml:"name" omitzero`
	Install string `yaml:"install" omitzero`
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
	if cfg.Module.Version == "" {
		cfg.Module.Version = "1.0.0"
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

func GenerateTemplate() string {
	return `# .bxpack.yml - Конфигурация для bx-pack
# Документация: https://github.com/example/bx-pack

module:
  id: "example.module"      # Уникальный идентификатор (например, vendor.name.module)
  version: "1.0.0"          # Семантическая версия (например, 1.0.0, 2.1.0-beta)
  name: "Example Module"    # Человекочитаемое название
  install: "install"        # Директория с установочными скриптами (обычно install)

build:
  sourceDir: "."            # Корневая директория модуля (где лежит .bxpack.yml)
  outputDir: "./dist"       # Директория для сохранения готового архива
  stagingDir: "./.bxpack/staging" # Временная директория для сборки
  archiveName: "{module.id}-{module.version}.zip" # Шаблон имени архива

# Паттерны для исключения из сборки
exclude:
  - ".git"
  - "node_modules"
  - ".bxpack"
  - "dist"
  - ".DS_Store"
`
}
