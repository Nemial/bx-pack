package main

import (
	"fmt"
	"os"

	"bx-pack/internal/cli"
	"bx-pack/internal/report"

	"github.com/spf13/cobra"
)

const (
	ExitSuccess   = 0
	ExitError     = 1
	ExitValError  = 2
	ExitConfigErr = 3
)

var (
	formatStr string
	dryRun    bool
	reporter  report.Reporter
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "bx-pack",
		Short: "bx-pack — инструмент для сборки модулей Bitrix",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			format := report.Format(formatStr)
			if format != report.JSONFormat && format != report.TextFormat {
				fmt.Fprintf(os.Stderr, "Ошибка: неизвестный формат %q\n", formatStr)
				os.Exit(ExitError)
			}
			reporter = report.NewReporter(format)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if reporter != nil {
				reporter.Finalize()
			}
		},
	}

	// Глобальные флаги
	rootCmd.PersistentFlags().StringVarP(&formatStr, "format", "f", "text", "Формат вывода (text, json)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Показать план без создания файлов")

	// Команды
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newScaffoldCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newBuildCmd())
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitError)
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Инициализировать новый проект",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Init(reporter)
		},
	}
}

func newScaffoldCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold",
		Short: "Создать структуру Bitrix-модуля",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Scaffold(reporter, dryRun)
		},
	}
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Проверить конфигурацию",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Validate(reporter)
		},
	}
}

func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Собрать архив проекта",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Build(reporter, dryRun)
		},
	}
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Управление версией модуля",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Показать текущую версию модуля",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.VersionShow(reporter)
		},
	}

	bumpCmd := &cobra.Command{
		Use:   "bump <patch|minor|major>",
		Short: "Инкрементировать версию",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			level := args[0]
			if level != "patch" && level != "minor" && level != "major" {
				return fmt.Errorf("неизвестный уровень инкремента: %q", level)
			}
			return cli.VersionBump(reporter, level)
		},
	}

	cmd.AddCommand(showCmd, bumpCmd)
	return cmd
}
