package cmd

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func makeCmdRepoImport() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoMoveCopyImport,
		UsageLine: "import <src-mirror> <dst-repo> <package-spec> ...",
		Short:     "import package from mirror and put it into local repo",
		Long: `
Command import looks up packages matching <package-spec> in mirror <src-mirror>
and copies them to local repo <dst-repo>.

ex:
  $ aptly repo import wheezy-main testing nginx
`,
		Flag: *flag.NewFlagSet("aptly-repo-import", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't import, just show what would be imported")
	cmd.Flag.Bool("with-deps", false, "follow dependencies when processing package-spec")

	return cmd
}
