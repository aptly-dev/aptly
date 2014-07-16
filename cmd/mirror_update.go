package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyMirrorUpdate(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]

	repo, err := context.CollectionFactory().RemoteRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	ignoreMismatch := context.flags.Lookup("ignore-checksums").Value.Get().(bool)

	verifier, err := getVerifier(context.flags)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.Downloader(), verifier)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	var filterQuery deb.PackageQuery

	if repo.Filter != "" {
		filterQuery, err = query.Parse(repo.Filter)
		if err != nil {
			return fmt.Errorf("unable to update: %s", err)
		}
	}

	err = repo.Download(context.Progress(), context.Downloader(), context.CollectionFactory(), context.PackagePool(), ignoreMismatch,
		context.DependencyOptions(), filterQuery)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = context.CollectionFactory().RemoteRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	context.Progress().Printf("\nMirror `%s` has been successfully updated.\n", repo.Name)
	return err
}

func makeCmdMirrorUpdate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorUpdate,
		UsageLine: "update <name>",
		Short:     "update mirror",
		Long: `
Updates remote mirror (downloads package files and meta information). When mirror is created,
this command should be run for the first time to fetch mirror contents. This command can be
run multiple times to get updated repository contents. If interrupted, command can be safely restarted.

Example:

  $ aptly mirror update wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-update", flag.ExitOnError),
	}

	cmd.Flag.Bool("ignore-checksums", false, "ignore checksum mismatches while downloading package files and metadata")
	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Int64("download-limit", 0, "limit download speed (kbytes/sec)")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
