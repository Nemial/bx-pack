package cli

import (
	"bx-pack/internal/config"
	"bx-pack/internal/pack"
	"bx-pack/internal/report"
	"bx-pack/internal/validate"
	"fmt"
	"os"
)

func Init() error {
	if _, err := os.Stat(config.DefaultConfigPath); err == nil {
		return fmt.Errorf("файл конфигурации %q уже существует", config.DefaultConfigPath)
	}

	content := config.GenerateTemplate()
	if err := os.WriteFile(config.DefaultConfigPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("ошибка записи шаблона конфигурации: %w", err)
	}

	fmt.Printf("Создан стандартный шаблон конфигурации: %s\n", config.DefaultConfigPath)
	return nil
}

func Validate(format report.Format) error {
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return err
	}
	cfg = config.ApplyDefaults(cfg)

	reporter := report.NewReporter(format)
	issues := validate.Run(cfg)
	reporter.PrintIssues(issues)

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			return fmt.Errorf("валидация завершилась с ошибками")
		}
	}

	return nil
}

func Build(format report.Format) error {
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		return err
	}
	cfg = config.ApplyDefaults(cfg)

	reporter := report.NewReporter(format)

	// 1. Validate
	issues := validate.Run(cfg)
	if format == report.TextFormat {
		reporter.PrintIssues(issues)
	}

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			if format == report.JSONFormat {
				reporter.PrintIssues(issues)
			}
			return fmt.Errorf("сборка невозможна: валидация завершилась с ошибками")
		}
	}

	// 2. Prepare staging
	if format == report.TextFormat {
		fmt.Println("Подготовка временной директории...")
	}
	if err := pack.PrepareStaging(cfg); err != nil {
		return fmt.Errorf("подготовка staging: %w", err)
	}

	// 3. Create archive
	if format == report.TextFormat {
		fmt.Println("Создание архива...")
	}
	archivePath, err := pack.CreateArchive(cfg)
	if err != nil {
		return fmt.Errorf("создание архива: %w", err)
	}

	reporter.PrintSummary(archivePath)
	return nil
}
