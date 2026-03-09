package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"bx-pack/internal/config"
	"bx-pack/internal/report"
)

func TestJSONOutput_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Setup a project
	cfg := config.Default()
	cfg.Module.ID = "test.json"
	cfg.Module.Name = "Test JSON Output"
	cfg.Module.Version = "1.0.0"
	cfg.Module.Install = "install"
	if err := os.Mkdir("install", 0755); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfg, config.DefaultConfigPath); err != nil {
		t.Fatal(err)
	}

	t.Run("validate command", func(t *testing.T) {
		var buf bytes.Buffer
		reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

		err := Validate(reporter)
		if err != nil {
			t.Logf("Validate returned error (expected if issues found): %v", err)
		}
		reporter.Finalize()

		var res report.JSONReport
		if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
			t.Fatalf("Failed to unmarshal JSON output: %v\nOutput: %s", err, buf.String())
		}

		if res.Command != "validate" {
			t.Errorf("Expected command 'validate', got %q", res.Command)
		}
		// Based on our setup, it might have some findings (e.g. Forbidden paths if any)
		// but since it's a temp dir it should be relatively clean.
	})

	t.Run("build command", func(t *testing.T) {
		var buf bytes.Buffer
		reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

		err := Build(reporter, false)
		if err != nil {
			t.Fatalf("Build failed: %v\nOutput: %s", err, buf.String())
		}
		reporter.Finalize()

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
	})

	t.Run("version show command", func(t *testing.T) {
		versionContent := `<?php
$VERSION = "2.4.6";
$VERSION_DATE = "2023-01-01 00:00:00";
?>`
		if err := os.WriteFile("install/version.php", []byte(versionContent), 0644); err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

		err := VersionShow(reporter)
		if err != nil {
			t.Fatalf("VersionShow failed: %v\nOutput: %s", err, buf.String())
		}
		reporter.Finalize()

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
	})

	t.Run("version bump command", func(t *testing.T) {
		versionContent := `<?php
$VERSION = "1.0.0";
$VERSION_DATE = "2023-01-01 00:00:00";
?>`
		if err := os.WriteFile("install/version.php", []byte(versionContent), 0644); err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		reporter := report.NewReporterWithWriter(report.JSONFormat, &buf, &buf)

		err := VersionBump(reporter, "patch")
		if err != nil {
			t.Fatalf("VersionBump failed: %v\nOutput: %s", err, buf.String())
		}
		reporter.Finalize()

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
	})
}
