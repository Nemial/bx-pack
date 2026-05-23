package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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
	PrintValidationResult(issues []validate.Issue) error
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
	w            io.Writer
	err          io.Writer
	command      string
	stdoutStyled bool
	stderrStyled bool
}

func NewTextReporter(w, err io.Writer) Reporter {
	return &textReporter{
		w:            w,
		err:          err,
		stdoutStyled: shouldUseANSI(w),
		stderrStyled: shouldUseANSI(err),
	}
}

func (r *textReporter) SetCommand(command string) {
	r.command = command
}

func (r *textReporter) PrintIssues(issues []validate.Issue) error {
	for _, issue := range issues {
		if issue.Severity == validate.Error {
			fmt.Fprintln(r.err, r.styleError(issue.String()))
		} else if issue.Severity == validate.Warning {
			fmt.Fprintln(r.w, r.styleWarning(issue.String()))
		}
	}
	return nil
}

func (r *textReporter) PrintValidationResult(issues []validate.Issue) error {
	errorsCount := 0
	warningsCount := 0

	for _, issue := range issues {
		if issue.Severity == validate.Error {
			errorsCount++
		} else if issue.Severity == validate.Warning {
			warningsCount++
		}
	}

	_ = r.PrintIssues(issues)

	if len(issues) > 0 {
		fmt.Fprintf(
			r.w,
			"\n%s\n",
			r.styleSummary(fmt.Sprintf("Итог: Валидация завершена. Ошибок: %d, предупреждений: %d.", errorsCount, warningsCount)),
		)
	} else {
		fmt.Fprintln(r.w, r.styleSuccess("Готово: Валидация прошла успешно. Ошибок не обнаружено."))
	}
	return nil
}

func (r *textReporter) PrintSummary(archivePath string) error {
	fmt.Fprintf(
		r.w,
		"%s\n%s\n",
		r.styleSuccess("Готово: Сборка успешно завершена!"),
		r.styleSummary(fmt.Sprintf("Итог: Архив создан: %s", archivePath)),
	)
	return nil
}

func (r *textReporter) PrintConfigError(err error) error {
	fmt.Fprintf(r.err, "%s\n", r.styleError(fmt.Sprintf("Ошибка конфигурации: %v", err)))
	return nil
}

func (r *textReporter) PrintSuccess(msg string) error {
	fmt.Fprintf(r.w, "%s\n", r.styleSuccess(fmt.Sprintf("Готово: %s", msg)))
	return nil
}

func (r *textReporter) PrintInfo(msg string) error {
	fmt.Fprintf(r.w, "%s\n", r.styleInfo(msg))
	return nil
}

func (r *textReporter) PrintDryRunPlan(cfg config.Config, archivePath string) error {
	fmt.Fprintf(r.w, "\n%s\n", r.styleSummary("--- ПЛАН СБОРКИ (DRY RUN) ---"))
	fmt.Fprintf(r.w, "Модуль:      %s (версия %s)\n", cfg.Module.ID, cfg.Module.Version)
	fmt.Fprintf(r.w, "Исходники:   %s\n", cfg.Build.SourceDir)
	fmt.Fprintf(r.w, "Staging:     %s\n", cfg.Build.StagingDir)
	fmt.Fprintf(r.w, "Output:      %s\n", cfg.Build.OutputDir)
	fmt.Fprintf(r.w, "Имя архива:  %s\n", archivePath)

	if len(cfg.Exclude) > 0 {
		fmt.Fprintf(r.w, "\n%s\n", r.styleSummary("Исключения:"))
		for _, exc := range cfg.Exclude {
			fmt.Fprintf(r.w, "  - %s\n", exc)
		}
	}
	fmt.Fprintf(r.w, "%s\n", r.styleSummary("----------------------------"))
	fmt.Fprintln(r.w, "")
	fmt.Fprintln(r.w, r.styleInfo("Dry run завершен. Файлы не были изменены."))
	return nil
}

func (r *textReporter) PrintVersion(version string) error {
	fmt.Fprintf(r.w, "%s\n", r.styleSummary(fmt.Sprintf("Версия модуля: %s", version)))
	return nil
}

func (r *textReporter) PrintVersionBump(oldVersion, newVersion string) error {
	fmt.Fprintf(r.w, "%s\n", r.styleSuccess(fmt.Sprintf("Версия обновлена: %s -> %s", oldVersion, newVersion)))
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

func (r *jsonReporter) PrintValidationResult(issues []validate.Issue) error {
	return r.PrintIssues(issues)
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

func (r *jsonReporter) String() string { return "JSONReporter" }
func (r *textReporter) String() string { return "TextReporter" }

func shouldUseANSI(w io.Writer) bool {
	if w == nil {
		return false
	}
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}

	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}

func (r *textReporter) styleSuccess(msg string) string {
	return r.styleStdout(msg, "32")
}

func (r *textReporter) styleWarning(msg string) string {
	return r.styleStdout(msg, "33")
}

func (r *textReporter) styleInfo(msg string) string {
	return r.styleStdout(msg, "36")
}

func (r *textReporter) styleSummary(msg string) string {
	return r.styleStdout(msg, "1")
}

func (r *textReporter) styleError(msg string) string {
	return r.styleStderr(msg, "31")
}

func (r *textReporter) styleStdout(msg, code string) string {
	if !r.stdoutStyled {
		return msg
	}
	return ansiWrap(msg, code)
}

func (r *textReporter) styleStderr(msg, code string) string {
	if !r.stderrStyled {
		return msg
	}
	return ansiWrap(msg, code)
}

func ansiWrap(msg, code string) string {
	return "\x1b[" + code + "m" + msg + "\x1b[0m"
}
