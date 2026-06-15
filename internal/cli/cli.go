package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	opts, err := parseGlobalFlags(args)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", appName, err)
		return 1
	}
	args = opts.args

	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "version", "-v", "--version":
		if opts.jsonOutput {
			if err := printVersionJSON(stdout); err != nil {
				fmt.Fprintf(stderr, "%s: failed to write JSON output: %v\n", appName, err)
				return 1
			}
			return 0
		}
		fmt.Fprintf(stdout, "%s %s\n", appName, versionString())
		return 0
	case "list":
		if err := runList(stdout, opts); err != nil {
			if opts.jsonOutput {
				_ = printErrorJSON(stderr, err)
			} else {
				fmt.Fprintf(stderr, "%s: %v\n", appName, err)
			}
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "%s: unknown command %q\n\n", appName, args[0])
		printHelp(stderr)
		return 1
	}
}

func parseGlobalFlags(args []string) (runOptions, error) {
	remaining := make([]string, 0, len(args))
	opts := runOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			opts.jsonOutput = true
			continue
		}
		if arg == "--config" {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return runOptions{}, errors.New("--config requires a path")
			}
			opts.configPath = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--config=") {
			opts.configPath = strings.TrimPrefix(arg, "--config=")
			if opts.configPath == "" {
				return runOptions{}, errors.New("--config requires a path")
			}
			continue
		}
		remaining = append(remaining, arg)
	}

	opts.args = remaining
	return opts, nil
}

func printVersionJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(versionOutput{
		Name:    appName,
		Version: effectiveVersion(),
		Commit:  Commit,
		Date:    Date,
	})
}

func printErrorJSON(w io.Writer, err error) error {
	return json.NewEncoder(w).Encode(errorOutput{
		Error: err.Error(),
	})
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `%s is a bastion management command line interface.

Usage:
  %s <command>

Available Commands:
  help      Show help for %s
  list      List configured bastions
  version   Print the %s version

Flags:
      --config    Path to config file
  -h, --help      Show help
      --json      Output supported commands as JSON
  -v, --version   Print the %s version
`, appName, appName, appName, appName, appName)
}

func runList(w io.Writer, opts runOptions) error {
	result, err := loadBastionList(opts.configPath)
	if err != nil {
		return err
	}

	if len(result.Bastions) == 0 {
		result.ConfigSuggestion = newConfigSuggestion(result.ConfigPath)
	}

	if opts.jsonOutput {
		return json.NewEncoder(w).Encode(result)
	}

	if len(result.Bastions) == 0 {
		fmt.Fprintln(w, "No bastions configured.")
		fmt.Fprintf(w, "Create a config file at: %s\n", result.ConfigPath)
		return nil
	}

	printBastionTable(w, result.Bastions)
	return nil
}

func loadBastionList(configPath string) (bastionListOutput, error) {
	resolvedPath, explicit, err := resolveConfigPath(configPath)
	if err != nil {
		return bastionListOutput{}, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return bastionListOutput{
				Bastions:   []bastionSummary{},
				Count:      0,
				ConfigPath: resolvedPath,
			}, nil
		}
		return bastionListOutput{}, fmt.Errorf("failed to read config %s: %w", resolvedPath, err)
	}

	var cfg bastionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return bastionListOutput{}, fmt.Errorf("invalid config %s: %w", resolvedPath, err)
	}

	bastions := make([]bastionSummary, 0, len(cfg.Bastions))
	for i, bastion := range cfg.Bastions {
		summary := bastion.summary()
		if summary.Name == "" {
			return bastionListOutput{}, fmt.Errorf("invalid config %s: bastions[%d].name is required", resolvedPath, i)
		}
		if summary.Hostname == "" {
			return bastionListOutput{}, fmt.Errorf("invalid config %s: bastions[%d].hostname is required", resolvedPath, i)
		}
		if summary.Port < 0 || summary.Port > 65535 {
			return bastionListOutput{}, fmt.Errorf("invalid config %s: bastions[%d].port must be between 0 and 65535", resolvedPath, i)
		}
		bastions = append(bastions, summary)
	}

	return bastionListOutput{
		Bastions:   bastions,
		Count:      len(bastions),
		ConfigPath: resolvedPath,
	}, nil
}

func resolveConfigPath(configPath string) (string, bool, error) {
	if configPath != "" {
		return filepath.Clean(configPath), true, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve user config directory: %w", err)
	}

	return filepath.Join(configDir, appName, "config.json"), false, nil
}

func newConfigSuggestion(configPath string) *configSuggestion {
	return &configSuggestion{
		Message:    "No bastions configured.",
		ConfigPath: configPath,
	}
}

func printBastionTable(w io.Writer, bastions []bastionSummary) {
	fmt.Fprintln(w, "NAME\tHOST\tPORT\tENV\tREGION")
	for _, bastion := range bastions {
		port := ""
		if bastion.Port != 0 {
			port = fmt.Sprintf("%d", bastion.Port)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", bastion.Name, bastion.Hostname, port, bastion.Environment, bastion.Region)
	}
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

type runOptions struct {
	args       []string
	jsonOutput bool
	configPath string
}

type bastionConfig struct {
	Bastions []bastionConfigEntry `json:"bastions"`
}

type bastionConfigEntry struct {
	Name        string            `json:"name"`
	Environment string            `json:"environment,omitempty"`
	Region      string            `json:"region,omitempty"`
	Hostname    string            `json:"hostname,omitempty"`
	Host        string            `json:"host,omitempty"`
	Port        int               `json:"port,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

func (entry bastionConfigEntry) summary() bastionSummary {
	hostname := entry.Hostname
	if hostname == "" {
		hostname = entry.Host
	}

	return bastionSummary{
		Name:        entry.Name,
		Environment: entry.Environment,
		Region:      entry.Region,
		Hostname:    hostname,
		Port:        entry.Port,
		Description: entry.Description,
		Tags:        entry.Tags,
	}
}

type bastionListOutput struct {
	Bastions         []bastionSummary  `json:"bastions"`
	Count            int               `json:"count"`
	ConfigSuggestion *configSuggestion `json:"configSuggestion"`
	ConfigPath       string            `json:"-"`
}

type bastionSummary struct {
	Name        string            `json:"name"`
	Environment string            `json:"environment,omitempty"`
	Region      string            `json:"region,omitempty"`
	Hostname    string            `json:"hostname,omitempty"`
	Port        int               `json:"port,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

type configSuggestion struct {
	Message    string `json:"message"`
	ConfigPath string `json:"configPath,omitempty"`
}

type errorOutput struct {
	Error string `json:"error"`
}
