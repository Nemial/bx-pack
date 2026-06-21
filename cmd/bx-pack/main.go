package main

import (
	"bx-pack/internal/cli"

	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetBuildInfo(version, commit, date)
	os.Exit(cli.Run(os.Args[1:]))
}
