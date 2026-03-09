package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"bx-pack/internal/validate"
)

type Format string

const (
	TextFormat Format = "text"
	JSONFormat Format = "json"
)

type Reporter interface {
	PrintIssues(issues []validate.Issue) error
	PrintSummary(archivePath string) error
	PrintConfigError(err error) error
	PrintSuccess(msg string) error
	PrintInfo(msg string) error
}

type textReporter struct {
	w   io.Writer
	err io.Writer
}

func NewTextReporter(w, err io.Writer) Reporter {
	return &textReporter{w: w, err: err}
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

type jsonReporter struct {
	w io.Writer
}

func NewJSONReporter(w io.Writer) Reporter {
	return &jsonReporter{w: w}
}

func (r *jsonReporter) PrintIssues(issues []validate.Issue) error {
	if issues == nil {
		issues = []validate.Issue{}
	}
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(issues); err != nil {
		return fmt.Errorf("encode issues to JSON: %w", err)
	}
	return nil
}

func (r *jsonReporter) PrintSummary(archivePath string) error {
	summary := struct {
		Success     bool   `json:"success"`
		ArchivePath string `json:"archivePath"`
	}{
		Success:     true,
		ArchivePath: archivePath,
	}
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

func (r *jsonReporter) PrintConfigError(err error) error {
	msg := struct {
		Error string `json:"error"`
		Type  string `json:"type"`
	}{
		Error: err.Error(),
		Type:  "CONFIG_ERROR",
	}
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(msg)
}

func (r *jsonReporter) PrintSuccess(msg string) error {
	res := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: true,
		Message: msg,
	}
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(res)
}

func (r *jsonReporter) PrintInfo(msg string) error {
	// Для JSON просто ничего не выводим или выводим как лог,
	// но обычно CLI инструменты в JSON моде не должны спамить инфо сообщениями в stdout.
	return nil
}

func NewReporter(format Format) Reporter {
	switch format {
	case JSONFormat:
		return NewJSONReporter(os.Stdout)
	default:
		return NewTextReporter(os.Stdout, os.Stderr)
	}
}
