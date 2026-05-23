package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bx-pack/internal/config"
	"bx-pack/internal/report"
)

func TestInit_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// 1. First init
	reporter := report.NewReporter(report.TextFormat)
	err := Init(reporter)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if _, err := os.Stat(config.DefaultConfigPath); os.IsNotExist(err) {
		t.Error("config file not created")
	}

	// 2. Second init should fail
	err = Init(reporter)
	if err == nil {
		t.Error("Init should fail if config already exists")
	}
}

func TestBuild_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Setup minimal valid project
	cfg := config.Default()
	cfg.Module.ID = "test.integration"
	cfg.Module.Name = "Test Integration"
	cfg.Module.Version = "1.0.0"
	cfg.Module.Install = "install"
	cfg.Build.SourceDir = "."
	cfg.Build.OutputDir = "./dist"
	cfg.Build.StagingDir = "./.bxpack/staging"

	if err := os.Mkdir("install", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("readme.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}

	// Run Build
	reporter := report.NewReporter(report.TextFormat)
	err := Build(reporter, false)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify output
	archiveName := "test.integration-1.0.0.zip"
	archivePath := filepath.Join("dist", archiveName)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("archive %s not created", archivePath)
	}
}

func TestBuild_DryRun_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Setup minimal valid project
	cfg := config.Default()
	cfg.Module.ID = "test.dryrun"
	cfg.Module.Version = "1.0.0"
	cfg.Build.OutputDir = "./dist"

	if err := os.Mkdir("install", 0755); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}

	// Run Build with dry-run
	reporter := report.NewReporter(report.TextFormat)
	err := Build(reporter, true)
	if err != nil {
		t.Fatalf("Build dry-run failed: %v", err)
	}

	// Verify no output
	archiveName := "test.dryrun-1.0.0.zip"
	archivePath := filepath.Join("dist", archiveName)
	if _, err := os.Stat(archivePath); err == nil {
		t.Errorf("archive %s should NOT be created in dry-run", archivePath)
	}

	if _, err := os.Stat("./.bxpack/staging"); err == nil {
		t.Error("staging directory should NOT be created in dry-run")
	}
}

func TestValidate_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Setup invalid project
	cfg := config.Default()
	cfg.Module.ID = "example.module" // Default ID should trigger error
	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}

	// Run Validate
	reporter := report.NewReporter(report.TextFormat)
	err := Validate(reporter)
	if err == nil {
		t.Error("Validate should fail for default config")
	}
}

func TestValidate_InvalidVersionFile_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	cfg := config.Default()
	cfg.Module.ID = "test.invalid.version"
	cfg.Module.Name = "Test Invalid Version"
	cfg.Module.Version = ""
	cfg.Module.Install = "install"

	if err := os.Mkdir("install", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("install/version.php", []byte("invalid content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}

	reporter := report.NewReporter(report.TextFormat)
	err := Validate(reporter)
	if err == nil {
		t.Fatal("Validate should fail for invalid version.php")
	}
}

func TestVersionShow_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// 1. Success case
	cfg := config.Default()
	cfg.Module.Install = "install"
	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir("install", 0755); err != nil {
		t.Fatal(err)
	}
	versionContent := `<?php
$VERSION = "1.2.3";
$VERSION_DATE = "2023-01-01 00:00:00";
?>`
	if err := os.WriteFile("install/version.php", []byte(versionContent), 0644); err != nil {
		t.Fatal(err)
	}

	reporter := report.NewReporter(report.TextFormat)
	err := VersionShow(reporter)
	if err != nil {
		t.Fatalf("VersionShow failed: %v", err)
	}

	// 2. Missing file case
	os.Remove("install/version.php")
	err = VersionShow(reporter)
	if err == nil {
		t.Error("VersionShow should fail if version file is missing")
	}

	// 3. Invalid file case
	if err := os.WriteFile("install/version.php", []byte("invalid content"), 0644); err != nil {
		t.Fatal(err)
	}
	err = VersionShow(reporter)
	if err == nil {
		t.Error("VersionShow should fail if version file is invalid")
	}
}

func TestVersionBump_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	cfg := config.Default()
	cfg.Module.Install = "install"
	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir("install", 0755); err != nil {
		t.Fatal(err)
	}
	versionContent := `<?php
$VERSION = "1.0.0";
$VERSION_DATE = "2023-01-01 00:00:00";
?>`
	if err := os.WriteFile("install/version.php", []byte(versionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Patch bump
	reporter := report.NewReporter(report.TextFormat)
	err := VersionBump(reporter, "patch")
	if err != nil {
		t.Fatalf("VersionBump failed: %v", err)
	}

	// Verify file updated
	data, err := os.ReadFile("install/version.php")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `$VERSION = "1.0.1";`) {
		t.Errorf("version not updated to 1.0.1: %s", string(data))
	}

	// 2. Invalid bump level
	err = VersionBump(reporter, "invalid")
	if err == nil {
		t.Error("VersionBump should fail for invalid level")
	}

	// 3. Invalid version format
	if err := os.WriteFile("install/version.php", []byte(`<?php $VERSION = "1.0"; ?>`), 0644); err != nil {
		t.Fatal(err)
	}
	err = VersionBump(reporter, "patch")
	if err == nil {
		t.Error("VersionBump should fail for invalid version format")
	}
}

func TestVersionBump_PreservesConfigComments(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	configContent := `module:
  id: "example.module"
  version: "1.0.0" # important comment
  versionScheme: "semver"
  name: "Test Module"
  install: "install"

build:
  sourceDir: "."
  outputDir: "./dist"
  stagingDir: "./.bxpack/staging"
  archiveName: "{module.id}-{module.version}.zip"
`

	if err := os.WriteFile(config.DefaultConfigPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir("install", 0755); err != nil {
		t.Fatal(err)
	}
	versionContent := `<?php
$VERSION = "1.0.0";
$VERSION_DATE = "2023-01-01 00:00:00";
?>`
	if err := os.WriteFile("install/version.php", []byte(versionContent), 0644); err != nil {
		t.Fatal(err)
	}

	reporter := report.NewReporter(report.TextFormat)
	if err := VersionBump(reporter, "patch"); err != nil {
		t.Fatalf("VersionBump failed: %v", err)
	}

	data, err := os.ReadFile(config.DefaultConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if !strings.Contains(got, `version: "1.0.1" # important comment`) {
		t.Fatalf("expected updated version with preserved comment, got:\n%s", got)
	}
	if !strings.Contains(got, "build:\n  sourceDir: \".\"") {
		t.Fatalf("expected build section to remain intact, got:\n%s", got)
	}
}
