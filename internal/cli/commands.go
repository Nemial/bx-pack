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
	reporter.SetCommand("init")
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

	if !report.IsJSON(reporter) {
		reporter.PrintSuccess(fmt.Sprintf("Создан стандартный шаблон конфигурации: %s", config.DefaultConfigPath))
	} else {
		reporter.PrintSuccess(fmt.Sprintf("Создан шаблон %s", config.DefaultConfigPath))
	}
	return nil
}

func Validate(reporter report.Reporter) error {
	reporter.SetCommand("validate")
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		reporter.PrintConfigError(err)
		return err
	}
	cfg = config.ApplyDefaults(cfg)

	issues := validate.Run(cfg)
	if !report.IsJSON(reporter) || len(issues) > 0 {
		reporter.PrintIssues(issues)
	}

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			reporter.PrintSuccess("Валидация завершилась с ошибками")
			return fmt.Errorf("валидация завершилась с ошибками")
		}
	}

	if len(issues) == 0 {
		reporter.PrintSuccess("Валидация прошла успешно")
	} else {
		reporter.PrintSuccess("Валидация завершена с предупреждениями")
	}

	return nil
}

func Build(reporter report.Reporter) error {
	reporter.SetCommand("build")
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		reporter.PrintConfigError(err)
		return err
	}
	cfg = config.ApplyDefaults(cfg)

	// 1. Validate
	issues := validate.Run(cfg)

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
	if !report.IsJSON(reporter) {
		reporter.PrintInfo("Подготовка временной директории...")
	}
	if err := pack.PrepareStaging(cfg); err != nil {
		err := fmt.Errorf("подготовка staging: %w", err)
		reporter.PrintConfigError(err)
		return err
	}

	// 3. Create archive
	if !report.IsJSON(reporter) {
		reporter.PrintInfo("Создание архива...")
	}
	archivePath, err := pack.CreateArchive(cfg)
	if err != nil {
		err := fmt.Errorf("создание архива: %w", err)
		reporter.PrintConfigError(err)
		return err
	}

	reporter.PrintSummary(archivePath)
	return nil
}
