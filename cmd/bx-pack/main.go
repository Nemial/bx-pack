package main

import (
	"fmt"
	"os"
	"strings"

	"bx-pack/internal/cli"
	"bx-pack/internal/report"

	"github.com/spf13/cobra"
)

var (
	formatStr string
	dryRun    bool
	reporter  report.Reporter
)

const (
	ansiBold  = "\x1b[1m"
	ansiCyan  = "\x1b[36m"
	ansiReset = "\x1b[0m"
)

func helpHeading(text string) string {
	return ansiBold + ansiCyan + text + ansiReset
}

func main() {
	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	rootCmd := newRootCmd()

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

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bx-pack",
		Short: "bx-pack — инструмент для сборки модулей Bitrix",
		Long:  "bx-pack — CLI для проверки, подготовки и упаковки Bitrix-модулей.",
		Example: strings.TrimSpace(`
  bx-pack init
  bx-pack scaffold
  bx-pack validate
  bx-pack build --dry-run
  bx-pack version show
  bx-pack version bump patch`),
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

	configureHelp(rootCmd)

	rootCmd.PersistentFlags().StringVarP(&formatStr, "format", "f", "text", "Формат вывода (text, json)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Показать план без создания файлов")

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newScaffoldCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newBuildCmd())
	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Short:   "Инициализировать новый проект",
		Example: "  bx-pack init",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Init(reporter)
		},
	}
}

func newScaffoldCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "scaffold",
		Short:   "Создать структуру Bitrix-модуля",
		Example: "  bx-pack scaffold",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Scaffold(reporter, dryRun)
		},
	}
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "validate",
		Short:   "Проверить конфигурацию",
		Example: "  bx-pack validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Validate(reporter)
		},
	}
}

func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "build",
		Short:   "Собрать архив проекта",
		Example: "  bx-pack build\n  bx-pack build --dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Build(reporter, dryRun)
		},
	}
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Управление версией модуля",
		Example: "  bx-pack version show\n  bx-pack version bump auto",
	}

	showCmd := &cobra.Command{
		Use:     "show",
		Short:   "Показать текущую версию модуля",
		Example: "  bx-pack version show",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.VersionShow(reporter)
		},
	}

	bumpCmd := &cobra.Command{
		Use:     "bump [patch|minor|major|auto]",
		Short:   "Инкрементировать версию (по умолчанию auto)",
		Example: "  bx-pack version bump patch\n  bx-pack version bump auto",
		Args:    cobra.MaximumNArgs(1),
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

func configureHelp(rootCmd *cobra.Command) {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	helpCmd := &cobra.Command{
		Use:   "help [команда]",
		Short: "Показать справку по команде",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := cmd.Root()
			if len(args) > 0 {
				found, _, err := cmd.Root().Find(args)
				if err != nil {
					return err
				}
				target = found
			}

			return target.Help()
		},
	}

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.AddCommand(helpCmd)

	cobra.AddTemplateFunc("helpHeading", helpHeading)
	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.SetHelpTemplate(helpTemplate)
}

const usageTemplate = `{{if .Runnable}}{{helpHeading "Использование:"}}
  {{.UseLine}}{{else if .HasAvailableSubCommands}}{{helpHeading "Использование:"}}
  {{.CommandPath}} [команда]{{end}}{{if gt (len .Aliases) 0}}

{{helpHeading "Псевдонимы:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{helpHeading "Примеры:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{helpHeading "Команды:"}}
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{helpHeading "Флаги:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{helpHeading "Глобальные флаги:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`

const helpTemplate = `{{with (or .Long .Short)}}{{.}}{{end}}

{{if .Runnable}}{{helpHeading "Использование:"}}
  {{.UseLine}}{{else if .HasAvailableSubCommands}}{{helpHeading "Использование:"}}
  {{.CommandPath}} [команда]{{end}}{{if gt (len .Aliases) 0}}

{{helpHeading "Псевдонимы:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{helpHeading "Примеры:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{helpHeading "Команды:"}}
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{helpHeading "Флаги:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{helpHeading "Глобальные флаги:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Используйте "{{.CommandPath}} [команда] --help" для подробностей о команде.{{end}}`
