// Package cmd implements console commands
package cmd

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"os"
	"time"
)

// ListPackagesRefList shows list of packages in PackageRefList
func ListPackagesRefList(reflist *deb.PackageRefList) (err error) {
	fmt.Printf("Packages:\n")

	if reflist == nil {
		return
	}

	err = reflist.ForEach(func(key []byte) error {
		p, err2 := context.CollectionFactory().PackageCollection().ByKey(key)
		if err2 != nil {
			return err2
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
back as Debian repositories.

aptly's goal is to establish repeatability and controlled changes
in a package-centric environment. aptly allows to fix a set of packages
in a repository, so that package installation and upgrade becomes
deterministic. At the same time aptly allows to perform controlled,
fine-grained changes in repository contents to transition your
package environment to new version.`,
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
