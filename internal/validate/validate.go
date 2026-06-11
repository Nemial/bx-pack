package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"bx-pack/internal/config"
	"bx-pack/internal/pack"
	"bx-pack/internal/version"
)

var (
	reModuleID        = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]+[a-z0-9]$`)
	reModuleVersion   = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9a-zA-Z.-]+)?(\+[0-9a-zA-Z.-]+)?$`)
	reInstallModuleID = regexp.MustCompile(`(?m)\$MODULE_ID\s*=\s*['"]([^'"]+)['"]`)
	reVersionDate     = regexp.MustCompile(`(?m)(?:\$VERSION_DATE|["']VERSION_DATE["'])\s*(?:=|=>)\s*["']([^"']+)["']`)
)

type Severity string

const (
	Error   Severity = "ERROR"
	Warning Severity = "WARNING"
	Info    Severity = "INFO"
)

const (
	CodeModuleIDInvalid              string = "MODULE_ID_INVALID"
	CodeModuleVersionRequired        string = "MODULE_VERSION_REQUIRED"
	CodeModuleVersionInvalid         string = "MODULE_VERSION_INVALID"
	CodeModuleNameRequired           string = "MODULE_NAME_REQUIRED"
	CodeModuleInstallRequired        string = "MODULE_INSTALL_REQUIRED"
	CodeModuleInstallNotFound        string = "MODULE_INSTALL_NOT_FOUND"
	CodeModuleInstallStatError       string = "MODULE_INSTALL_STAT_ERROR"
	CodeModuleInstallNotDir          string = "MODULE_INSTALL_NOT_DIR"
	CodeModuleInstallIndexNotFound   string = "MODULE_INSTALL_INDEX_NOT_FOUND"
	CodeModuleInstallVersionNotFound string = "MODULE_INSTALL_VERSION_NOT_FOUND"
	CodeModuleLangInstallNotFound    string = "MODULE_LANG_INSTALL_NOT_FOUND"
	CodeModuleInstallIDMismatch      string = "MODULE_INSTALL_ID_MISMATCH"
	CodeModuleVersionDateInvalid     string = "MODULE_VERSION_DATE_INVALID"
	CodeBuildSourceDirRequired       string = "BUILD_SOURCE_DIR_REQUIRED"
	CodeBuildSourceDirNotFound       string = "BUILD_SOURCE_DIR_NOT_FOUND"
	CodeBuildSourceDirStatError      string = "BUILD_SOURCE_DIR_STAT_ERROR"
	CodeBuildSourceDirNotDir         string = "BUILD_SOURCE_DIR_NOT_DIR"
	CodeBuildOutputDirRequired       string = "BUILD_OUTPUT_DIR_REQUIRED"
	CodeOutputDirEqualsSourceDir     string = "OUTPUT_DIR_EQUALS_SOURCE_DIR"
	CodeBuildStagingDirRequired      string = "BUILD_STAGING_DIR_REQUIRED"
	CodeStagingDirEqualsOutputDir    string = "STAGING_DIR_EQUALS_OUTPUT_DIR"
	CodeStagingDirEqualsSourceDir    string = "STAGING_DIR_EQUALS_SOURCE_DIR"
	CodeBuildArchiveNameRequired     string = "BUILD_ARCHIVE_NAME_REQUIRED"
	CodeBuildArchiveNameInvalid      string = "BUILD_ARCHIVE_NAME_INVALID"
	CodeExcludePatternEmpty          string = "EXCLUDE_PATTERN_EMPTY"
	CodeForbiddenPathFound           string = "FORBIDDEN_PATH_FOUND"
	CodeForbiddenPathScanError       string = "FORBIDDEN_PATH_SCAN_ERROR"
	CodeModuleVersionSchemeInvalid   string = "MODULE_VERSION_SCHEME_INVALID"
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

	issues := ResolveAndValidateVersion(&cfg)

	validators := []Validator{
		ValidateModuleID,
		ValidateModuleVersionScheme,
		ValidateModuleName,
		ValidateModuleInstall,
		ValidateInstallVersionFile,
		ValidateInstallIndexFile,
		ValidateInstallLangFile,
		ValidateInstallIndexModuleID,
		ValidateInstallVersionDate,
		ValidateBuildSourceDir,
		ValidateBuildOutputDir,
		ValidateBuildStagingDir,
		ValidateBuildArchiveName,
		ValidateExcludePatterns,
		ValidateForbiddenPaths,
	}

	allIssues := make([]Issue, 0, len(issues)+len(validators))
	allIssues = append(allIssues, issues...)
	for _, v := range validators {
		allIssues = append(allIssues, v(cfg)...)
	}
	return allIssues
}

func RunWithResolvedVersion(cfg *config.Config) []Issue {
	_ = cfg.NormalizePaths()

	issues := ResolveAndValidateVersion(cfg)

	validators := []Validator{
		ValidateModuleID,
		ValidateModuleVersionScheme,
		ValidateModuleName,
		ValidateModuleInstall,
		ValidateInstallVersionFile,
		ValidateInstallIndexFile,
		ValidateInstallLangFile,
		ValidateInstallIndexModuleID,
		ValidateInstallVersionDate,
		ValidateBuildSourceDir,
		ValidateBuildOutputDir,
		ValidateBuildStagingDir,
		ValidateBuildArchiveName,
		ValidateExcludePatterns,
		ValidateForbiddenPaths,
	}

	allIssues := make([]Issue, 0, len(issues)+len(validators))
	allIssues = append(allIssues, issues...)
	for _, v := range validators {
		allIssues = append(allIssues, v(*cfg)...)
	}
	return allIssues
}

func installDirPath(cfg config.Config) string {
	return filepath.Join(cfg.Build.SourceDir, cfg.Module.Install)
}

func installDirReady(cfg config.Config) bool {
	info, err := os.Stat(installDirPath(cfg))
	return err == nil && info.IsDir()
}

func moduleFilePath(cfg config.Config, relPath string) string {
	return filepath.Join(cfg.Build.SourceDir, relPath)
}

func ValidateInstallVersionFile(cfg config.Config) []Issue {
	if !installDirReady(cfg) {
		return nil
	}

	versionPath := filepath.Join(installDirPath(cfg), "version.php")
	info, err := os.Stat(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Issue{{
				Code:     CodeModuleInstallVersionNotFound,
				Message:  "в директории install отсутствует обязательный файл version.php",
				Severity: Error,
			}}
		}
		return nil
	}
	if info.IsDir() {
		return []Issue{{
			Code:     CodeModuleInstallVersionNotFound,
			Message:  "путь install/version.php должен быть файлом",
			Severity: Error,
		}}
	}
	return nil
}

func ValidateInstallIndexFile(cfg config.Config) []Issue {
	if !installDirReady(cfg) {
		return nil
	}

	indexPath := filepath.Join(installDirPath(cfg), "index.php")
	info, err := os.Stat(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Issue{{
				Code:     CodeModuleInstallIndexNotFound,
				Message:  "в директории install отсутствует обязательный файл index.php",
				Severity: Error,
			}}
		}
		return nil
	}
	if info.IsDir() {
		return []Issue{{
			Code:     CodeModuleInstallIndexNotFound,
			Message:  "путь install/index.php должен быть файлом",
			Severity: Error,
		}}
	}
	return nil
}

func ValidateInstallLangFile(cfg config.Config) []Issue {
	if !installDirReady(cfg) {
		return nil
	}

	langPath := moduleFilePath(cfg, filepath.Join("lang", "ru", cfg.Module.Install, "index.php"))
	info, err := os.Stat(langPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Issue{{
				Code:     CodeModuleLangInstallNotFound,
				Message:  fmt.Sprintf("не найден файл локализации %q", filepath.Join("lang", "ru", cfg.Module.Install, "index.php")),
				Severity: Warning,
			}}
		}
		return nil
	}
	if info.IsDir() {
		return []Issue{{
			Code:     CodeModuleLangInstallNotFound,
			Message:  fmt.Sprintf("путь %q должен быть файлом локализации", filepath.Join("lang", "ru", cfg.Module.Install, "index.php")),
			Severity: Warning,
		}}
	}
	return nil
}

func ValidateInstallIndexModuleID(cfg config.Config) []Issue {
	if !installDirReady(cfg) {
		return nil
	}

	indexPath := filepath.Join(installDirPath(cfg), "index.php")
	//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}

	match := reInstallModuleID.FindStringSubmatch(string(data))
	if len(match) == 0 {
		return []Issue{{
			Code:     CodeModuleInstallIDMismatch,
			Message:  "в install/index.php не найдено присваивание $MODULE_ID",
			Severity: Warning,
		}}
	}

	actualID := strings.TrimSpace(match[1])
	if actualID != cfg.Module.ID {
		return []Issue{{
			Code:     CodeModuleInstallIDMismatch,
			Message:  fmt.Sprintf("MODULE_ID в install/index.php (%q) не совпадает с module.id (%q)", actualID, cfg.Module.ID),
			Severity: Error,
		}}
	}

	return nil
}

func ValidateInstallVersionDate(cfg config.Config) []Issue {
	if !installDirReady(cfg) {
		return nil
	}

	versionPath := filepath.Join(installDirPath(cfg), "version.php")
	//nolint:gosec // G304 - путь контролируется пользователем через конфигурационный файл утилиты
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return nil
	}

	match := reVersionDate.FindStringSubmatch(string(data))
	if len(match) == 0 {
		return []Issue{{
			Code:     CodeModuleVersionDateInvalid,
			Message:  "в install/version.php не найдено поле VERSION_DATE",
			Severity: Error,
		}}
	}

	rawDate := strings.TrimSpace(match[1])
	if _, err := time.Parse("2006-01-02 15:04:05", rawDate); err != nil {
		return []Issue{{
			Code:     CodeModuleVersionDateInvalid,
			Message:  fmt.Sprintf("VERSION_DATE %q в install/version.php должен быть в формате YYYY-MM-DD HH:MM:SS", rawDate),
			Severity: Error,
		}}
	}

	return nil
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

func ResolveAndValidateVersion(cfg *config.Config) []Issue {
	if cfg.Module.Version == "" {
		versionFile := filepath.Join(cfg.Build.SourceDir, cfg.Module.Install, "version.php")

		if _, err := os.Stat(versionFile); err != nil {
			return []Issue{{
				Code:     CodeModuleVersionRequired,
				Message:  "поле module.version в конфиге пусто, и файл install/version.php не найден",
				Severity: Error,
			}}
		}

		resolvedVersion, err := version.ParseVersion(versionFile)
		if err != nil {
			return []Issue{{
				Code:     CodeModuleVersionInvalid,
				Message:  fmt.Sprintf("ошибка парсинга версии из %s: %v", versionFile, err),
				Severity: Error,
			}}
		}

		cfg.Module.Version = resolvedVersion
	}

	return validateResolvedVersion(cfg.Module.Version, cfg.Module.VersionScheme)
}

func validateResolvedVersion(moduleVersion string, versionScheme string) []Issue {
	if moduleVersion == "" {
		return []Issue{{
			Code:     CodeModuleVersionRequired,
			Message:  "поле module.version обязательно для заполнения",
			Severity: Error,
		}}
	}

	if versionScheme == "semver" || versionScheme == "" {
		if !reModuleVersion.MatchString(moduleVersion) {
			return []Issue{{
				Code:     CodeModuleVersionInvalid,
				Message:  fmt.Sprintf("module.version %q не соответствует формату семантического версионирования", moduleVersion),
				Severity: Error,
			}}
		}
		return nil
	}

	parts := strings.Split(moduleVersion, ".")
	if len(parts) != 3 {
		return []Issue{{
			Code:     CodeModuleVersionInvalid,
			Message:  fmt.Sprintf("module.version %q должен состоять из 3-х сегментов (напр. 1.0.0)", moduleVersion),
			Severity: Error,
		}}
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

	if cfg.Build.OutputDir == cfg.Build.SourceDir {
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

	if cfg.Build.StagingDir == cfg.Build.OutputDir {
		return []Issue{{
			Code:     CodeStagingDirEqualsOutputDir,
			Message:  "stagingDir не должен совпадать с outputDir",
			Severity: Error,
		}}
	}

	if cfg.Build.StagingDir == cfg.Build.SourceDir {
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

	if !strings.HasSuffix(cfg.Build.ArchiveName, ".zip") && !strings.HasSuffix(cfg.Build.ArchiveName, ".tar.gz") {
		return []Issue{{
			Code:     CodeBuildArchiveNameInvalid,
			Message:  "build.archiveName должен оканчиваться на .zip или .tar.gz",
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
		excluded, err := pack.IsExcluded(relPath, cfg.Exclude)
		if err != nil {
			return nil // Игнорируем ошибки при обходе
		}

		if excluded {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		ext := filepath.Ext(name)

		isForbidden := slices.Contains(forbiddenNames, name)

		if !isForbidden {
			if slices.Contains(forbiddenExts, ext) {
				isForbidden = true
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
