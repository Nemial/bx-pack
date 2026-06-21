package report

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestTextReporter_UsesANSIWhenEnabled(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := &textReporter{
		w:            &stdout,
		err:          &stderr,
		stdoutStyled: true,
		stderrStyled: true,
	}

	if err := r.PrintSuccess("Операция выполнена"); err != nil {
		t.Fatal(err)
	}
	if err := r.PrintConfigError(errors.New("broken")); err != nil {
		t.Fatal(err)
	}

	if got := stdout.String(); !strings.Contains(got, "\x1b[32mГотово: Операция выполнена\x1b[0m") {
		t.Errorf("stdout should contain green ANSI sequence, got %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "\x1b[31mОшибка конфигурации: broken\x1b[0m") {
		t.Errorf("stderr should contain red ANSI sequence, got %q", got)
	}
}
