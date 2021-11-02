package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyMirrorShow(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlyMirrorShowJson(cmd, args)
	}

	return aptlyMirrorShowTxt(cmd, args)
}

func aptlyMirrorShowTxt(cmd *commander.Command, args []string) error {
	var err error

	name := args[0]

	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.RemoteRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = collectionFactory.RemoteRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", repo.Name)
	if repo.Status == deb.MirrorUpdating {
		fmt.Printf("Status: In Update (PID %d)\n", repo.WorkerPID)
	}
	fmt.Printf("Archive Root URL: %s\n", repo.ArchiveRoot)
	fmt.Printf("Distribution: %s\n", repo.Distribution)
	fmt.Printf("Components: %s\n", strings.Join(repo.Components, ", "))
	fmt.Printf("Architectures: %s\n", strings.Join(repo.Architectures, ", "))
	downloadSources := No
	if repo.DownloadSources {
		downloadSources = Yes
	}
	fmt.Printf("Download Sources: %s\n", downloadSources)
	downloadUdebs := No
	if repo.DownloadUdebs {
		downloadUdebs = Yes
	}
	fmt.Printf("Download .udebs: %s\n", downloadUdebs)
	if repo.Filter != "" {
		fmt.Printf("Filter: %s\n", repo.Filter)
		filterWithDeps := No
		if repo.FilterWithDeps {
			filterWithDeps = Yes
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

	withPackages := context.Flags().Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		if repo.LastDownloadDate.IsZero() {
			fmt.Printf("Unable to show package list, mirror hasn't been downloaded yet.\n")
		} else {
			ListPackagesRefList(repo.RefList(), collectionFactory)
		}
	}

	return err
}

func aptlyMirrorShowJson(cmd *commander.Command, args []string) error {
	var err error

	name := args[0]

	repo, err := context.NewCollectionFactory().RemoteRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = context.NewCollectionFactory().RemoteRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	// include packages if requested
	withPackages := context.Flags().Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		if repo.RefList() != nil {
			var list *deb.PackageList
			list, err = deb.NewPackageListFromRefList(repo.RefList(), context.NewCollectionFactory().PackageCollection(), context.Progress())

			list.PrepareIndex()
			list.ForEachIndexed(func(p *deb.Package) error {
				repo.Packages = append(repo.Packages, p.GetFullName())
				return nil
			})

			sort.Strings(repo.Packages)
		}
	}

	var output []byte
	if output, err = json.MarshalIndent(repo, "", "  "); err == nil {
		fmt.Println(string(output))
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

	cmd.Flag.Bool("json", false, "display record in JSON format")
	cmd.Flag.Bool("with-packages", false, "show detailed list of packages and versions stored in the mirror")

	return cmd
}
