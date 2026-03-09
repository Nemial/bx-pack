package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"bx-pack/internal/config"
)

var (
	reModuleID      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]+[a-z0-9]$`)
	reModuleVersion = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9a-zA-Z.-]+)?(\+[0-9a-zA-Z.-]+)?$`)
)

type Severity string

const (
	Error   Severity = "ERROR"
	Warning Severity = "WARNING"
	Info    Severity = "INFO"
)

const (
	CodeModuleIDInvalid            string = "MODULE_ID_INVALID"
	CodeModuleVersionRequired      string = "MODULE_VERSION_REQUIRED"
	CodeModuleVersionInvalid       string = "MODULE_VERSION_INVALID"
	CodeModuleNameRequired         string = "MODULE_NAME_REQUIRED"
	CodeModuleInstallRequired      string = "MODULE_INSTALL_REQUIRED"
	CodeModuleInstallNotFound      string = "MODULE_INSTALL_NOT_FOUND"
	CodeModuleInstallStatError     string = "MODULE_INSTALL_STAT_ERROR"
	CodeModuleInstallNotDir        string = "MODULE_INSTALL_NOT_DIR"
	CodeBuildSourceDirRequired     string = "BUILD_SOURCE_DIR_REQUIRED"
	CodeBuildSourceDirNotFound     string = "BUILD_SOURCE_DIR_NOT_FOUND"
	CodeBuildSourceDirStatError    string = "BUILD_SOURCE_DIR_STAT_ERROR"
	CodeBuildSourceDirNotDir       string = "BUILD_SOURCE_DIR_NOT_DIR"
	CodeBuildOutputDirRequired     string = "BUILD_OUTPUT_DIR_REQUIRED"
	CodeOutputDirEqualsSourceDir   string = "OUTPUT_DIR_EQUALS_SOURCE_DIR"
	CodeBuildStagingDirRequired    string = "BUILD_STAGING_DIR_REQUIRED"
	CodeStagingDirEqualsOutputDir  string = "STAGING_DIR_EQUALS_OUTPUT_DIR"
	CodeStagingDirEqualsSourceDir  string = "STAGING_DIR_EQUALS_SOURCE_DIR"
	CodeBuildArchiveNameRequired   string = "BUILD_ARCHIVE_NAME_REQUIRED"
	CodeExcludePatternEmpty        string = "EXCLUDE_PATTERN_EMPTY"
	CodeForbiddenPathFound         string = "FORBIDDEN_PATH_FOUND"
	CodeForbiddenPathScanError     string = "FORBIDDEN_PATH_SCAN_ERROR"
	CodeModuleVersionSchemeInvalid string = "MODULE_VERSION_SCHEME_INVALID"
)

type Issue struct {
	Code     string
	Message  string
	Severity Severity
}

func (i Issue) String() string {
	var prefix string
	switch i.Severity {
	case Error:
		prefix = "Ошибка проверки"
	case Warning:
		prefix = "Предупреждение"
	case Info:
		prefix = "Инфо"
	default:
		prefix = string(i.Severity)
	}

	return fmt.Sprintf("%s: %s (%s)", prefix, i.Message, i.Code)
}

type Validator func(cfg config.Config) []Issue

func Run(cfg config.Config) []Issue {
	// Нормализация путей перед валидацией, если возможно
	_ = cfg.NormalizePaths()

	validators := []Validator{
		ValidateModuleID,
		ValidateModuleVersion,
		ValidateModuleVersionScheme,
		ValidateModuleName,
		ValidateModuleInstall,
		ValidateBuildSourceDir,
		ValidateBuildOutputDir,
		ValidateBuildStagingDir,
		ValidateBuildArchiveName,
		ValidateExcludePatterns,
		ValidateForbiddenPaths,
	}

	var allIssues []Issue
	for _, v := range validators {
		allIssues = append(allIssues, v(cfg)...)
	}
	return allIssues
}

func ValidateModuleID(cfg config.Config) []Issue {
	if cfg.Module.ID == "" || cfg.Module.ID == "example.module" {
		return []Issue{{
			Code:     CodeModuleIDInvalid,
			Message:  "module.id должен быть установлен в значение, отличное от стандартного",
			Severity: Error,
		}}
	}

	if !reModuleID.MatchString(cfg.Module.ID) {
		return []Issue{{
			Code:     CodeModuleIDInvalid,
			Message:  fmt.Sprintf("module.id %q содержит недопустимые символы", cfg.Module.ID),
			Severity: Error,
		}}
	}
	return nil
}

func ValidateModuleVersion(cfg config.Config) []Issue {
	if cfg.Module.Version == "" {
		// Если версия не указана в конфиге, она должна быть в install/version.php
		versionFile := filepath.Join(cfg.Build.SourceDir, cfg.Module.Install, "version.php")
		if _, err := os.Stat(versionFile); err != nil {
			return []Issue{{
				Code:     CodeModuleVersionRequired,
				Message:  "поле module.version в конфиге пусто, и файл install/version.php не найден",
				Severity: Error,
			}}
		}
		// Сама валидация содержимого version.php происходит в CLI слое при попытке сборки,
		// здесь мы только подтверждаем, что "источник версии" доступен.
		return nil
	}

	// Для SemVer используем строгую проверку регулярным выражением
	if cfg.Module.VersionScheme == "semver" || cfg.Module.VersionScheme == "" {
		if !reModuleVersion.MatchString(cfg.Module.Version) {
			return []Issue{{
				Code:     CodeModuleVersionInvalid,
				Message:  fmt.Sprintf("module.version %q не соответствует формату семантического версионирования", cfg.Module.Version),
				Severity: Error,
			}}
		}
	} else {
		// Для остальных схем проверяем только наличие 3-х сегментов через точку
		parts := strings.Split(cfg.Module.Version, ".")
		if len(parts) != 3 {
			return []Issue{{
				Code:     CodeModuleVersionInvalid,
				Message:  fmt.Sprintf("module.version %q должен состоять из 3-х сегментов (напр. 1.0.0)", cfg.Module.Version),
				Severity: Error,
			}}
		}
	}
	return nil
}

func ValidateModuleVersionScheme(cfg config.Config) []Issue {
	scheme := strings.ToLower(cfg.Module.VersionScheme)
	if scheme == "" {
		return nil
	}

	validSchemes := map[string]bool{
		"semver":      true,
		"calver":      true,
		"year-semver": true,
		"custom":      true,
	}

	if !validSchemes[scheme] {
		return []Issue{{
			Code:     CodeModuleVersionSchemeInvalid,
			Message:  fmt.Sprintf("неизвестная схема версионирования %q. Доступные: semver, calver, year-semver, custom", cfg.Module.VersionScheme),
			Severity: Error,
		}}
	}
	return nil
}

func ValidateModuleName(cfg config.Config) []Issue {
	if cfg.Module.Name == "" {
		return []Issue{{
			Code:     CodeModuleNameRequired,
			Message:  "поле module.name обязательно для заполнения",
			Severity: Warning,
		}}
	}
	return nil
}

func ValidateModuleInstall(cfg config.Config) []Issue {
	if cfg.Module.Install == "" {
		return []Issue{{
			Code:     CodeModuleInstallRequired,
			Message:  "поле module.install обязательно для заполнения",
			Severity: Error,
		}}
	}

	installPath := filepath.Join(cfg.Build.SourceDir, cfg.Module.Install)
	info, err := os.Stat(installPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Issue{{
				Code:     CodeModuleInstallNotFound,
				Message:  fmt.Sprintf("директория установки %q не найдена", cfg.Module.Install),
				Severity: Error,
			}}
		}
		return []Issue{{
			Code:     CodeModuleInstallStatError,
			Message:  fmt.Sprintf("ошибка при проверке директории установки %q: %v", cfg.Module.Install, err),
			Severity: Warning,
		}}
	}

	if !info.IsDir() {
		return []Issue{{
			Code:     CodeModuleInstallNotDir,
			Message:  fmt.Sprintf("путь установки %q должен быть директорией", cfg.Module.Install),
			Severity: Error,
		}}
	}

	return nil
}

func ValidateBuildSourceDir(cfg config.Config) []Issue {
	if cfg.Build.SourceDir == "" {
		return []Issue{{
			Code:     CodeBuildSourceDirRequired,
			Message:  "поле build.sourceDir обязательно для заполнения",
			Severity: Error,
		}}
	}

	info, err := os.Stat(cfg.Build.SourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Issue{{
				Code:     CodeBuildSourceDirNotFound,
				Message:  fmt.Sprintf("исходная директория %q не найдена", cfg.Build.SourceDir),
				Severity: Error,
			}}
		}
		return []Issue{{
			Code:     CodeBuildSourceDirStatError,
			Message:  fmt.Sprintf("ошибка при проверке исходной директории %q: %v", cfg.Build.SourceDir, err),
			Severity: Warning,
		}}
	}

	if !info.IsDir() {
		return []Issue{{
			Code:     CodeBuildSourceDirNotDir,
			Message:  fmt.Sprintf("исходный путь %q должен быть директорией", cfg.Build.SourceDir),
			Severity: Error,
		}}
	}

	return nil
}

func ValidateBuildOutputDir(cfg config.Config) []Issue {
	if cfg.Build.OutputDir == "" {
		return []Issue{{
			Code:     CodeBuildOutputDirRequired,
			Message:  "поле build.outputDir обязательно для заполнения",
			Severity: Error,
		}}
	}

	absOutput, _ := filepath.Abs(cfg.Build.OutputDir)
	absSource, _ := filepath.Abs(cfg.Build.SourceDir)

	if absOutput == absSource {
		return []Issue{{
			Code:     CodeOutputDirEqualsSourceDir,
			Message:  "outputDir не должен совпадать с sourceDir",
			Severity: Error,
		}}
	}

	return nil
}

func ValidateBuildStagingDir(cfg config.Config) []Issue {
	if cfg.Build.StagingDir == "" {
		return []Issue{{
			Code:     CodeBuildStagingDirRequired,
			Message:  "поле build.stagingDir обязательно для заполнения",
			Severity: Error,
		}}
	}

	// Сравниваем нормализованные пути
	absStaging, _ := filepath.Abs(cfg.Build.StagingDir)
	absOutput, _ := filepath.Abs(cfg.Build.OutputDir)
	absSource, _ := filepath.Abs(cfg.Build.SourceDir)

	if absStaging == absOutput {
		return []Issue{{
			Code:     CodeStagingDirEqualsOutputDir,
			Message:  "stagingDir не должен совпадать с outputDir",
			Severity: Error,
		}}
	}

	if absStaging == absSource {
		return []Issue{{
			Code:     CodeStagingDirEqualsSourceDir,
			Message:  "stagingDir не должен совпадать с sourceDir",
			Severity: Error,
		}}
	}

	return nil
}

func ValidateBuildArchiveName(cfg config.Config) []Issue {
	if cfg.Build.ArchiveName == "" {
		return []Issue{{
			Code:     CodeBuildArchiveNameRequired,
			Message:  "поле build.archiveName обязательно для заполнения",
			Severity: Error,
		}}
	}
	return nil
}

func ValidateExcludePatterns(cfg config.Config) []Issue {
	var issues []Issue
	for _, pattern := range cfg.Exclude {
		if pattern == "" {
			issues = append(issues, Issue{
				Code:     CodeExcludePatternEmpty,
				Message:  "в списке exclude не должно быть пустых строк",
				Severity: Warning,
			})
		}
	}
	return issues
}

func ValidateForbiddenPaths(cfg config.Config) []Issue {
	var issues []Issue
	forbiddenNames := []string{".git", ".idea", ".bxpack", ".DS_Store"}
	forbiddenExts := []string{".log", ".tmp", ".bak"}

	// Проверяем только если sourceDir существует
	if _, err := os.Stat(cfg.Build.SourceDir); err != nil {
		return nil
	}

	err := filepath.Walk(cfg.Build.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Игнорируем ошибки при обходе
		}

		if path == cfg.Build.SourceDir {
			return nil
		}

		relPath, err := filepath.Rel(cfg.Build.SourceDir, path)
		if err != nil {
			return nil
		}

		// Проверяем, не исключен ли уже этот путь
		isExcluded := false
		for _, exc := range cfg.Exclude {
			if relPath == exc || strings.HasPrefix(relPath, exc+string(filepath.Separator)) {
				isExcluded = true
				break
			}
		}

		if isExcluded {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		ext := filepath.Ext(name)

		isForbidden := false
		for _, fn := range forbiddenNames {
			if name == fn {
				isForbidden = true
				break
			}
		}

		if !isForbidden {
			for _, fe := range forbiddenExts {
				if ext == fe {
					isForbidden = true
					break
				}
			}
		}

		if isForbidden {
			issues = append(issues, Issue{
				Code:     CodeForbiddenPathFound,
				Message:  fmt.Sprintf("обнаружен запрещенный путь в исходниках: %s (рекомендуется добавить в exclude)", relPath),
				Severity: Warning,
			})
			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		issues = append(issues, Issue{
			Code:     CodeForbiddenPathScanError,
			Message:  fmt.Sprintf("ошибка при сканировании запрещенных путей: %v", err),
			Severity: Warning,
		})
	}

	return issues
}
