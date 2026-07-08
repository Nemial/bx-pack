package report_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Nemial/bx-pack/internal/config"
	"github.com/Nemial/bx-pack/internal/report"
	"github.com/Nemial/bx-pack/internal/validate"
)

// readGoldenFile reads a golden file from testdata directory
func readGoldenFile(t *testing.T, filename string) string {
	t.Helper()
	//nolint:gosec // G304 - test file path is controlled by test code
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", filename, err)
	}
	return string(data)
}

func TestTextReporter_PrintIssues(t *testing.T) {
	tests := []struct {
		name             string
		issues           []validate.Issue
		wantStdoutGolden string
		wantStderrGolden string
	}{
		{
			name:             "no issues",
			issues:           []validate.Issue{},
			wantStdoutGolden: "validation_no_issues.stdout.golden",
			wantStderrGolden: "validation_no_issues.stderr.golden",
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
			wantStdoutGolden: "validation_warnings.stdout.golden",
			wantStderrGolden: "validation_warnings.stderr.golden",
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
			wantStdoutGolden: "validation_errors.stdout.golden",
			wantStderrGolden: "validation_errors.stderr.golden",
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
			wantStdoutGolden: "validation_mixed.stdout.golden",
			wantStderrGolden: "validation_mixed.stderr.golden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			r := report.NewTextReporter(&stdout, &stderr)
			err := r.PrintValidationResult(tt.issues)
			if err != nil {
				t.Fatalf("PrintValidationResult failed: %v", err)
			}

			wantStdout := readGoldenFile(t, tt.wantStdoutGolden)
			wantStderr := readGoldenFile(t, tt.wantStderrGolden)

			if got := stdout.String(); got != wantStdout {
				t.Errorf("stdout:\ngot:  %q\nwant: %q", got, wantStdout)
			}
			if got := stderr.String(); got != wantStderr {
				t.Errorf("stderr:\ngot:  %q\nwant: %q", got, wantStderr)
			}
		})
	}
}

func TestTextReporter_PrintIssues_Simple(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := report.NewTextReporter(&stdout, &stderr)
	issues := []validate.Issue{
		{Code: "ERR_001", Message: "Error", Severity: validate.Error},
		{Code: "WARN_001", Message: "Warning", Severity: validate.Warning},
	}
	err := r.PrintIssues(issues)
	if err != nil {
		return
	}

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
	r := report.NewTextReporter(&stdout, &stderr)
	archivePath := "/path/to/archive.zip"
	err := r.PrintSummary(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	want := readGoldenFile(t, "summary.stdout.golden")
	if got := stdout.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTextReporter_PrintConfigError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := report.NewTextReporter(&stdout, &stderr)
	cfgErr := errors.New("missing field")
	err := r.PrintConfigError(cfgErr)
	if err != nil {
		t.Fatal(err)
	}

	want := readGoldenFile(t, "config_error.stderr.golden")
	if got := stderr.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTextReporter_PrintDryRunPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := report.NewTextReporter(&stdout, &stderr)
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

	want := readGoldenFile(t, "dryrun_plan.stdout.golden")
	if got := stdout.String(); got != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestJSONReporter(t *testing.T) {
	var buf bytes.Buffer
	r := report.NewJSONReporter(&buf)
	r.SetCommand("test-cmd")

	issues := []validate.Issue{
		{Code: "ERR1", Message: "Error 1", Severity: validate.Error},
		{Code: "WARN1", Message: "Warning 1", Severity: validate.Warning},
	}

	err := r.PrintIssues(issues)
	if err != nil {
		return
	}
	err = r.PrintSummary("/path/to/zip")
	if err != nil {
		return
	}
	err = r.Finalize()
	if err != nil {
		return
	}

	// Parse both actual and expected JSON to compare structurally
	got := buf.String()
	var gotJSON, wantJSON any
	if err := json.Unmarshal([]byte(got), &gotJSON); err != nil {
		t.Fatalf("failed to parse actual JSON: %v\nOutput:\n%s", err, got)
	}

	wantGolden := readGoldenFile(t, "json_reporter.golden")
	if err := json.Unmarshal([]byte(wantGolden), &wantJSON); err != nil {
		t.Fatalf("failed to parse golden JSON: %v", err)
	}

	// Compare as maps to ignore formatting differences
	gotMap, ok := gotJSON.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object, got %T", gotJSON)
	}
	wantMap, ok := wantJSON.(map[string]any)
	if !ok {
		t.Fatalf("expected golden JSON object, got %T", wantJSON)
	}

	// Check key fields
	for key, wantVal := range wantMap {
		gotVal, exists := gotMap[key]
		if !exists {
			t.Errorf("missing key %q in JSON output", key)
			continue
		}
		if !reflect.DeepEqual(gotVal, wantVal) {
			t.Errorf("key %q:\ngot:  %v\nwant: %v", key, gotVal, wantVal)
		}
	}
}
