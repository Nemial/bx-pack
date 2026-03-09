package config

import (
	"fmt"
	"os"
	"path/filepath"

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
	return `# .bxpack.yml - Конфигурация для bx-pack
# Используется для сборки и валидации модулей Bitrix.

# Раздел описания модуля
module:
  # Уникальный идентификатор модуля (например, "vendor.module.name").
  # Должен состоять из строчных латинских букв, цифр, точек и дефисов.
  id: "myvendor.my-module"

  # Версия модуля в формате SemVer (например, 1.0.0, 2.1.0-beta).
  version: "0.1.0"

  # Название модуля для отображения в отчетах и архиве.
  name: "Мой крутой модуль"

  # Директория с установочными скриптами Bitrix (обычно "install").
  # Будет проверено наличие этой папки в корне проекта.
  install: "install"

# Настройки сборки
build:
  # Корневая директория исходного кода модуля (где находится этот файл).
  sourceDir: "."

  # Директория, куда будет помещен готовый .zip архив.
  outputDir: "./dist"

  # Временная директория, используемая в процессе упаковки.
  stagingDir: "./.bxpack/staging"

  # Шаблон имени архива. Можно использовать {module.id} и {module.version}.
  archiveName: "{module.id}-{module.version}.zip"

# Паттерны для исключения из сборки. 
# Можно указывать файлы и директории относительно корня проекта.
exclude:
  - ".git"          # Служебные файлы Git
  - ".idea"         # Настройки JetBrains IDE
  - ".vscode"       # Настройки VS Code
  - "node_modules"  # Зависимости Node.js
  - ".bxpack"       # Служебная папка bx-pack (всегда исключается автоматически)
  - "dist"          # Папка с результатами сборки (всегда исключается автоматически)
  - "*.log"         # Лог-файлы
  - ".DS_Store"     # Служебные файлы macOS
  - "tests"         # Тесты (если они не должны попасть в модуль)
  - ".gitignore"    # Файл исключений Git
`
}
