package cmd

import (
	"fmt"
	"github.com/smira/aptly/debian"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"sort"
)

func aptlyMirrorList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	if context.collectionFactory.RemoteRepoCollection().Len() > 0 {
		fmt.Printf("List of mirrors:\n")
		repos := make([]string, context.collectionFactory.RemoteRepoCollection().Len())
		i := 0
		context.collectionFactory.RemoteRepoCollection().ForEach(func(repo *debian.RemoteRepo) error {
			repos[i] = repo.String()
			i++
			return nil
		})

		sort.Strings(repos)
		for _, repo := range repos {
			fmt.Printf(" * %s\n", repo)
		}

		fmt.Printf("\nTo get more information about mirror, run `aptly mirror show <name>`.\n")
	} else {
		fmt.Printf("No mirrors found, create one with `aptly mirror create ...`.\n")
	}
	return err
}

func makeCmdMirrorList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorList,
		UsageLine: "list",
		Short:     "list mirrors",
		Long: `
List shows full list of remote repository mirrors.

Example:

  $ aptly mirror list
`,
		Flag: *flag.NewFlagSet("aptly-mirror-list", flag.ExitOnError),
	}

	return cmd
}
