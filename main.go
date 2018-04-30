package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/cmd"
)

// Version variable, filled in at link time
var Version string

func main() {
	if Version == "" {
		Version = "unknown"
	}

	aptly.Version = Version

	rand.Seed(time.Now().UnixNano())

	os.Exit(cmd.Run(cmd.RootCommand(), os.Args[1:], true))
}
