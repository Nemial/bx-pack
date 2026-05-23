package main

import (
	"bx-pack/internal/cli"

	"os"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
