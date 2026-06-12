package cli

import (
	"bytes"
	"encoding/json"
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

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q, got %q", want, got)
	}
}
