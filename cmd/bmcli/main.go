package main

import (
	"os"

	"github.com/geoeee/bmcli/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
