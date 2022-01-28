package cmd

import (
	"fmt"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyMirrorEdit(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.RemoteRepoCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	err = repo.CheckLock()
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	fetchMirror := false
	context.Flags().Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "filter":
			repo.Filter = flag.Value.String()
		case "filter-with-deps":
			repo.FilterWithDeps = flag.Value.Get().(bool)
		case "with-installer":
			repo.DownloadInstaller = flag.Value.Get().(bool)
		case "with-sources":
			repo.DownloadSources = flag.Value.Get().(bool)
		case "with-udebs":
			repo.DownloadUdebs = flag.Value.Get().(bool)
		case "archive-url":
			repo.SetArchiveRoot(flag.Value.String())
			fetchMirror = true
		}
	})

	if repo.IsFlat() && repo.DownloadUdebs {
		return fmt.Errorf("unable to edit: flat mirrors don't support udebs")
	}

	if repo.Filter != "" {
		_, err = query.Parse(repo.Filter)
		if err != nil {
			return fmt.Errorf("unable to edit: %s", err)
		}
	}

	if context.GlobalFlags().Lookup("architectures").Value.String() != "" {
		repo.Architectures = context.ArchitecturesList()
		fetchMirror = true
	}

	if fetchMirror {
		var verifier pgp.Verifier
		verifier, err = getVerifier(context.Flags())
		if err != nil {
			return fmt.Errorf("unable to initialize GPG verifier: %s", err)
		}

		err = repo.Fetch(context.Downloader(), verifier)
		if err != nil {
			return fmt.Errorf("unable to edit: %s", err)
		}
	}

	err = collectionFactory.RemoteRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to edit: %s", err)
	}

	fmt.Printf("Mirror %s successfully updated.\n", repo)
	return err
}

func makeCmdMirrorEdit() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorEdit,
		UsageLine: "edit <name>",
		Short:     "edit mirror settings",
		Long: `
Command edit allows one to change settings of mirror:
filters, list of architectures.

Example:

  $ aptly mirror edit -filter=nginx -filter-with-deps some-mirror
`,
		Flag: *flag.NewFlagSet("aptly-mirror-edit", flag.ExitOnError),
	}

	cmd.Flag.String("archive-url", "", "archive url is the root of archive")
	cmd.Flag.String("filter", "", "filter packages in mirror")
	cmd.Flag.Bool("filter-with-deps", false, "when filtering, include dependencies of matching packages as well")
	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Bool("with-installer", false, "download additional not packaged installer files")
	cmd.Flag.Bool("with-sources", false, "download source packages in addition to binary packages")
	cmd.Flag.Bool("with-udebs", false, "download .udeb packages (Debian installer support)")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
