package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"bx-pack/internal/cli"
	"bx-pack/internal/report"
)

const (
	ExitSuccess   = 0
	ExitError     = 1
	ExitValError  = 2
	ExitConfigErr = 3
)

func main() {
	var formatStr string
	var fShort string
	var dryRun bool

	fs := flag.NewFlagSet("bx-pack", flag.ContinueOnError)
	fs.StringVar(&formatStr, "format", "text", "Формат вывода (text, json)")
	fs.StringVar(&fShort, "f", "", "Формат вывода (text, json) - сокращенно")
	fs.BoolVar(&dryRun, "dry-run", false, "Показать план сборки без создания файлов")
	fs.Usage = printUsage

	// Сначала собираем все флаги, потом смотрим команду
	// Но стандартный flag.Parse() прекращает разбор после первого не-флага.
	// Поэтому мы должны сначала вытащить команду, или разрешить флаги где угодно.

	// Упрощенный вариант: ищем флаги во всех аргументах.
	for i, arg := range os.Args {
		if arg == "-f" && i+1 < len(os.Args) {
			fShort = os.Args[i+1]
		}
		if arg == "--format" && i+1 < len(os.Args) {
			formatStr = os.Args[i+1]
		}
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	// Парсим позиционные аргументы для поддержки подкоманд
	var positionalArgs []string
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) == 0 {
		printUsage()
		os.Exit(ExitError)
	}

	command := positionalArgs[0]

	format := report.Format(formatStr)
	if fShort != "" {
		format = report.Format(fShort)
	}

	if format != report.JSONFormat && format != report.TextFormat {
		fmt.Fprintf(os.Stderr, "Ошибка: неизвестный формат %q\n", format)
		os.Exit(ExitError)
	}

	reporter := report.NewReporter(format)
	var err error

	switch command {
	case "init":
		err = cli.Init(reporter)
	case "scaffold":
		err = cli.Scaffold(reporter, dryRun)
	case "validate":
		err = cli.Validate(reporter)
	case "version":
		if len(positionalArgs) < 2 {
			printUsage()
			os.Exit(ExitError)
		}
		subcmd := positionalArgs[1]
		if subcmd == "show" {
			if len(positionalArgs) > 2 {
				printUsage()
				os.Exit(ExitError)
			}
			err = cli.VersionShow(reporter)
		} else if subcmd == "bump" {
			if len(positionalArgs) < 3 {
				printUsage()
				os.Exit(ExitError)
			}
			level := positionalArgs[2]
			if level != "patch" && level != "minor" && level != "major" {
				fmt.Fprintf(os.Stderr, "Неизвестный уровень инкремента: %q. Используйте: patch, minor, major\n", level)
				printUsage()
				os.Exit(ExitError)
			}
			err = cli.VersionBump(reporter, level)
		} else {
			fmt.Fprintf(os.Stderr, "Неизвестная подкоманда version: %q\n", subcmd)
			printUsage()
			os.Exit(ExitError)
		}
	case "build":
		err = cli.Build(reporter, dryRun)
	case "help":
		printUsage()
		os.Exit(ExitSuccess)
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная команда: %s\n", command)
		printUsage()
		os.Exit(ExitError)
	}

	reporter.Finalize()

	if err != nil {
		os.Exit(ExitValError)
	}
}

func printUsage() {
	fmt.Println("Использование: bx-pack [флаги] [команда [подкоманда] [аргументы]]")
	fmt.Println("\nФлаги:")
	fmt.Println("  -f, --format string   Формат вывода: text (по умолчанию), json")
	fmt.Println("      --dry-run         Показать план сборки без создания файлов")
	fmt.Println("\nКоманды:")
	fmt.Println("  init      Инициализировать новый проект со стандартной конфигурацией")
	fmt.Println("  scaffold  Создать базовую структуру Bitrix-модуля")
	fmt.Println("  validate  Проверить конфигурацию проекта")
	fmt.Println("  build     Собрать архив проекта")
	fmt.Println("  version show             Показать текущую версию модуля")
	fmt.Println("  version bump <patch|minor|major>  Инкрементировать версию SemVer и обновить VERSION_DATE")
	fmt.Println("  help      Показать это справочное сообщение")
}
