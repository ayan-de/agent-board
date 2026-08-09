package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func newTestApp(defaultCalled *bool) *App {
	return &App{
		Name:    "agentboard",
		Version: "v1.2.3",
		Commands: []Command{
			{
				Name:  "update",
				Short: "Update agentboard to the latest release",
				Run: func(out io.Writer, args []string) int {
					fmt.Fprint(out, "updating\n")
					return 0
				},
			},
		},
		DefaultRun: func(out io.Writer) int {
			*defaultCalled = true
			fmt.Fprint(out, "tui\n")
			return 0
		},
	}
}

func TestExecute_NoArgsRunsDefault(t *testing.T) {
	var called bool
	app := newTestApp(&called)
	var out bytes.Buffer

	code := app.Execute(&out, nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !called {
		t.Fatal("DefaultRun was not called")
	}
	if out.String() != "tui\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestExecute_VersionFlag(t *testing.T) {
	tests := []string{"-v", "--version", "version"}
	for _, arg := range tests {
		t.Run(arg, func(t *testing.T) {
			var called bool
			app := newTestApp(&called)
			var out bytes.Buffer

			code := app.Execute(&out, []string{arg})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if called {
				t.Fatal("DefaultRun should not be called")
			}
			if !strings.Contains(out.String(), "v1.2.3") {
				t.Fatalf("output = %q, want it to contain version", out.String())
			}
		})
	}
}

func TestExecute_HelpFlag(t *testing.T) {
	tests := []string{"-h", "--help", "help"}
	for _, arg := range tests {
		t.Run(arg, func(t *testing.T) {
			var called bool
			app := newTestApp(&called)
			var out bytes.Buffer

			code := app.Execute(&out, []string{arg})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if !strings.Contains(out.String(), "update") {
				t.Fatalf("help output = %q, want it to list the update command", out.String())
			}
			if !strings.Contains(out.String(), "-v, --version") {
				t.Fatalf("help output = %q, want it to document -v/--version", out.String())
			}
		})
	}
}

func TestExecute_KnownSubcommand(t *testing.T) {
	var called bool
	app := newTestApp(&called)
	var out bytes.Buffer

	code := app.Execute(&out, []string{"update"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out.String() != "updating\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestExecute_SubcommandReceivesRemainingArgs(t *testing.T) {
	var called bool
	var gotArgs []string
	app := newTestApp(&called)
	app.Commands[0].Run = func(out io.Writer, args []string) int {
		gotArgs = args
		return 0
	}
	var out bytes.Buffer

	app.Execute(&out, []string{"update", "--check"})

	if len(gotArgs) != 1 || gotArgs[0] != "--check" {
		t.Fatalf("gotArgs = %v, want [--check]", gotArgs)
	}
}

func TestExecute_UnknownSubcommand(t *testing.T) {
	var called bool
	app := newTestApp(&called)
	var out bytes.Buffer

	code := app.Execute(&out, []string{"bogus"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "bogus") {
		t.Fatalf("output = %q, want it to mention the unknown command", out.String())
	}
}
