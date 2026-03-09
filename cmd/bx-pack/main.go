package main

import (
	"flag"
	"fmt"
	"os"

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

	fs := flag.NewFlagSet("bx-pack", flag.ContinueOnError)
	fs.StringVar(&formatStr, "format", "text", "Формат вывода (text, json)")
	fs.StringVar(&fShort, "f", "", "Формат вывода (text, json) - сокращенно")
	fs.Usage = printUsage

	// Сначала собираем все флаги, потом смотрим команду
	// Но стандартный flag.Parse() прекращает разбор после первого не-флага.
	// Поэтому мы должны сначала вытащить команду, или разрешить флаги где угодно.

	// Упрощенный вариант: ищем флаг -f или --format во всех аргументах.
	for i, arg := range os.Args {
		if arg == "-f" && i+1 < len(os.Args) {
			fShort = os.Args[i+1]
		}
		if arg == "--format" && i+1 < len(os.Args) {
			formatStr = os.Args[i+1]
		}
	}

	// Команда - это первый аргумент, не являющийся флагом
	var command string
	var commandFound bool
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-f" || arg == "--format" {
			i++ // пропустить значение
			continue
		}
		if arg[0] == '-' {
			continue // пропустить другие флаги
		}
		command = arg
		commandFound = true
		break
	}

	if !commandFound {
		printUsage()
		os.Exit(ExitError)
	}

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
	case "validate":
		err = cli.Validate(reporter)
	case "build":
		err = cli.Build(reporter)
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
	fmt.Println("Использование: bx-pack [флаги] <команда>")
	fmt.Println("\nФлаги:")
	fmt.Println("  -f, --format string   Формат вывода: text (по умолчанию), json")
	fmt.Println("\nКоманды:")
	fmt.Println("  init      Инициализировать новый проект со стандартной конфигурацией")
	fmt.Println("  validate  Проверить конфигурацию проекта")
	fmt.Println("  build     Собрать архив проекта")
	fmt.Println("  help      Показать это справочное сообщение")
}
