package main

import (
	"github.com/smira/aptly/cmd"
	"os"
)

func main() {
	cmd.Run(os.Args[1:], true)
}
