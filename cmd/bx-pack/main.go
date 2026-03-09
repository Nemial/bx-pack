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
	formatFlag := flag.String("format", "text", "Формат вывода (text, json)")
	fShort := flag.String("f", "", "Формат вывода (text, json) - сокращенно")
	flag.Usage = printUsage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(ExitError)
	}

	format := report.Format(*formatFlag)
	if *fShort != "" {
		format = report.Format(*fShort)
	}

	if format != report.JSONFormat && format != report.TextFormat {
		fmt.Fprintf(os.Stderr, "Ошибка: неизвестный формат %q\n", format)
		os.Exit(ExitError)
	}

	reporter := report.NewReporter(format)
	command := args[0]
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

	if err != nil {
		// Ошибка уже выведена через репортер внутри функций cli,
		// или нам нужно вывести ее здесь, если она "внешняя" (например, ошибка конфига)
		// Для единообразия, CLI функции должны сами выводить свои ошибки через репортер
		// или мы проверяем тип ошибки.
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
