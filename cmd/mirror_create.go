package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"strings"
)

func aptlyMirrorCreate(cmd *commander.Command, args []string) error {
	var err error
	if !(len(args) == 2 && strings.HasPrefix(args[1], "ppa:") || len(args) >= 3) {
		cmd.Usage()
		return commander.ErrCommandError
	}

	downloadSources := context.Config().DownloadSourcePackages || context.flags.Lookup("with-sources").Value.Get().(bool)

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

	repo, err := deb.NewRemoteRepo(mirrorName, archiveURL, distribution, components, context.ArchitecturesList(), downloadSources)
	if err != nil {
		return fmt.Errorf("unable to create mirror: %s", err)
	}

	repo.Filter = context.flags.Lookup("filter").Value.String()
	repo.FilterWithDeps = context.flags.Lookup("filter-with-deps").Value.Get().(bool)

	if repo.Filter != "" {
		_, err = query.Parse(repo.Filter)
		if err != nil {
			return fmt.Errorf("unable to create mirror: %s", err)
		}
	}

	verifier, err := getVerifier(context.flags)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.Downloader(), verifier)
	if err != nil {
		return fmt.Errorf("unable to fetch mirror: %s", err)
	}

	err = context.CollectionFactory().RemoteRepoCollection().Add(repo)
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
via HTTP. aptly would try download Release file from remote repository and verify its' signature. Command
line format resembles apt utlitily sources.list(5).

PPA urls could specified in short format:

  $ aptly mirror create <name> ppa:<user>/<project>

Example:

  $ aptly mirror create wheezy-main http://mirror.yandex.ru/debian/ wheezy main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-create", flag.ExitOnError),
	}

	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Bool("with-sources", false, "download source packages in addition to binary packages")
	cmd.Flag.String("filter", "", "filter packages in mirror")
	cmd.Flag.Bool("filter-with-deps", false, "when filtering, include dependencies of matching packages as well")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
