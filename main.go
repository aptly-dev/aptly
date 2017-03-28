package main

import (
	"os"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/cmd"
)

// Version variable, filled in at link time
var Version string

func main() {
	if Version == "" {
		Version = "unknown"
	}

	aptly.Version = Version

	os.Exit(cmd.Run(cmd.RootCommand(), os.Args[1:], true))
}
