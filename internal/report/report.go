package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"bx-pack/internal/config"
	"bx-pack/internal/validate"
)

type Format string

const (
	TextFormat Format = "text"
	JSONFormat Format = "json"
)

type Reporter interface {
	SetCommand(command string)
	PrintIssues(issues []validate.Issue) error
	PrintSummary(archivePath string) error
	PrintConfigError(err error) error
	PrintSuccess(msg string) error
	PrintInfo(msg string) error
	PrintDryRunPlan(cfg config.Config, archivePath string) error
	PrintVersion(version string) error
	PrintVersionBump(oldVersion, newVersion string) error
	Finalize() error
}

type textReporter struct {
	w       io.Writer
	err     io.Writer
	command string
}

func NewTextReporter(w, err io.Writer) Reporter {
	return &textReporter{w: w, err: err}
}

func (r *textReporter) SetCommand(command string) {
	r.command = command
}

func (r *textReporter) PrintIssues(issues []validate.Issue) error {
	errorsCount := 0
	warningsCount := 0

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			fmt.Fprintln(r.err, issue.String())
			errorsCount++
		} else if issue.Severity == validate.Warning {
			fmt.Fprintln(r.w, issue.String())
			warningsCount++
		}
	}

	if len(issues) > 0 {
		fmt.Fprintf(r.w, "\nИтог: Валидация завершена. Ошибок: %d, предупреждений: %d.\n", errorsCount, warningsCount)
	} else {
		fmt.Fprintln(r.w, "Готово: Валидация прошла успешно. Ошибок не обнаружено.")
	}
	return nil
}

func (r *textReporter) PrintSummary(archivePath string) error {
	fmt.Fprintf(r.w, "Готово: Сборка успешно завершена!\nИтог: Архив создан: %s\n", archivePath)
	return nil
}

func (r *textReporter) PrintConfigError(err error) error {
	fmt.Fprintf(r.err, "Ошибка конфигурации: %v\n", err)
	return nil
}

func (r *textReporter) PrintSuccess(msg string) error {
	fmt.Fprintf(r.w, "Готово: %s\n", msg)
	return nil
}

func (r *textReporter) PrintInfo(msg string) error {
	fmt.Fprintf(r.w, "%s\n", msg)
	return nil
}

func (r *textReporter) PrintDryRunPlan(cfg config.Config, archivePath string) error {
	fmt.Fprintln(r.w, "\n--- ПЛАН СБОРКИ (DRY RUN) ---")
	fmt.Fprintf(r.w, "Модуль:      %s (версия %s)\n", cfg.Module.ID, cfg.Module.Version)
	fmt.Fprintf(r.w, "Исходники:   %s\n", cfg.Build.SourceDir)
	fmt.Fprintf(r.w, "Staging:     %s\n", cfg.Build.StagingDir)
	fmt.Fprintf(r.w, "Output:      %s\n", cfg.Build.OutputDir)
	fmt.Fprintf(r.w, "Имя архива:  %s\n", archivePath)

	if len(cfg.Exclude) > 0 {
		fmt.Fprintln(r.w, "\nИсключения:")
		for _, exc := range cfg.Exclude {
			fmt.Fprintf(r.w, "  - %s\n", exc)
		}
	}
	fmt.Fprintln(r.w, "----------------------------")
	fmt.Fprintln(r.w, "\nDry run завершен. Файлы не были изменены.")
	return nil
}

func (r *textReporter) PrintVersion(version string) error {
	fmt.Fprintf(r.w, "Версия модуля: %s\n", version)
	return nil
}

func (r *textReporter) PrintVersionBump(oldVersion, newVersion string) error {
	fmt.Fprintf(r.w, "Версия обновлена: %s -> %s\n", oldVersion, newVersion)
	return nil
}

func (r *textReporter) Finalize() error {
	return nil
}

type JSONReport struct {
	Command         string           `json:"command"`
	Success         bool             `json:"success"`
	Errors          []string         `json:"errors,omitzero"`
	Warnings        []string         `json:"warnings,omitzero"`
	Findings        []validate.Issue `json:"findings,omitzero"`
	Summary         string           `json:"summary,omitzero"`
	Version         string           `json:"version,omitzero"`
	PreviousVersion string           `json:"previousVersion,omitzero"`
	NewVersion      string           `json:"newVersion,omitzero"`
	ArchivePath     string           `json:"archivePath,omitzero"`
	DryRun          bool             `json:"dryRun"`
}

type jsonReporter struct {
	w      io.Writer
	report JSONReport
}

func NewJSONReporter(w io.Writer) Reporter {
	return &jsonReporter{
		w: w,
		report: JSONReport{
			Success: true,
		},
	}
}

func (r *jsonReporter) SetCommand(command string) {
	r.report.Command = command
}

func (r *jsonReporter) PrintIssues(issues []validate.Issue) error {
	if issues == nil {
		issues = []validate.Issue{}
	}
	r.report.Findings = append(r.report.Findings, issues...)

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			r.report.Success = false
			r.report.Errors = append(r.report.Errors, issue.Message)
		} else if issue.Severity == validate.Warning {
			r.report.Warnings = append(r.report.Warnings, issue.Message)
		}
	}
	return nil
}

func (r *jsonReporter) PrintSummary(archivePath string) error {
	r.report.ArchivePath = archivePath
	r.report.Summary = fmt.Sprintf("Архив создан: %s", archivePath)
	return nil
}

func (r *jsonReporter) PrintConfigError(err error) error {
	r.report.Success = false
	r.report.Errors = append(r.report.Errors, err.Error())
	return nil
}

func (r *jsonReporter) PrintSuccess(msg string) error {
	r.report.Summary = msg
	return nil
}

func (r *jsonReporter) PrintInfo(msg string) error {
	return nil
}

func (r *jsonReporter) PrintDryRunPlan(cfg config.Config, archivePath string) error {
	r.report.DryRun = true
	r.report.ArchivePath = archivePath
	r.report.Summary = "Dry run completed successfully"
	return nil
}

func (r *jsonReporter) PrintVersion(version string) error {
	r.report.Version = version
	r.report.Summary = fmt.Sprintf("Версия модуля: %s", version)
	return nil
}

func (r *jsonReporter) PrintVersionBump(oldVersion, newVersion string) error {
	r.report.PreviousVersion = oldVersion
	r.report.NewVersion = newVersion
	r.report.Summary = fmt.Sprintf("Версия обновлена: %s -> %s", oldVersion, newVersion)
	return nil
}

func (r *jsonReporter) Finalize() error {
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r.report); err != nil {
		return fmt.Errorf("encode report to JSON: %w", err)
	}
	return nil
}

func NewReporter(format Format) Reporter {
	return NewReporterWithWriter(format, os.Stdout, os.Stderr)
}

func NewReporterWithWriter(format Format, w, err io.Writer) Reporter {
	switch format {
	case JSONFormat:
		return NewJSONReporter(w)
	default:
		return NewTextReporter(w, err)
	}
}

func IsJSON(r Reporter) bool {
	_, ok := r.(*jsonReporter)
	return ok
}

func (r *jsonReporter) String() string { return "JSONReporter" }
func (r *textReporter) String() string { return "TextReporter" }
