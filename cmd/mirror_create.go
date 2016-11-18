package cmd

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyMirrorCreate(cmd *commander.Command, args []string) error {
	var err error
	if !(len(args) == 2 && strings.HasPrefix(args[1], "ppa:") || len(args) >= 3) {
		cmd.Usage()
		return commander.ErrCommandError
	}

	downloadSources := LookupOption(context.Config().DownloadSourcePackages, context.Flags(), "with-sources")
	downloadUdebs := context.Flags().Lookup("with-udebs").Value.Get().(bool)
	downloadInstaller := context.Flags().Lookup("with-installer").Value.Get().(bool)

	var (
		mirrorName, archiveURL, distribution string
		components                           []string
	)

	mirrorName = args[0]
	if len(args) == 2 {
		archiveURL, distribution, components, err = deb.ParsePPA(args[1], context.Config())
		if err != nil {
			return err
		}
	} else {
		archiveURL, distribution, components = args[1], args[2], args[3:]
	}

	repo, err := deb.NewRemoteRepo(mirrorName, archiveURL, distribution, components, context.ArchitecturesList(),
		downloadSources, downloadUdebs, downloadInstaller)
	if err != nil {
		return fmt.Errorf("unable to create mirror: %s", err)
	}

	repo.Filter = context.Flags().Lookup("filter").Value.String()
	repo.FilterWithDeps = context.Flags().Lookup("filter-with-deps").Value.Get().(bool)
	repo.SkipComponentCheck = context.Flags().Lookup("force-components").Value.Get().(bool)
	repo.SkipArchitectureCheck = context.Flags().Lookup("force-architectures").Value.Get().(bool)

	if repo.Filter != "" {
		_, err = query.Parse(repo.Filter)
		if err != nil {
			return fmt.Errorf("unable to create mirror: %s", err)
		}
	}

	verifier, err := getVerifier(context.Flags())
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.Downloader(), verifier)
	if err != nil {
		return fmt.Errorf("unable to fetch mirror: %s", err)
	}

	collectionFactory := context.NewCollectionFactory()
	err = collectionFactory.RemoteRepoCollection().Add(repo)
	if err != nil {
		return fmt.Errorf("unable to add mirror: %s", err)
	}

	fmt.Printf("\nMirror %s successfully added.\nYou can run 'aptly mirror update %s' to download repository contents.\n", repo, repo.Name)
	return err
}

func makeCmdMirrorCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorCreate,
		UsageLine: "create <name> <archive url> <distribution> [<component1> ...]",
		Short:     "create new mirror",
		Long: `
Creates mirror <name> of remote repository, aptly supports both regular and flat Debian repositories exported
via HTTP and FTP. aptly would try download Release file from remote repository and verify its' signature. Command
line format resembles apt utlitily sources.list(5).

PPA urls could specified in short format:

  $ aptly mirror create <name> ppa:<user>/<project>

Example:

  $ aptly mirror create wheezy-main http://mirror.yandex.ru/debian/ wheezy main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-create", flag.ExitOnError),
	}

	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Bool("with-installer", false, "download additional not packaged installer files")
	cmd.Flag.Bool("with-sources", false, "download source packages in addition to binary packages")
	cmd.Flag.Bool("with-udebs", false, "download .udeb packages (Debian installer support)")
	cmd.Flag.String("filter", "", "filter packages in mirror")
	cmd.Flag.Bool("filter-with-deps", false, "when filtering, include dependencies of matching packages as well")
	cmd.Flag.Bool("force-components", false, "(only with component list) skip check that requested components are listed in Release file")
	cmd.Flag.Bool("force-architectures", false, "(only with architecture list) skip check that requested architectures are listed in Release file")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
