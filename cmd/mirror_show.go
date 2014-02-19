package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"strings"
)

func aptlyMirrorShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repo, err := repoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = repoCollection.LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", repo.Name)
	fmt.Printf("Archive Root URL: %s\n", repo.ArchiveRoot)
	fmt.Printf("Distribution: %s\n", repo.Distribution)
	fmt.Printf("Components: %s\n", strings.Join(repo.Components, ", "))
	fmt.Printf("Architectures: %s\n", strings.Join(repo.Architectures, ", "))
	downloadSources := "no"
	if repo.DownloadSources {
		downloadSources = "yes"
	}
	fmt.Printf("Download Sources: %s\n", downloadSources)
	if repo.LastDownloadDate.IsZero() {
		fmt.Printf("Last update: never\n")
	} else {
		fmt.Printf("Last update: %s\n", repo.LastDownloadDate.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Number of packages: %d\n", repo.NumPackages())
	}

	fmt.Printf("\nInformation from release file:\n")
	for _, k := range utils.StrMapSortedKeys(repo.Meta) {
		fmt.Printf("%s: %s\n", k, repo.Meta[k])
	}

	withPackages := cmd.Flag.Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		if repo.LastDownloadDate.IsZero() {
			fmt.Printf("Unable to show package list, mirror hasn't been downloaded yet.\n")
		} else {
			ListPackagesRefList(repo.RefList())
		}
	}

	return err
}

func makeCmdMirrorShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorShow,
		UsageLine: "show <name>",
		Short:     "show details about remote repository mirror",
		Long: `
Show shows full information about mirror.

ex:
  $ aptly mirror show wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-packages", false, "show list of packages")

	return cmd
}
