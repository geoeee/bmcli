package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "help command", args: []string{"help"}},
		{name: "short flag", args: []string{"-h"}},
		{name: "long flag", args: []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d", code)
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected empty stderr, got %q", stderr.String())
			}
			assertContains(t, stdout.String(), "Usage:")
			assertContains(t, stdout.String(), "list      List configured bastions")
			assertContains(t, stdout.String(), "version   Print the bmcli version")
		})
	}
}

func TestRunVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if got, want := strings.TrimSpace(stdout.String()), "bmcli dev"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRunVersionCommandJSON(t *testing.T) {
	previousVersion, previousCommit, previousDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = previousVersion, previousCommit, previousDate
	})

	Version = "1.2.3"
	Commit = "abc123"
	Date = "2026-06-12"

	tests := []struct {
		name string
		args []string
	}{
		{name: "command flag", args: []string{"version", "--json"}},
		{name: "global flag", args: []string{"--json", "version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d", code)
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected empty stderr, got %q", stderr.String())
			}

			var got struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Commit  string `json:"commit"`
				Date    string `json:"date"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("expected valid JSON, got error %v and output %q", err, stdout.String())
			}

			if got.Name != "bmcli" {
				t.Fatalf("expected name %q, got %q", "bmcli", got.Name)
			}
			if got.Version != "1.2.3" {
				t.Fatalf("expected version %q, got %q", "1.2.3", got.Version)
			}
			if got.Commit != "abc123" {
				t.Fatalf("expected commit %q, got %q", "abc123", got.Commit)
			}
			if got.Date != "2026-06-12" {
				t.Fatalf("expected date %q, got %q", "2026-06-12", got.Date)
			}
		})
	}
}

func TestVersionStringIncludesBuildMetadata(t *testing.T) {
	previousVersion, previousCommit, previousDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = previousVersion, previousCommit, previousDate
	})

	Version = "1.2.3"
	Commit = "abc123"
	Date = "2026-06-12"

	if got, want := versionString(), "1.2.3 (abc123, 2026-06-12)"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"missing"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	assertContains(t, stderr.String(), `bmcli: unknown command "missing"`)
	assertContains(t, stderr.String(), "Available Commands:")
}

func TestRunListCommand(t *testing.T) {
	configPath := writeConfig(t, `{
		"bastions": [
			{
				"name": "dev-us-east",
				"environment": "dev",
				"region": "us-east-1",
				"hostname": "bastion.dev.example",
				"port": 22,
				"privateKey": "secret-key"
			},
			{
				"name": "prod-us-east",
				"host": "bastion.prod.example"
			}
		]
	}`)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"--config", configPath, "list"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	output := stdout.String()
	assertContains(t, output, "NAME\tHOST\tPORT\tENV\tREGION")
	assertContains(t, output, "dev-us-east\tbastion.dev.example\t22\tdev\tus-east-1")
	assertContains(t, output, "prod-us-east\tbastion.prod.example\t\t\t")
	assertNotContains(t, output, "secret-key")
}

func TestRunListCommandJSON(t *testing.T) {
	configPath := writeConfig(t, `{
		"bastions": [
			{
				"name": "dev-us-east",
				"environment": "dev",
				"region": "us-east-1",
				"hostname": "bastion.dev.example",
				"port": 22,
				"privateKey": "secret-key"
			}
		]
	}`)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"list", "--json", "--config=" + configPath}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var got struct {
		Bastions []struct {
			Name        string `json:"name"`
			Environment string `json:"environment"`
			Region      string `json:"region"`
			Hostname    string `json:"hostname"`
			Port        int    `json:"port"`
		} `json:"bastions"`
		Count            int         `json:"count"`
		ConfigSuggestion interface{} `json:"configSuggestion"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON, got error %v and output %q", err, stdout.String())
	}
	if got.Count != 1 {
		t.Fatalf("expected count 1, got %d", got.Count)
	}
	if len(got.Bastions) != 1 {
		t.Fatalf("expected one bastion, got %d", len(got.Bastions))
	}
	if got.Bastions[0].Name != "dev-us-east" || got.Bastions[0].Hostname != "bastion.dev.example" {
		t.Fatalf("unexpected bastion output: %+v", got.Bastions[0])
	}
	if got.ConfigSuggestion != nil {
		t.Fatalf("expected nil configSuggestion, got %#v", got.ConfigSuggestion)
	}
	assertNotContains(t, stdout.String(), "secret-key")
}

func TestRunListCommandEmptyConfig(t *testing.T) {
	configPath := writeConfig(t, `{"bastions":[]}`)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"list", "--config", configPath}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	assertContains(t, stdout.String(), "No bastions configured.")
	assertContains(t, stdout.String(), "Create a config file at: "+configPath)
}

func TestRunListCommandEmptyConfigJSON(t *testing.T) {
	configPath := writeConfig(t, `{}`)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"--json", "--config", configPath, "list"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var got struct {
		Bastions         []json.RawMessage `json:"bastions"`
		Count            int               `json:"count"`
		ConfigSuggestion struct {
			Message    string `json:"message"`
			ConfigPath string `json:"configPath"`
		} `json:"configSuggestion"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON, got error %v and output %q", err, stdout.String())
	}
	if got.Count != 0 {
		t.Fatalf("expected count 0, got %d", got.Count)
	}
	if len(got.Bastions) != 0 {
		t.Fatalf("expected empty bastions, got %d", len(got.Bastions))
	}
	if got.ConfigSuggestion.Message != "No bastions configured." {
		t.Fatalf("expected empty config suggestion, got %+v", got.ConfigSuggestion)
	}
	if got.ConfigSuggestion.ConfigPath != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, got.ConfigSuggestion.ConfigPath)
	}
}

func TestRunListCommandMissingDefaultConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"list"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	assertContains(t, stdout.String(), "No bastions configured.")
	assertContains(t, stdout.String(), filepath.Join(configHome, appName, "config.json"))
}

func TestRunListCommandMissingExplicitConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.json")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"list", "--config", configPath}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	assertContains(t, stderr.String(), "failed to read config "+configPath)
}

func TestRunListCommandInvalidConfig(t *testing.T) {
	tests := []struct {
		name       string
		configBody string
		want       string
	}{
		{name: "malformed json", configBody: `{`, want: "invalid config"},
		{name: "missing name", configBody: `{"bastions":[{"hostname":"bastion.dev.example"}]}`, want: "bastions[0].name is required"},
		{name: "missing hostname", configBody: `{"bastions":[{"name":"dev"}]}`, want: "bastions[0].hostname is required"},
		{name: "invalid port", configBody: `{"bastions":[{"name":"dev","hostname":"bastion.dev.example","port":70000}]}`, want: "bastions[0].port must be between 0 and 65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := writeConfig(t, tt.configBody)
			var stdout, stderr bytes.Buffer

			code := Run([]string{"list", "--config", configPath}, &stdout, &stderr)

			if code != 1 {
				t.Fatalf("expected exit code 1, got %d", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("expected empty stdout, got %q", stdout.String())
			}
			assertContains(t, stderr.String(), tt.want)
		})
	}
}

func TestRunListCommandInvalidConfigJSON(t *testing.T) {
	configPath := writeConfig(t, `{"bastions":[{"name":"dev"}]}`)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"list", "--config", configPath, "--json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}

	var got struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON error, got error %v and output %q", err, stderr.String())
	}
	assertContains(t, got.Error, "bastions[0].hostname is required")
}

func TestRunRequiresConfigPath(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"list", "--config"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	assertContains(t, stderr.String(), "--config requires a path")
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(body), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	return configPath
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q, got %q", want, got)
	}
}

func assertNotContains(t *testing.T, got string, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("expected output not to contain %q, got %q", want, got)
	}
}
