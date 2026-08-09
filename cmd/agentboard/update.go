package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/ayan-de/agent-board/internal/cli"
	"github.com/ayan-de/agent-board/internal/selfupdate"
	"github.com/ayan-de/agent-board/internal/version"
)

func updateCommand() cli.Command {
	return cli.Command{
		Name:  "update",
		Short: "Update agentboard to the latest release",
		Run:   runUpdate,
	}
}

func runUpdate(out io.Writer, args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(out)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(out, "error locating current executable: %v\n", err)
		return 1
	}

	fetcher := selfupdate.NewHTTPFetcher(selfupdate.DefaultRepo)
	fmt.Fprintln(out, "checking for updates...")

	rel, needsUpdate, err := selfupdate.Check(fetcher, version.Version)
	if err != nil {
		fmt.Fprintf(out, "error checking latest version: %v\n", err)
		return 1
	}
	if !needsUpdate {
		fmt.Fprintf(out, "agentboard is already up to date (%s)\n", version.Version)
		return 0
	}

	fmt.Fprintf(out, "updating agentboard %s -> %s...\n", version.Version, rel.TagName)
	if err := selfupdate.Apply(fetcher, rel, execPath, runtime.GOOS, runtime.GOARCH); err != nil {
		fmt.Fprintf(out, "update failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "updated to %s\n", rel.TagName)
	return 0
}
