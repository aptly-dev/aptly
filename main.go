package main

import (
	"os"

	"github.com/smira/aptly/cmd"
)

func main() {
	os.Exit(cmd.Run(cmd.RootCommand(), os.Args[1:], true))
}
