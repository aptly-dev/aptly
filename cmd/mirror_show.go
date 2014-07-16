package cmd

import (
	"fmt"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"strings"
)

func aptlyMirrorShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]

	repo, err := context.CollectionFactory().RemoteRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
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
	if repo.Filter != "" {
		fmt.Printf("Filter: %s\n", repo.Filter)
		filterWithDeps := "no"
		if repo.FilterWithDeps {
			filterWithDeps = "yes"
		}
		fmt.Printf("Filter With Deps: %s\n", filterWithDeps)
	}
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

	withPackages := context.flags.Lookup("with-packages").Value.Get().(bool)
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
		Short:     "show details about mirror",
		Long: `
Shows detailed information about the mirror.

Example:

  $ aptly mirror show wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-packages", false, "show detailed list of packages and versions stored in the mirror")

	return cmd
}
