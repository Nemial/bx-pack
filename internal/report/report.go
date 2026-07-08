package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Nemial/bx-pack/internal/config"
	"github.com/Nemial/bx-pack/internal/validate"
)

type Format string

const (
	TextFormat Format = "text"
	JSONFormat Format = "json"
)

type CommandReporter interface {
	SetCommand(command string)
}

type ConfigErrorReporter interface {
	PrintConfigError(err error) error
}

type SuccessReporter interface {
	PrintSuccess(msg string) error
}

type InfoReporter interface {
	PrintInfo(msg string) error
}

type IssuesReporter interface {
	PrintIssues(issues []validate.Issue) error
}

type ValidationResultReporter interface {
	PrintValidationResult(issues []validate.Issue) error
}

type SummaryReporter interface {
	PrintSummary(archivePath string) error
}

type DryRunReporter interface {
	PrintDryRunPlan(cfg config.Config, archivePath string) error
}

type ModuleVersionReporter interface {
	PrintVersion(version string) error
	PrintVersionBump(oldVersion, newVersion string) error
}

type Finalizer interface {
	Finalize() error
}

type InitReporter interface {
	CommandReporter
	ConfigErrorReporter
	SuccessReporter
}

type ScaffoldReporter interface {
	CommandReporter
	InfoReporter
	SuccessReporter
}

type ValidationReporter interface {
	CommandReporter
	ConfigErrorReporter
	SuccessReporter
	IssuesReporter
	ValidationResultReporter
}

type BuildReporter interface {
	CommandReporter
	ConfigErrorReporter
	InfoReporter
	IssuesReporter
	DryRunReporter
	SummaryReporter
}

type VersionReporter interface {
	CommandReporter
	ConfigErrorReporter
	ModuleVersionReporter
}

type Reporter struct {
	setCommand            func(command string)
	printIssues           func(issues []validate.Issue) error
	printValidationResult func(issues []validate.Issue) error
	printSummary          func(archivePath string) error
	printConfigError      func(err error) error
	printSuccess          func(msg string) error
	printInfo             func(msg string) error
	printDryRunPlan       func(cfg config.Config, archivePath string) error
	printVersion          func(version string) error
	printVersionBump      func(oldVersion, newVersion string) error
	finalize              func() error
}

func (r *Reporter) SetCommand(command string) {
	r.setCommand(command)
}

func (r *Reporter) PrintIssues(issues []validate.Issue) error {
	return r.printIssues(issues)
}

func (r *Reporter) PrintValidationResult(issues []validate.Issue) error {
	return r.printValidationResult(issues)
}

func (r *Reporter) PrintSummary(archivePath string) error {
	return r.printSummary(archivePath)
}

func (r *Reporter) PrintConfigError(err error) error {
	return r.printConfigError(err)
}

func (r *Reporter) PrintSuccess(msg string) error {
	return r.printSuccess(msg)
}

func (r *Reporter) PrintInfo(msg string) error {
	return r.printInfo(msg)
}

func (r *Reporter) PrintDryRunPlan(cfg config.Config, archivePath string) error {
	return r.printDryRunPlan(cfg, archivePath)
}

func (r *Reporter) PrintVersion(version string) error {
	return r.printVersion(version)
}

func (r *Reporter) PrintVersionBump(oldVersion, newVersion string) error {
	return r.printVersionBump(oldVersion, newVersion)
}

func (r *Reporter) Finalize() error {
	return r.finalize()
}

type textReporter struct {
	w            io.Writer
	err          io.Writer
	command      string
	stdoutStyled bool
	stderrStyled bool
}

func NewTextReporter(w, err io.Writer) *Reporter {
	impl := &textReporter{
		w:            w,
		err:          err,
		stdoutStyled: shouldUseANSI(w),
		stderrStyled: shouldUseANSI(err),
	}

	return &Reporter{
		setCommand:            impl.SetCommand,
		printIssues:           impl.PrintIssues,
		printValidationResult: impl.PrintValidationResult,
		printSummary:          impl.PrintSummary,
		printConfigError:      impl.PrintConfigError,
		printSuccess:          impl.PrintSuccess,
		printInfo:             impl.PrintInfo,
		printDryRunPlan:       impl.PrintDryRunPlan,
		printVersion:          impl.PrintVersion,
		printVersionBump:      impl.PrintVersionBump,
		finalize:              impl.Finalize,
	}
}

func (r *textReporter) SetCommand(command string) {
	r.command = command
}

func (r *textReporter) PrintIssues(issues []validate.Issue) error {
	for _, issue := range issues {
		switch issue.Severity {
		case validate.Error:
			if _, err := fmt.Fprintln(r.err, r.styleError(issue.String())); err != nil {
				return err
			}
		case validate.Warning:
			if _, err := fmt.Fprintln(r.w, r.styleWarning(issue.String())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *textReporter) PrintValidationResult(issues []validate.Issue) error {
	errorsCount := 0
	warningsCount := 0

	for _, issue := range issues {
		switch issue.Severity {
		case validate.Error:
			errorsCount++
		case validate.Warning:
			warningsCount++
		}
	}

	_ = r.PrintIssues(issues)

	if len(issues) > 0 {
		_, err := fmt.Fprintf(
			r.w,
			"\n%s\n",
			r.styleSummary(fmt.Sprintf("Итог: Валидация завершена. Ошибок: %d, предупреждений: %d.", errorsCount, warningsCount)),
		)
		if err != nil {
			return err
		}
	} else {
		_, err := fmt.Fprintln(r.w, r.styleSuccess("Готово: Валидация прошла успешно. Ошибок не обнаружено."))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *textReporter) PrintSummary(archivePath string) error {
	_, err := fmt.Fprintf(
		r.w,
		"%s\n%s\n",
		r.styleSuccess("Готово: Сборка успешно завершена!"),
		r.styleSummary("Итог: Архив создан: "+archivePath),
	)
	return err
}

func (r *textReporter) PrintConfigError(err error) error {
	_, writeErr := fmt.Fprintf(r.err, "%s\n", r.styleError(fmt.Sprintf("Ошибка конфигурации: %v", err)))
	return writeErr
}

func (r *textReporter) PrintSuccess(msg string) error {
	_, err := fmt.Fprintf(r.w, "%s\n", r.styleSuccess("Готово: "+msg))
	return err
}

func (r *textReporter) PrintInfo(msg string) error {
	_, err := fmt.Fprintf(r.w, "%s\n", r.styleInfo(msg))
	return err
}

func (r *textReporter) PrintDryRunPlan(cfg config.Config, archivePath string) error {
	if _, err := fmt.Fprintf(r.w, "\n%s\n", r.styleSummary("--- ПЛАН СБОРКИ (DRY RUN) ---")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.w, "Модуль:      %s (версия %s)\n", cfg.Module.ID, cfg.Module.Version); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.w, "Исходники:   %s\n", cfg.Build.SourceDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.w, "Staging:     %s\n", cfg.Build.StagingDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.w, "Output:      %s\n", cfg.Build.OutputDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.w, "Имя архива:  %s\n", archivePath); err != nil {
		return err
	}

	if len(cfg.Exclude) > 0 {
		if _, err := fmt.Fprintf(r.w, "\n%s\n", r.styleSummary("Исключения:")); err != nil {
			return err
		}
		for _, exc := range cfg.Exclude {
			if _, err := fmt.Fprintf(r.w, "  - %s\n", exc); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(r.w, "%s\n", r.styleSummary("----------------------------")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(r.w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(r.w, r.styleInfo("Dry run завершен. Файлы не были изменены.")); err != nil {
		return err
	}
	return nil
}

func (r *textReporter) PrintVersion(version string) error {
	_, err := fmt.Fprintf(r.w, "%s\n", r.styleSummary("Версия модуля: "+version))
	return err
}

func (r *textReporter) PrintVersionBump(oldVersion, newVersion string) error {
	_, err := fmt.Fprintf(r.w, "%s\n", r.styleSuccess(fmt.Sprintf("Версия обновлена: %s -> %s", oldVersion, newVersion)))
	return err
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

func NewJSONReporter(w io.Writer) *Reporter {
	impl := &jsonReporter{
		w: w,
		report: JSONReport{
			Success: true,
		},
	}

	return &Reporter{
		setCommand:            impl.SetCommand,
		printIssues:           impl.PrintIssues,
		printValidationResult: impl.PrintValidationResult,
		printSummary:          impl.PrintSummary,
		printConfigError:      impl.PrintConfigError,
		printSuccess:          impl.PrintSuccess,
		printInfo:             impl.PrintInfo,
		printDryRunPlan:       impl.PrintDryRunPlan,
		printVersion:          impl.PrintVersion,
		printVersionBump:      impl.PrintVersionBump,
		finalize:              impl.Finalize,
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
		switch issue.Severity {
		case validate.Error:
			r.report.Success = false
			r.report.Errors = append(r.report.Errors, issue.Message)
		case validate.Warning:
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
	r.report.Summary = "Архив создан: " + archivePath
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
	r.report.Summary = "Версия модуля: " + version
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

func NewReporter(format Format) *Reporter {
	return NewReporterWithWriter(format, os.Stdout, os.Stderr)
}

func NewReporterWithWriter(format Format, w, err io.Writer) *Reporter {
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
