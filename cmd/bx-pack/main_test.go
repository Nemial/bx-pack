package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpIsLocalized(t *testing.T) {
	rootCmd := newRootCmd()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	if err := rootCmd.Help(); err != nil {
		t.Fatalf("Help failed: %v", err)
	}

	help := out.String()
	for _, want := range []string{
		"Использование:",
		"Команды:",
		"Показать справку по команде",
		"Примеры:",
		"bx-pack build --dry-run",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help should contain %q, got:\n%s", want, help)
		}
	}

	if strings.Contains(help, "Generate the autocompletion script") {
		t.Errorf("help should not contain default completion command, got:\n%s", help)
	}
}

func TestBuildUsageIsLocalized(t *testing.T) {
	buildCmd, _, err := newRootCmd().Find([]string{"build"})
	if err != nil {
		t.Fatalf("Find build command failed: %v", err)
	}

	var out bytes.Buffer
	buildCmd.SetOut(&out)
	buildCmd.SetErr(&out)

	if err := buildCmd.Usage(); err != nil {
		t.Fatalf("Usage failed: %v", err)
	}

	usage := out.String()
	for _, want := range []string{
		"Использование:",
		"bx-pack build",
		"Примеры:",
		"bx-pack build --dry-run",
		"Глобальные флаги:",
	} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage should contain %q, got:\n%s", want, usage)
		}
	}
}
