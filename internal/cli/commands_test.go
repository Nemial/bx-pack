package cli

import (
	"bx-pack/internal/config"
	"bx-pack/internal/report"
	"os"
	"path/filepath"
	"testing"
)

func TestInit_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// 1. First init
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if _, err := os.Stat(config.DefaultConfigPath); os.IsNotExist(err) {
		t.Error("config file not created")
	}

	// 2. Second init should fail
	err = Init()
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
	err := Build(report.TextFormat)
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
	err := Validate(report.TextFormat)
	if err == nil {
		t.Error("Validate should fail for default config")
	}
}
