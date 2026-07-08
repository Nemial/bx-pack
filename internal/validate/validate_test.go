package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nemial/bx-pack/internal/config"
)

func writeValidInstallFixture(t *testing.T, rootDir, moduleID, version string) {
	t.Helper()

	installDir := filepath.Join(rootDir, "install")
	if err := os.MkdirAll(installDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "lang", "ru", "install"), 0o750); err != nil {
		t.Fatal(err)
	}

	versionContent := "<?php\n" +
		"$VERSION = \"" + version + "\";\n" +
		"$VERSION_DATE = \"2026-01-01 00:00:00\";\n" +
		"?>\n"
	if err := os.WriteFile(filepath.Join(installDir, "version.php"), []byte(versionContent), 0o600); err != nil {
		t.Fatal(err)
	}

	indexContent := "<?php\n" +
		"$MODULE_ID = \"" + moduleID + "\";\n" +
		"?>\n"
	if err := os.WriteFile(filepath.Join(installDir, "index.php"), []byte(indexContent), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(rootDir, "lang", "ru", "install", "index.php"), []byte("<?php\n$MESS = [];\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRun_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "install")
	if err := os.Mkdir(installDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Создаем version.php для прохождения валидации, если версия в конфиге пуста
	writeValidInstallFixture(t, tmpDir, "my.custom.id", "1.0.0")

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)
	for _, issue := range issues {
		if issue.Severity == Error {
			t.Errorf("unexpected error: %v", issue)
		}
	}
}

func TestRun_SourceDirNotFound(t *testing.T) {
	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Build.SourceDir = "./non-existent-dir"

	issues := Run(cfg)
	found := false
	for _, issue := range issues {
		if issue.Code == "BUILD_SOURCE_DIR_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected BUILD_SOURCE_DIR_NOT_FOUND error")
	}
}

func TestRun_InstallDirNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "missing-install"

	issues := Run(cfg)
	found := false
	for _, issue := range issues {
		if issue.Code == "MODULE_INSTALL_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MODULE_INSTALL_NOT_FOUND error")
	}
}

func TestRun_EmptyConfig(t *testing.T) {
	cfg := config.Config{}
	issues := Run(cfg)
	hasError := false
	for _, issue := range issues {
		if issue.Severity == Error {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("expected errors for empty config, got none")
	}
}

func TestRun_MissingOptionalName(t *testing.T) {
	cfg := config.Default()
	cfg.Module.Name = ""
	issues := Run(cfg)
	hasWarning := false
	for _, issue := range issues {
		if issue.Severity == Warning && issue.Code == "MODULE_NAME_REQUIRED" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning for empty module name, got none")
	}
}

func TestRun_StagingEqualsOutput(t *testing.T) {
	cfg := config.Default()
	cfg.Build.OutputDir = "./dist"
	cfg.Build.StagingDir = "./dist"
	issues := Run(cfg)
	hasError := false
	for _, issue := range issues {
		if issue.Severity == Error && issue.Code == "STAGING_DIR_EQUALS_OUTPUT_DIR" {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("expected error when stagingDir equals outputDir")
	}
}

func TestRun_EmptyExcludePattern(t *testing.T) {
	cfg := config.Default()
	cfg.Exclude = []string{".git", ""}
	issues := Run(cfg)
	hasWarning := false
	for _, issue := range issues {
		if issue.Severity == Warning && issue.Code == "EXCLUDE_PATTERN_EMPTY" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning for empty exclude pattern")
	}
}

func TestRun_ForbiddenPathsFound(t *testing.T) {
	tmpDir := t.TempDir()
	// Создаем .git директорию
	if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0750); err != nil {
		t.Fatal(err)
	}
	// Создаем .idea директорию
	if err := os.Mkdir(filepath.Join(tmpDir, ".idea"), 0750); err != nil {
		t.Fatal(err)
	}
	// Создаем .log файл
	if err := os.WriteFile(filepath.Join(tmpDir, "error.log"), []byte("some log"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Создаем разрешенный файл
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("some text"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Создаем разрешенную директорию install
	if err := os.Mkdir(filepath.Join(tmpDir, "install"), 0750); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)

	expectedCodes := map[string]bool{
		"FORBIDDEN_PATH_FOUND": false,
	}

	for _, issue := range issues {
		if _, ok := expectedCodes[issue.Code]; ok {
			expectedCodes[issue.Code] = true
		}
	}

	for code, found := range expectedCodes {
		if !found {
			t.Errorf("expected issue code %s not found", code)
		}
	}
}

func TestRun_InvalidModuleID(t *testing.T) {
	cfg := config.Default()
	cfg.Module.ID = "invalid id!"
	issues := Run(cfg)
	found := false
	for _, issue := range issues {
		if issue.Code == "MODULE_ID_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MODULE_ID_INVALID error")
	}
}

func TestRun_InvalidModuleVersion(t *testing.T) {
	cfg := config.Default()
	cfg.Module.Version = "invalid-version"
	issues := Run(cfg)
	found := false
	for _, issue := range issues {
		if issue.Code == "MODULE_VERSION_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MODULE_VERSION_INVALID error")
	}
}

func TestRun_InvalidArchiveFormat(t *testing.T) {
	cfg := config.Default()
	cfg.Build.ArchiveName = "module.rar"

	issues := Run(cfg)
	found := false
	for _, issue := range issues {
		if issue.Code == "BUILD_ARCHIVE_NAME_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected BUILD_ARCHIVE_NAME_INVALID error")
	}
}

func TestRun_VersionResolvedFromInstallVersionFile(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidInstallFixture(t, tmpDir, "my.custom.id", "2.3.4")

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Module.Version = ""
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := RunWithResolvedVersion(&cfg)
	for _, issue := range issues {
		if issue.Severity == Error {
			t.Fatalf("unexpected error: %v", issue)
		}
	}

	if cfg.Module.Version != "2.3.4" {
		t.Fatalf("expected resolved version 2.3.4, got %q", cfg.Module.Version)
	}
}

func TestRun_InvalidVersionFileReturnsValidationIssue(t *testing.T) {
	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "install")
	if err := os.MkdirAll(filepath.Join(tmpDir, "install"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "lang", "ru", "install"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "install", "version.php"), []byte("invalid content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "install", "index.php"), []byte("<?php\n$MODULE_ID = \"my.custom.id\";\n?>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "lang", "ru", "install", "index.php"), []byte("<?php\n$MESS = [];\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "version.php"), []byte("invalid content"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Module.Version = ""
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := RunWithResolvedVersion(&cfg)

	found := false
	for _, issue := range issues {
		if issue.Code == "MODULE_VERSION_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected MODULE_VERSION_INVALID error")
	}
}

func TestRun_MissingInstallIndexFile(t *testing.T) {
	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "install")
	if err := os.Mkdir(installDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "version.php"), []byte(`<?php $arModuleVersion = ["VERSION" => "1.0.0", "VERSION_DATE" => "2026-01-01 00:00:00"]; ?>`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)

	found := false
	for _, issue := range issues {
		if issue.Code == CodeModuleInstallIndexNotFound {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected MODULE_INSTALL_INDEX_NOT_FOUND error")
	}
}

func TestRun_MissingInstallVersionFile(t *testing.T) {
	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "install")
	if err := os.Mkdir(installDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "index.php"), []byte(`<?php class Test extends CModule { public $MODULE_ID = 'my.custom.id'; }`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Module.Version = "1.0.0"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)

	found := false
	for _, issue := range issues {
		if issue.Code == CodeModuleInstallVersionNotFound {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected MODULE_INSTALL_VERSION_NOT_FOUND error")
	}
}

func TestRun_MissingRuInstallLocalizationFile(t *testing.T) {
	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "install")
	if err := os.Mkdir(installDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "index.php"), []byte(`<?php class Test extends CModule { public $MODULE_ID = 'my.custom.id'; }`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "version.php"), []byte(`<?php $arModuleVersion = ["VERSION" => "1.0.0", "VERSION_DATE" => "2026-01-01 00:00:00"]; ?>`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Module.Version = "1.0.0"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)

	found := false
	for _, issue := range issues {
		if issue.Code == CodeModuleLangInstallNotFound && issue.Severity == Warning {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected MODULE_LANG_INSTALL_NOT_FOUND warning")
	}
}

func TestRun_InstallModuleIDMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "install")
	langDir := filepath.Join(tmpDir, "lang", "ru", "install")
	if err := os.MkdirAll(langDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(installDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "index.php"), []byte(`<?php class Test extends CModule { public $MODULE_ID = 'wrong.module.id'; }`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "version.php"), []byte(`<?php $arModuleVersion = ["VERSION" => "1.0.0", "VERSION_DATE" => "2026-01-01 00:00:00"]; ?>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "index.php"), []byte(`<?php`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Module.Version = "1.0.0"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)

	found := false
	for _, issue := range issues {
		if issue.Code == CodeModuleInstallIDMismatch && issue.Severity == Error {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected MODULE_INSTALL_ID_MISMATCH error")
	}
}

func TestRun_InvalidVersionDateFormat(t *testing.T) {
	tmpDir := t.TempDir()
	installDir := filepath.Join(tmpDir, "install")
	langDir := filepath.Join(tmpDir, "lang", "ru", "install")
	if err := os.MkdirAll(langDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(installDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "index.php"), []byte(`<?php class Test extends CModule { public $MODULE_ID = 'my.custom.id'; }`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "version.php"), []byte(`<?php $arModuleVersion = ["VERSION" => "1.0.0", "VERSION_DATE" => "01.01.2026"]; ?>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "index.php"), []byte(`<?php`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Module.Version = "1.0.0"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)

	found := false
	for _, issue := range issues {
		if issue.Code == CodeModuleVersionDateInvalid {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected MODULE_VERSION_DATE_INVALID error")
	}
}

func TestRun_ForbiddenPathsExcluded(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "install"), 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.custom.id"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"
	cfg.Exclude = []string{".git"}

	issues := Run(cfg)
	for _, issue := range issues {
		if issue.Code == "FORBIDDEN_PATH_FOUND" {
			t.Errorf("unexpected issue for excluded path: %v", issue)
		}
	}
}

func TestRun_ForbiddenNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "some/nested/dir")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "error.log"), []byte("log"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "install"), 0o750); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Module.ID = "my.id"
	cfg.Build.SourceDir = tmpDir
	cfg.Module.Install = "install"

	issues := Run(cfg)
	found := false
	for _, issue := range issues {
		if issue.Code == "FORBIDDEN_PATH_FOUND" && strings.Contains(issue.Message, "error.log") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected FORBIDDEN_PATH_FOUND for nested forbidden file")
	}
}
