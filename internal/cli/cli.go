// Package cli provides a minimal, dependency-free command dispatcher for
// the agentboard binary: global -h/--help and -v/--version flags, a
// registry of named subcommands, and a default action when none is given.
package cli

import (
	"fmt"
	"io"
)

// Command is a named subcommand. Run receives any args after the command
// name and returns a process exit code.
type Command struct {
	Name  string
	Short string
	Run   func(out io.Writer, args []string) int
}

// App is the root command dispatcher.
type App struct {
	Name       string
	Version    string
	Commands   []Command
	DefaultRun func(out io.Writer) int
}

// Execute dispatches args (os.Args[1:]) to the matching command and
// returns the process exit code.
func (a *App) Execute(out io.Writer, args []string) int {
	if len(args) == 0 {
		return a.DefaultRun(out)
	}

	switch args[0] {
	case "-h", "--help", "help":
		a.printUsage(out)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintf(out, "%s %s\n", a.Name, a.Version)
		return 0
	}

	for _, c := range a.Commands {
		if c.Name == args[0] {
			return c.Run(out, args[1:])
		}
	}

	fmt.Fprintf(out, "unknown command %q\n\n", args[0])
	a.printUsage(out)
	return 1
}

func (a *App) printUsage(out io.Writer) {
	fmt.Fprintf(out, "%s %s\n\nUsage:\n  %s [command]\n\nCommands:\n", a.Name, a.Version, a.Name)
	for _, c := range a.Commands {
		fmt.Fprintf(out, "  %-10s %s\n", c.Name, c.Short)
	}
	fmt.Fprintf(out, "\nFlags:\n  -h, --help      show help\n  -v, --version   show version\n")
}
