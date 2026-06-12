package cli

import (
	"fmt"
	"io"
)

const (
	appName = "bmcli"
)

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "%s %s\n", appName, versionString())
		return 0
	default:
		fmt.Fprintf(stderr, "%s: unknown command %q\n\n", appName, args[0])
		printHelp(stderr)
		return 1
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `%s is a bastion management command line interface.

Usage:
  %s <command>

Available Commands:
  help      Show help for %s
  version   Print the %s version

Flags:
  -h, --help      Show help
  -v, --version   Print the %s version
`, appName, appName, appName, appName, appName)
}

func versionString() string {
	version := Version
	if version == "" {
		version = "dev"
	}

	if Commit != "" {
		version += " (" + Commit
		if Date != "" {
			version += ", " + Date
		}
		version += ")"
	} else if Date != "" {
		version += " (" + Date + ")"
	}

	return version
}
