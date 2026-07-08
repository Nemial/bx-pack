package cli

import (
	"fmt"
	"strings"

	"github.com/Nemial/bx-pack/internal/report"

	"github.com/spf13/cobra"
)

const (
	ansiBold  = "\x1b[1m"
	ansiCyan  = "\x1b[36m"
	ansiReset = "\x1b[0m"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

func SetBuildInfo(v, c, d string) {
	buildVersion = v
	buildCommit = c
	buildDate = d
}

func Run(args []string) int {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		exitCode := GetExitCode(err)
		if exitCode == ExitUsageErr {
			cmd, _, _ := rootCmd.Find(args)
			if cmd != nil {
				_ = cmd.Usage()
			} else {
				_ = rootCmd.Usage()
			}
			_, _ = fmt.Fprintf(rootCmd.ErrOrStderr(), "\nОшибка: %v\n", err)
		}
		return exitCode
	}

	return ExitSuccess
}

func NewRootCmd() *cobra.Command {
	var (
		formatStr        string
		dryRun           bool
		initReporter     report.InitReporter
		scaffoldReporter report.ScaffoldReporter
		validateReporter report.ValidationReporter
		buildReporter    report.BuildReporter
		versionReporter  report.VersionReporter
		finalizer        report.Finalizer
	)

	rootCmd := &cobra.Command{
		Use:     "bx-pack",
		Short:   "bx-pack — инструмент для сборки модулей Bitrix",
		Long:    "bx-pack — CLI для проверки, подготовки и упаковки Bitrix-модулей.",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", buildVersion, buildCommit, buildDate),
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
				return NewCLIError(ExitUsageErr, fmt.Errorf("неизвестный формат %q", formatStr))
			}
			r := report.NewReporter(format)
			initReporter = r
			scaffoldReporter = r
			validateReporter = r
			buildReporter = r
			versionReporter = r
			finalizer = r
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if finalizer != nil {
				_ = finalizer.Finalize()
			}
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	configureHelp(rootCmd)

	rootCmd.PersistentFlags().StringVarP(&formatStr, "format", "f", "text", "Формат вывода (text, json)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Показать план без создания файлов")

	rootCmd.AddCommand(newInitCmd(&initReporter))
	rootCmd.AddCommand(newScaffoldCmd(&scaffoldReporter, &dryRun))
	rootCmd.AddCommand(newValidateCmd(&validateReporter))
	rootCmd.AddCommand(newBuildCmd(&buildReporter, &dryRun))
	rootCmd.AddCommand(newVersionCmd(&versionReporter))
	localizeHelpFlags(rootCmd)

	return rootCmd
}

func newInitCmd(reporter *report.InitReporter) *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Short:   "Инициализировать новый проект",
		Example: "  bx-pack init",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Init(*reporter)
		},
	}
}

func newScaffoldCmd(reporter *report.ScaffoldReporter, dryRun *bool) *cobra.Command {
	return &cobra.Command{
		Use:     "scaffold",
		Short:   "Создать структуру Bitrix-модуля",
		Example: "  bx-pack scaffold",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Scaffold(*reporter, *dryRun)
		},
	}
}

func newValidateCmd(reporter *report.ValidationReporter) *cobra.Command {
	return &cobra.Command{
		Use:     "validate",
		Short:   "Проверить конфигурацию",
		Example: "  bx-pack validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Validate(*reporter)
		},
	}
}

func newBuildCmd(reporter *report.BuildReporter, dryRun *bool) *cobra.Command {
	return &cobra.Command{
		Use:     "build",
		Short:   "Собрать архив проекта",
		Example: "  bx-pack build\n  bx-pack build --dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Build(*reporter, *dryRun)
		},
	}
}

func newVersionCmd(reporter *report.VersionReporter) *cobra.Command {
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
			return VersionShow(*reporter)
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

			if !isValidBumpLevel(level) {
				return fmt.Errorf("неизвестный уровень инкремента: %q. Допустимые: patch, minor, major, auto", level)
			}
			return VersionBump(*reporter, level)
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

	cobra.AddTemplateFunc("formatExample", formatExample)
	cobra.AddTemplateFunc("formatFlagUsages", formatFlagUsages)
	cobra.AddTemplateFunc("commandText", commandText)
	cobra.AddTemplateFunc("helpHeading", helpHeading)
	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.SetHelpTemplate(helpTemplate)
}

func helpHeading(text string) string {
	return ansiBold + ansiCyan + text + ansiReset
}

func commandText(text string) string {
	return ansiBold + text + ansiReset
}

func formatExample(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		lines[i] = "  " + commandText(strings.TrimSpace(line))
	}
	return strings.Join(lines, "\n")
}

func formatFlagUsages(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	maxFlagWidth := 0
	for _, line := range lines {
		_, flagPart, _, ok := splitFlagUsageLine(line)
		if !ok {
			continue
		}
		if len(flagPart) > maxFlagWidth {
			maxFlagWidth = len(flagPart)
		}
	}

	for i, line := range lines {
		lines[i] = formatFlagUsageLine(line, maxFlagWidth)
	}
	return strings.Join(lines, "\n")
}

func formatFlagUsageLine(line string, maxFlagWidth int) string {
	_, flagPart, descPart, ok := splitFlagUsageLine(line)
	if !ok {
		trimmed := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			return trimmed
		}
		return trimmed
	}

	const (
		baseIndent = "  "
		padding    = 3
	)

	extraPadding := 0
	if maxFlagWidth > len(flagPart) {
		extraPadding = maxFlagWidth - len(flagPart)
	}

	return baseIndent + commandText(flagPart) + strings.Repeat(" ", padding+extraPadding) + descPart
}

func splitFlagUsageLine(line string) (string, string, string, bool) {
	trimmed := strings.TrimRight(line, " \t")
	if strings.TrimSpace(trimmed) == "" {
		return trimmed, "", "", false
	}

	indentWidth := len(trimmed) - len(strings.TrimLeft(trimmed, " "))
	indent := trimmed[:indentWidth]
	body := trimmed[indentWidth:]
	split := findFlagUsageSplit(body)
	if split < 0 {
		return indent, strings.TrimSpace(body), "", false
	}

	flagPart := strings.TrimRight(body[:split], " ")
	descPart := strings.TrimLeft(body[split:], " ")
	return indent, flagPart, descPart, true
}

func findFlagUsageSplit(text string) int {
	spaceRun := 0
	for i, r := range text {
		if r == ' ' {
			spaceRun++
			if spaceRun >= 2 {
				return i - spaceRun + 1
			}
			continue
		}
		spaceRun = 0
	}
	return -1
}

func localizeHelpFlags(rootCmd *cobra.Command) {
	var visit func(cmd *cobra.Command)
	visit = func(cmd *cobra.Command) {
		cmd.InitDefaultHelpFlag()
		if flag := cmd.Flags().Lookup("help"); flag != nil {
			flag.Usage = "Показать справку" + helpFlagSuffix(cmd)
		}
		for _, subCmd := range cmd.Commands() {
			visit(subCmd)
		}
	}
	visit(rootCmd)
}

func helpFlagSuffix(cmd *cobra.Command) string {
	if cmd == nil || cmd.Name() == "" {
		return ""
	}
	return " для " + cmd.CommandPath()
}

func isValidBumpLevel(level string) bool {
	switch level {
	case "patch", "minor", "major", "auto":
		return true
	default:
		return false
	}
}

const usageTemplate = `{{if .Runnable}}{{helpHeading "Использование:"}}
  {{commandText .UseLine}}{{else if .HasAvailableSubCommands}}{{helpHeading "Использование:"}}
  {{commandText (printf "%s [команда]" .CommandPath)}}{{end}}{{if gt (len .Aliases) 0}}

{{helpHeading "Псевдонимы:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{helpHeading "Примеры:"}}
{{formatExample .Example}}{{end}}{{if .HasAvailableSubCommands}}

{{helpHeading "Команды:"}}
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad (commandText .Name) 24 }} {{.Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{helpHeading "Флаги:"}}
{{formatFlagUsages (.LocalFlags.FlagUsages | trimTrailingWhitespaces)}}{{end}}{{if .HasAvailableInheritedFlags}}

{{helpHeading "Глобальные флаги:"}}
{{formatFlagUsages (.InheritedFlags.FlagUsages | trimTrailingWhitespaces)}}{{end}}
`

const helpTemplate = `{{with (or .Long .Short)}}{{.}}{{end}}

{{if .Runnable}}{{helpHeading "Использование:"}}
  {{commandText .UseLine}}{{else if .HasAvailableSubCommands}}{{helpHeading "Использование:"}}
  {{commandText (printf "%s [команда]" .CommandPath)}}{{end}}{{if gt (len .Aliases) 0}}

{{helpHeading "Псевдонимы:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{helpHeading "Примеры:"}}
{{formatExample .Example}}{{end}}{{if .HasAvailableSubCommands}}

{{helpHeading "Команды:"}}
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad (commandText .Name) 24 }} {{.Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{helpHeading "Флаги:"}}
{{formatFlagUsages (.LocalFlags.FlagUsages | trimTrailingWhitespaces)}}{{end}}{{if .HasAvailableInheritedFlags}}

{{helpHeading "Глобальные флаги:"}}
{{formatFlagUsages (.InheritedFlags.FlagUsages | trimTrailingWhitespaces)}}{{end}}{{if .HasAvailableSubCommands}}

Используйте "{{commandText (printf "%s [команда] --help" .CommandPath)}}" для подробностей о команде.{{end}}`
