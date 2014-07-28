package cmd

import (
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func makeCmdRepoImport() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoMoveCopyImport,
		UsageLine: "import <src-mirror> <dst-repo> <package-query> ...",
		Short:     "import packages from mirror to local repository",
		Long: `
Command import looks up packages matching <package-query> in mirror <src-mirror>
and copies them to local repo <dst-repo>.

Example:

  $ aptly repo import wheezy-main testing nginx
`,
		Flag: *flag.NewFlagSet("aptly-repo-import", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't import, just show what would be imported")
	cmd.Flag.Bool("with-deps", false, "follow dependencies when processing package-spec")

	return cmd
}
