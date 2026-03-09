package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"bx-pack/internal/config"
	"bx-pack/internal/pack"
	"bx-pack/internal/report"
	"bx-pack/internal/scaffold"
	"bx-pack/internal/validate"
	"bx-pack/internal/version"
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

func Scaffold(reporter report.Reporter, dryRun bool) error {
	return scaffold.Run(reporter, dryRun)
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

func Build(reporter report.Reporter, dryRun bool) error {
	reporter.SetCommand("build")
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		reporter.PrintConfigError(err)
		return err
	}
	cfg = config.ApplyDefaults(cfg)
	_ = cfg.NormalizePaths()

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

	archivePath := pack.GetArchivePath(cfg)

	if dryRun {
		return reporter.PrintDryRunPlan(cfg, archivePath)
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
	archivePath, err = pack.CreateArchive(cfg)
	if err != nil {
		err := fmt.Errorf("создание архива: %w", err)
		reporter.PrintConfigError(err)
		return err
	}

	reporter.PrintSummary(archivePath)
	return nil
}

func VersionShow(reporter report.Reporter) error {
	reporter.SetCommand("version show")
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		reporter.PrintConfigError(fmt.Errorf("загрузка конфигурации: %w", err))
		return err
	}
	cfg = config.ApplyDefaults(cfg)

	installPath := filepath.Join(cfg.Build.SourceDir, cfg.Module.Install)
	versionFile := filepath.Join(installPath, "version.php")

	if _, err := os.Stat(versionFile); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("файл версии не найден: %s (проверьте module.install в конфиге)", versionFile)
		} else {
			err = fmt.Errorf("ошибка доступа к файлу версии %s: %w", versionFile, err)
		}
		reporter.PrintConfigError(err)
		return err
	}

	ver, err := version.ParseVersion(versionFile)
	if err != nil {
		reporter.PrintConfigError(fmt.Errorf("чтение версии: %w", err))
		return err
	}
	reporter.PrintVersion(ver)
	return nil
}

func VersionBump(reporter report.Reporter, bumpLevel string) error {
	reporter.SetCommand(fmt.Sprintf("version bump %s", bumpLevel))
	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		reporter.PrintConfigError(err)
		return err
	}
	cfg = config.ApplyDefaults(cfg)
	path := filepath.Join(cfg.Build.SourceDir, cfg.Module.Install, "version.php")
	oldVer, newVer, err := version.BumpVersion(path, cfg.Module.VersionScheme, bumpLevel)
	if err != nil {
		reporter.PrintConfigError(fmt.Errorf("обновление версии: %w", err))
		return err
	}
	reporter.PrintVersionBump(oldVer, newVer)
	return nil
}
