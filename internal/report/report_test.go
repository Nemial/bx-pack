package report

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"bx-pack/internal/config"
	"bx-pack/internal/validate"
)

func TestTextReporter_PrintIssues(t *testing.T) {
	tests := []struct {
		name       string
		issues     []validate.Issue
		wantStdout string
		wantStderr string
	}{
		{
			name:       "no issues",
			issues:     []validate.Issue{},
			wantStdout: "Готово: Валидация прошла успешно. Ошибок не обнаружено.\n",
			wantStderr: "",
		},
		{
			name: "only warnings",
			issues: []validate.Issue{
				{
					Code:     "WARN_001",
					Message:  "Some warning",
					Severity: validate.Warning,
				},
			},
			wantStdout: "Предупреждение: Some warning (WARN_001)\n\nИтог: Валидация завершена. Ошибок: 0, предупреждений: 1.\n",
			wantStderr: "",
		},
		{
			name: "only errors",
			issues: []validate.Issue{
				{
					Code:     "ERR_001",
					Message:  "Some error",
					Severity: validate.Error,
				},
			},
			wantStdout: "\nИтог: Валидация завершена. Ошибок: 1, предупреждений: 0.\n",
			wantStderr: "Ошибка проверки: Some error (ERR_001)\n",
		},
		{
			name: "mixed issues",
			issues: []validate.Issue{
				{
					Code:     "ERR_001",
					Message:  "Critical error",
					Severity: validate.Error,
				},
				{
					Code:     "WARN_002",
					Message:  "Minor warning",
					Severity: validate.Warning,
				},
			},
			wantStdout: "Предупреждение: Minor warning (WARN_002)\n\nИтог: Валидация завершена. Ошибок: 1, предупреждений: 1.\n",
			wantStderr: "Ошибка проверки: Critical error (ERR_001)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			r := NewTextReporter(&stdout, &stderr)
			err := r.PrintValidationResult(tt.issues)
			if err != nil {
				t.Fatalf("PrintValidationResult failed: %v", err)
			}

			if got := stdout.String(); got != tt.wantStdout {
				t.Errorf("stdout:\ngot:  %q\nwant: %q", got, tt.wantStdout)
			}
			if got := stderr.String(); got != tt.wantStderr {
				t.Errorf("stderr:\ngot:  %q\nwant: %q", got, tt.wantStderr)
			}
		})
	}
}

func TestTextReporter_PrintIssues_Simple(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := NewTextReporter(&stdout, &stderr)
	issues := []validate.Issue{
		{Code: "ERR_001", Message: "Error", Severity: validate.Error},
		{Code: "WARN_001", Message: "Warning", Severity: validate.Warning},
	}
	r.PrintIssues(issues)

	if !strings.Contains(stderr.String(), "Ошибка проверки: Error (ERR_001)") {
		t.Errorf("stderr should contain error message, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Предупреждение: Warning (WARN_001)") {
		t.Errorf("stdout should contain warning message, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Итог:") {
		t.Errorf("stdout should NOT contain summary in PrintIssues, got %q", stdout.String())
	}
}

func TestTextReporter_PrintSummary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := NewTextReporter(&stdout, &stderr)
	archivePath := "/path/to/archive.zip"
	err := r.PrintSummary(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf("Готово: Сборка успешно завершена!\nИтог: Архив создан: %s\n", archivePath)
	if got := stdout.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTextReporter_PrintConfigError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := NewTextReporter(&stdout, &stderr)
	cfgErr := errors.New("missing field")
	err := r.PrintConfigError(cfgErr)
	if err != nil {
		t.Fatal(err)
	}

	want := "Ошибка конфигурации: missing field\n"
	if got := stderr.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTextReporter_PrintDryRunPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := NewTextReporter(&stdout, &stderr)
	cfg := config.Config{
		Module: config.Module{
			ID:      "test.mod",
			Version: "1.0.0",
		},
		Build: config.Build{
			SourceDir:  "/src",
			StagingDir: "/staging",
			OutputDir:  "/dist",
		},
		Exclude: []string{"node_modules", ".git"},
	}
	archivePath := "/dist/test.mod-1.0.0.zip"

	err := r.PrintDryRunPlan(cfg, archivePath)
	if err != nil {
		t.Fatal(err)
	}

	got := stdout.String()
	keywords := []string{
		"--- ПЛАН СБОРКИ (DRY RUN) ---",
		"Модуль:      test.mod (версия 1.0.0)",
		"Исходники:   /src",
		"Исключения:",
		"  - node_modules",
		"  - .git",
		"Dry run завершен. Файлы не были изменены.",
	}

	for _, kw := range keywords {
		if !strings.Contains(got, kw) {
			t.Errorf("expected output to contain %q, but it didn't.\nOutput:\n%s", kw, got)
		}
	}
}

func TestJSONReporter(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONReporter(&buf)
	r.SetCommand("test-cmd")

	issues := []validate.Issue{
		{Code: "ERR1", Message: "Error 1", Severity: validate.Error},
		{Code: "WARN1", Message: "Warning 1", Severity: validate.Warning},
	}

	r.PrintIssues(issues)
	r.PrintSummary("/path/to/zip")
	r.Finalize()

	got := buf.String()
	// Проверяем наличие ключевых полей и значений, не завязываясь на форматирование (хотя Finalize его задает)
	keywords := []string{
		`"command": "test-cmd"`,
		`"success": false`,
		`"errors": [`,
		`"Error 1"`,
		`"warnings": [`,
		`"Warning 1"`,
		`"archivePath": "/path/to/zip"`,
		`"summary": "Архив создан: /path/to/zip"`,
	}

	for _, kw := range keywords {
		if !strings.Contains(got, kw) {
			t.Errorf("expected JSON to contain %q, but it didn't.\nOutput:\n%s", kw, got)
		}
	}
}
