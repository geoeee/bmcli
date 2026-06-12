package cli

import (
	"encoding/json"
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
	args, jsonOutput := parseGlobalFlags(args)

	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "version", "-v", "--version":
		if jsonOutput {
			if err := printVersionJSON(stdout); err != nil {
				fmt.Fprintf(stderr, "%s: failed to write JSON output: %v\n", appName, err)
				return 1
			}
			return 0
		}
		fmt.Fprintf(stdout, "%s %s\n", appName, versionString())
		return 0
	default:
		fmt.Fprintf(stderr, "%s: unknown command %q\n\n", appName, args[0])
		printHelp(stderr)
		return 1
	}
}

func parseGlobalFlags(args []string) ([]string, bool) {
	remaining := make([]string, 0, len(args))
	jsonOutput := false

	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			continue
		}
		remaining = append(remaining, arg)
	}

	return remaining, jsonOutput
}

func printVersionJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(versionOutput{
		Name:    appName,
		Version: effectiveVersion(),
		Commit:  Commit,
		Date:    Date,
	})
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
      --json      Output supported commands as JSON
  -v, --version   Print the %s version
`, appName, appName, appName, appName, appName)
}

func versionString() string {
	version := effectiveVersion()

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

func effectiveVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

type versionOutput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
}
