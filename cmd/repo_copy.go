package cmd

import (
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func makeCmdRepoCopy() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoMoveCopyImport,
		UsageLine: "copy <src-name> <dst-name> <package-query> ...",
		Short:     "copy packages between local repositories",
		Long: `
Command copy copies packages matching <package-query> from local repo
<src-name> to local repo <dst-name>.

Example:

  $ aptly repo copy testing stable 'myapp (=0.1.12)'
`,
		Flag: *flag.NewFlagSet("aptly-repo-copy", flag.ExitOnError),
	}

	cmd.Flag.Bool("dry-run", false, "don't copy, just show what would be copied")
	cmd.Flag.Bool("with-deps", false, "follow dependencies when processing package-spec")

	return cmd
}
