package main

import (
	"fmt"
	"os"

	"bx-pack/internal/cli"
	"bx-pack/internal/report"

	"github.com/spf13/cobra"
)

var (
	formatStr string
	dryRun    bool
	reporter  report.Reporter
)

func main() {
	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	rootCmd := &cobra.Command{
		Use:   "bx-pack",
		Short: "bx-pack — инструмент для сборки модулей Bitrix",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			format := report.Format(formatStr)
			if format != report.JSONFormat && format != report.TextFormat {
				return cli.NewCLIError(cli.ExitUsageErr, fmt.Errorf("неизвестный формат %q", formatStr))
			}
			reporter = report.NewReporter(format)
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if reporter != nil {
				reporter.Finalize()
			}
		},
		SilenceErrors: true,
		SilenceUsage:  true,
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
		exitCode = cli.GetExitCode(err)
		if exitCode == cli.ExitUsageErr {
			cmd, _, _ := rootCmd.Find(os.Args[1:])
			if cmd != nil {
				cmd.Usage()
			} else {
				rootCmd.Usage()
			}
			fmt.Fprintf(os.Stderr, "\nОшибка: %v\n", err)
		}
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
		Use:   "bump [patch|minor|major|auto]",
		Short: "Инкрементировать версию (по умолчанию auto)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			level := "auto"
			if len(args) > 0 {
				level = args[0]
			}

			if level != "patch" && level != "minor" && level != "major" && level != "auto" {
				return fmt.Errorf("неизвестный уровень инкремента: %q. Допустимые: patch, minor, major, auto", level)
			}
			return cli.VersionBump(reporter, level)
		},
	}

	cmd.AddCommand(showCmd, bumpCmd)
	return cmd
}
