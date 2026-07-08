package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Nemial/bx-pack/internal/config"
	"github.com/Nemial/bx-pack/internal/pack"
	"github.com/Nemial/bx-pack/internal/report"
	"github.com/Nemial/bx-pack/internal/scaffold"
	"github.com/Nemial/bx-pack/internal/validate"
	"github.com/Nemial/bx-pack/internal/version"
)

func Init(reporter report.InitReporter) error {
	reporter.SetCommand("init")
	if _, err := os.Stat(config.DefaultConfigPath); err == nil {
		err := fmt.Errorf("файл конфигурации %q уже существует", config.DefaultConfigPath)
		err = reporter.PrintConfigError(err)
		if err != nil {
			return err
		}
		return NewCLIError(ExitConfigErr, err)
	}

	content := config.GenerateTemplate()
	if err := os.WriteFile(config.DefaultConfigPath, []byte(content), 0o600); err != nil {
		err := fmt.Errorf("ошибка записи шаблона конфигурации: %w", err)
		_ = reporter.PrintConfigError(err)
		return NewCLIError(ExitConfigErr, err)
	}

	_ = reporter.PrintSuccess("Создан стандартный шаблон конфигурации: " + config.DefaultConfigPath)
	return nil
}

func Scaffold(reporter report.ScaffoldReporter, dryRun bool) error {
	return scaffold.Run(reporter, dryRun)
}

func Validate(reporter report.ValidationReporter) error {
	reporter.SetCommand("validate")
	cfg, err := config.LoadAndPrepare(config.DefaultConfigPath)
	if err != nil {
		_ = reporter.PrintConfigError(err)
		return NewCLIError(ExitConfigErr, err)
	}

	issues := validate.RunWithResolvedVersion(&cfg)
	_ = reporter.PrintValidationResult(issues)

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			_ = reporter.PrintSuccess("Валидация завершилась с ошибками")
			return NewCLIError(ExitValError, errors.New("валидация завершилась с ошибками"))
		}
	}

	if len(issues) == 0 {
		_ = reporter.PrintSuccess("Валидация прошла успешно")
	} else {
		_ = reporter.PrintSuccess("Валидация завершена с предупреждениями")
	}

	return nil
}

func Build(reporter report.BuildReporter, dryRun bool) error {
	reporter.SetCommand("build")
	cfg, err := config.LoadAndPrepare(config.DefaultConfigPath)
	if err != nil {
		_ = reporter.PrintConfigError(err)
		return NewCLIError(ExitConfigErr, err)
	}

	// 1. Validate
	issues := validate.RunWithResolvedVersion(&cfg)

	hasErrors := false
	for _, issue := range issues {
		if issue.Severity == validate.Error {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		_ = reporter.PrintIssues(issues)
		return NewCLIError(ExitValError, errors.New("сборка невозможна: валидация завершилась с ошибками"))
	}

	// Выводим предупреждения, если они есть
	if len(issues) > 0 {
		_ = reporter.PrintIssues(issues)
	}

	archivePath := pack.GetArchivePath(cfg)

	if dryRun {
		return reporter.PrintDryRunPlan(cfg, archivePath)
	}

	// 2. Prepare staging
	_ = reporter.PrintInfo("Подготовка временной директории...")
	if err := pack.PrepareStaging(cfg); err != nil {
		err := fmt.Errorf("подготовка staging: %w", err)
		_ = reporter.PrintConfigError(err)
		return NewCLIError(ExitConfigErr, err)
	}

	// 3. Create archive
	_ = reporter.PrintInfo("Создание архива...")
	archivePath, err = pack.CreateArchive(cfg)
	if err != nil {
		err := fmt.Errorf("создание архива: %w", err)
		_ = reporter.PrintConfigError(err)
		return NewCLIError(ExitValError, err)
	}

	_ = reporter.PrintSummary(archivePath)
	return nil
}

func VersionShow(reporter report.VersionReporter) error {
	reporter.SetCommand("version show")
	cfg, err := config.LoadAndPrepare(config.DefaultConfigPath)
	if err != nil {
		_ = reporter.PrintConfigError(fmt.Errorf("загрузка конфигурации: %w", err))
		return NewCLIError(ExitConfigErr, err)
	}

	installPath := filepath.Join(cfg.Build.SourceDir, cfg.Module.Install)
	versionFile := filepath.Join(installPath, "version.php")

	if _, err := os.Stat(versionFile); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("файл версии не найден: %s (проверьте module.install в конфиге)", versionFile)
		} else {
			err = fmt.Errorf("ошибка доступа к файлу версии %s: %w", versionFile, err)
		}
		_ = reporter.PrintConfigError(err)
		return NewCLIError(ExitConfigErr, err)
	}

	ver, err := version.ParseVersion(versionFile)
	if err != nil {
		_ = reporter.PrintConfigError(fmt.Errorf("чтение версии: %w", err))
		return NewCLIError(ExitValError, err)
	}
	_ = reporter.PrintVersion(ver)
	return nil
}

func VersionBump(reporter report.VersionReporter, bumpLevel string) error {
	reporter.SetCommand("version bump " + bumpLevel)
	cfg, err := config.Load(config.DefaultConfigPath) // Здесь Load нужен без ApplyDefaults/Normalize для проверки versionInConfig
	if err != nil {
		_ = reporter.PrintConfigError(err)
		return NewCLIError(ExitConfigErr, err)
	}

	// Запоминаем, была ли версия в конфиге до применения дефолтов
	versionInConfig := cfg.Module.Version

	cfg = config.ApplyDefaults(cfg)
	_ = cfg.NormalizePaths()
	path := filepath.Join(cfg.Build.SourceDir, cfg.Module.Install, "version.php")
	oldVer, newVer, err := version.BumpVersion(path, cfg.Module.VersionScheme, bumpLevel)
	if err != nil {
		_ = reporter.PrintConfigError(fmt.Errorf("обновление версии в %s: %w", path, err))
		return NewCLIError(ExitValError, err)
	}

	// Обновляем .bxpack.yml только если версия там была прописана
	if versionInConfig != "" {
		if err := config.UpdateModuleVersion(config.DefaultConfigPath, newVer); err != nil {
			err = fmt.Errorf("обновление %s: %w", config.DefaultConfigPath, err)
			_ = reporter.PrintConfigError(err)
			return NewCLIError(ExitConfigErr, err)
		}
	}

	_ = reporter.PrintVersionBump(oldVer, newVer)
	return nil
}
