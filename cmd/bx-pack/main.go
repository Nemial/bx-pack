package main

import (
	"bx-pack/internal/cli"
	"bx-pack/internal/report"
	"flag"
	"fmt"
	"os"
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

	command := args[0]
	var err error

	switch command {
	case "init":
		err = cli.Init()
	case "validate":
		err = cli.Validate(format)
	case "build":
		err = cli.Build(format)
	case "help":
		printUsage()
		os.Exit(ExitSuccess)
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная команда: %s\n", command)
		printUsage()
		os.Exit(ExitError)
	}

	if err != nil {
		if format == report.TextFormat {
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		}
		// В будущем можно добавить типизированные ошибки для разных кодов
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
