package main

import (
	"github.com/smira/aptly/cmd"
	"os"
)

func main() {
	os.Exit(cmd.Run(cmd.RootCommand(), os.Args[1:], true))
}
