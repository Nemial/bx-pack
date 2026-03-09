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
		fmt.Fprintln(r.err, issue.String())
		if issue.Severity == validate.Error {
			errorsCount++
		} else if issue.Severity == validate.Warning {
			warningsCount++
		}
	}

	if len(issues) > 0 {
		fmt.Fprintf(r.w, "\nВалидация завершена. Ошибок: %d, предупреждений: %d.\n", errorsCount, warningsCount)
	} else {
		fmt.Fprintln(r.w, "Валидация прошла успешно. Ошибок не обнаружено.")
	}
	return nil
}

func (r *textReporter) PrintSummary(archivePath string) error {
	fmt.Fprintf(r.w, "\nСборка успешно завершена!\nАрхив создан: %s\n", archivePath)
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

func NewReporter(format Format) Reporter {
	switch format {
	case JSONFormat:
		return NewJSONReporter(os.Stdout)
	default:
		return NewTextReporter(os.Stdout, os.Stderr)
	}
}
