//go:build testruncli
// +build testruncli

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/cmd"
)

func filterOutTestArgs(args []string) (out []string) {
	for _, arg := range args {
		if !strings.Contains(arg, "-test.coverprofile") {
			out = append(out, arg)
		}
	}
	return
}

// redefine all the flags otherwise the go testing tool
// is not able to parse them ...
var _ = flag.Int("db-open-attempts", 10, "number of attempts to open DB if it's locked by other instance")
var _ = flag.Bool("dep-follow-suggests", false, "when processing dependencies, follow Suggests")
var _ = flag.Bool("dep-follow-source", false, "when processing dependencies, follow from binary to Source packages")
var _ = flag.Bool("dep-follow-recommends", false, "when processing dependencies, follow Recommends")
var _ = flag.Bool("dep-follow-all-variants", false, "when processing dependencies, follow a & b if dependency is 'a|b'")
var _ = flag.Bool("dep-verbose-resolve", false, "when processing dependencies, print detailed logs")
var _ = flag.String("architectures", "", "list of architectures to consider during (comma-separated), default to all available")
var _ = flag.String("config", "", "location of configuration file (default locations are /etc/aptly.conf, ~/.aptly.conf)")
var _ = flag.String("gpg-provider", "", "PGP implementation (\"gpg\", \"gpg1\", \"gpg2\" for external gpg or \"internal\" for Go internal implementation)")

var _ = flag.String("cpuprofile", "", "write cpu profile to file")
var _ = flag.String("memprofile", "", "write memory profile to this file")
var _ = flag.String("memstats", "", "write memory stats periodically to this file")
var _ = flag.Duration("meminterval", 100*time.Millisecond, "memory stats dump interval")

var _ = flag.Bool("raw", false, "raw")
var _ = flag.String("sort", "false", "sort")
var _ = flag.Bool("json", false, "json")

func TestRunMain(t *testing.T) {
	if Version == "" {
		Version = "unknown"
	}

	aptly.Version = Version

	args := filterOutTestArgs(os.Args[1:])
	root := cmd.RootCommand()
	root.UsageLine = "aptly"

	fmt.Printf("EXIT: %d\n", cmd.Run(root, args, true))
}
