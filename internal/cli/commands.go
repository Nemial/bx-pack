package cli

import (
	"bx-pack/internal/config"
	"bx-pack/internal/pack"
	"bx-pack/internal/report"
	"bx-pack/internal/validate"
	"fmt"
	"os"
)

func Init(reporter report.Reporter) error {
	if _, err := os.Stat(config.DefaultConfigPath); err == nil {
		err := fmt.Errorf("файл конфигурации %q уже существует", config.DefaultConfigPath)
		reporter.PrintConfigError(err)
		return err
	}

	content := config.GenerateTemplate()
	if err := os.WriteFile(config.DefaultConfigPath, []byte(content), 0644); err != nil {
		err := fmt.Errorf("ошибка записи шаблона конфигурации: %w", err)
		reporter.PrintConfigError(err)
		return err
	}

	reporter.PrintSuccess(fmt.Sprintf("Создан стандартный шаблон конфигурации: %s", config.DefaultConfigPath))
	return nil
}

func Validate(reporter report.Reporter) error {
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		reporter.PrintConfigError(err)
		return err
	}
	cfg = config.ApplyDefaults(cfg)

	issues := validate.Run(cfg)
	reporter.PrintIssues(issues)

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			return fmt.Errorf("валидация завершилась с ошибками")
		}
	}

	return nil
}

func Build(reporter report.Reporter) error {
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		reporter.PrintConfigError(err)
		return err
	}
	cfg = config.ApplyDefaults(cfg)

	// 1. Validate
	issues := validate.Run(cfg)
	// Для JSON выводим ошибки только если они есть и мешают сборке,
	// или если мы хотим видеть весь отчет. В текущей реализации Build
	// при ошибках возвращает ошибку.

	hasErrors := false
	for _, issue := range issues {
		if issue.Severity == validate.Error {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		reporter.PrintIssues(issues)
		return fmt.Errorf("сборка невозможна: валидация завершилась с ошибками")
	}

	// Выводим предупреждения, если они есть
	if len(issues) > 0 {
		reporter.PrintIssues(issues)
	}

	// 2. Prepare staging
	reporter.PrintInfo("Подготовка временной директории...")
	if err := pack.PrepareStaging(cfg); err != nil {
		err := fmt.Errorf("подготовка staging: %w", err)
		reporter.PrintConfigError(err) // Можно использовать PrintConfigError или создать PrintError
		return err
	}

	// 3. Create archive
	reporter.PrintInfo("Создание архива...")
	archivePath, err := pack.CreateArchive(cfg)
	if err != nil {
		err := fmt.Errorf("создание архива: %w", err)
		reporter.PrintConfigError(err)
		return err
	}

	reporter.PrintSummary(archivePath)
	return nil
}
