package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/Nemial/bx-pack/internal/config"
	"github.com/Nemial/bx-pack/internal/report"
)

func setupTestProject(t *testing.T) string {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	t.Chdir(tmpDir)
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	cfg := config.Default()
	cfg.Module.ID = "test.json"
	cfg.Module.Name = "Test JSON Output"
	cfg.Module.Version = "1.0.0"
	cfg.Module.Install = "install"
	writeValidModuleFixture(t, cfg.Module.ID, cfg.Module.Version)
	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestJSONOutput_Validate(t *testing.T) {
	setupTestProject(t)

	var buf bytes.Buffer
	reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

	err := Validate(reporter)
	if err != nil {
		t.Logf("Validate returned error (expected if issues found): %v", err)
	}
	_ = reporter.Finalize()

	var res report.JSONReport
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v\nOutput: %s", err, buf.String())
	}

	if res.Command != "validate" {
		t.Errorf("Expected command 'validate', got %q", res.Command)
	}
}

func TestJSONOutput_Build(t *testing.T) {
	setupTestProject(t)

	var buf bytes.Buffer
	reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

	err := Build(reporter, false)
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, buf.String())
	}
	_ = reporter.Finalize()

	var res report.JSONReport
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v\nOutput: %s", err, buf.String())
	}

	if res.Command != "build" {
		t.Errorf("Expected command 'build', got %q", res.Command)
	}
	if !res.Success {
		t.Errorf("Expected success true, got false. Errors: %v", res.Errors)
	}
	if res.ArchivePath == "" {
		t.Error("Expected archivePath to be set")
	}
}

func TestJSONOutput_VersionShow(t *testing.T) {
	setupTestProject(t)

	versionContent := `<?php
$VERSION = "2.4.6";
$VERSION_DATE = "2023-01-01 00:00:00";
?>`
	if err := os.WriteFile("install/version.php", []byte(versionContent), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

	err := VersionShow(reporter)
	if err != nil {
		t.Fatalf("VersionShow failed: %v\nOutput: %s", err, buf.String())
	}
	_ = reporter.Finalize()

	var res report.JSONReport
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v\nOutput: %s", err, buf.String())
	}

	if res.Command != "version show" {
		t.Errorf("Expected command 'version show', got %q", res.Command)
	}
	if res.Version != "2.4.6" {
		t.Errorf("Expected version '2.4.6', got %q", res.Version)
	}
	if !res.Success {
		t.Errorf("Expected success true, got false. Errors: %v", res.Errors)
	}
}

func TestJSONOutput_VersionBump(t *testing.T) {
	setupTestProject(t)

	versionContent := `<?php
$VERSION = "1.0.0";
$VERSION_DATE = "2023-01-01 00:00:00";
?>`
	if err := os.WriteFile("install/version.php", []byte(versionContent), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

	err := VersionBump(reporter, "patch")
	if err != nil {
		t.Fatalf("VersionBump failed: %v\nOutput: %s", err, buf.String())
	}
	_ = reporter.Finalize()

	var res report.JSONReport
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v\nOutput: %s", err, buf.String())
	}

	if res.Command != "version bump patch" {
		t.Errorf("Expected command 'version bump patch', got %q", res.Command)
	}
	if res.PreviousVersion != "1.0.0" {
		t.Errorf("Expected previousVersion '1.0.0', got %q", res.PreviousVersion)
	}
	if res.NewVersion != "1.0.1" {
		t.Errorf("Expected newVersion '1.0.1', got %q", res.NewVersion)
	}
	if !res.Success {
		t.Errorf("Expected success true, got false. Errors: %v", res.Errors)
	}
}
