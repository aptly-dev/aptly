package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"sort"
)

func aptlyMirrorList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	repoCollection := debian.NewRemoteRepoCollection(context.database)

	if repoCollection.Len() > 0 {
		fmt.Printf("List of mirrors:\n")
		repos := make([]string, repoCollection.Len())
		i := 0
		repoCollection.ForEach(func(repo *debian.RemoteRepo) error {
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
		Short:     "list mirrors of remote repositories",
		Long: `
List shows full list of remote repositories.

ex:
  $ aptly mirror list
`,
		Flag: *flag.NewFlagSet("aptly-mirror-list", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose output")

	return cmd
}
