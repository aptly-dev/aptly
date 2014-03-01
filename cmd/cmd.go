// Package cmd implements console commands
package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/debian"
	"os"
	"time"
)

// ListPackagesRefList shows list of packages in PackageRefList
func ListPackagesRefList(reflist *debian.PackageRefList) (err error) {
	fmt.Printf("Packages:\n")

	if reflist == nil {
		return
	}

	packageCollection := debian.NewPackageCollection(context.database)

	err = reflist.ForEach(func(key []byte) error {
		p, err := packageCollection.ByKey(key)
		if err != nil {
			return err
		}
		fmt.Printf("  %s\n", p)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	return
}

// RootCommand creates root command in command tree
func RootCommand() *commander.Command {
	cmd := &commander.Command{
		UsageLine: os.Args[0],
		Short:     "Debian repository management tool",
		Long: `
aptly is a tool to create partial and full mirrors of remote
repositories, manage local repositories, filter them, merge,
upgrade individual packages, take snapshots and publish them
back as Debian repositories.`,
		Flag: *flag.NewFlagSet("aptly", flag.ExitOnError),
		Subcommands: []*commander.Command{
			makeCmdDb(),
			makeCmdGraph(),
			makeCmdMirror(),
			makeCmdRepo(),
			makeCmdServe(),
			makeCmdSnapshot(),
			makeCmdPublish(),
			makeCmdVersion(),
		},
	}

	cmd.Flag.Bool("dep-follow-suggests", false, "when processing dependencies, follow Suggests")
	cmd.Flag.Bool("dep-follow-source", false, "when processing dependencies, follow from binary to Source packages")
	cmd.Flag.Bool("dep-follow-recommends", false, "when processing dependencies, follow Recommends")
	cmd.Flag.Bool("dep-follow-all-variants", false, "when processing dependencies, follow a & b if depdency is 'a|b'")
	cmd.Flag.String("architectures", "", "list of architectures to consider during (comma-separated), default to all available")
	cmd.Flag.String("config", "", "location of configuration file (default locations are /etc/aptly.conf, ~/.aptly.conf)")

	if aptly.EnableDebug {
		cmd.Flag.String("cpuprofile", "", "write cpu profile to file")
		cmd.Flag.String("memprofile", "", "write memory profile to this file")
		cmd.Flag.String("memstats", "", "write memory stats periodically to this file")
		cmd.Flag.Duration("meminterval", 100*time.Millisecond, "memory stats dump interval")
	}
	return cmd
}
