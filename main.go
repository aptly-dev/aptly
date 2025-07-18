package main

import (
	"os"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/cmd"

	_ "embed"
)

//go:generate sh -c "make -s version | tr -d '\n' > VERSION"
//go:embed VERSION
var Version string

//go:embed debian/aptly.conf
var AptlyConf []byte

func main() {
	if Version == "" {
		Version = "unknown"
	}

	aptly.Version = Version
	aptly.AptlyConf = AptlyConf

	os.Exit(cmd.RunCommand(cmd.RootCommand(), os.Args[1:], true))
}
