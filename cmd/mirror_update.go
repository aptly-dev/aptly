package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"os"
	"os/signal"
	"strings"
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

	force := context.Flags().Lookup("force").Value.Get().(bool)
	if !force {
		err = repo.CheckLock()
		if err != nil {
			return fmt.Errorf("unable to update: %s", err)
		}
	}

	ignoreMismatch := context.Flags().Lookup("ignore-checksums").Value.Get().(bool)

	verifier, err := getVerifier(context.Flags())
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.Downloader(), verifier)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	context.Progress().Printf("Downloading & parsing package files...\n")
	err = repo.DownloadPackageIndexes(context.Progress(), context.Downloader(), context.CollectionFactory(), ignoreMismatch)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	if repo.Filter != "" {
		context.Progress().Printf("Applying filter...\n")
		var filterQuery deb.PackageQuery

		filterQuery, err = query.Parse(repo.Filter)
		if err != nil {
			return fmt.Errorf("unable to update: %s", err)
		}

		var oldLen, newLen int
		oldLen, newLen, err = repo.ApplyFilter(context.DependencyOptions(), filterQuery)
		if err != nil {
			return fmt.Errorf("unable to update: %s", err)
		}
		context.Progress().Printf("Packages filtered: %d -> %d.\n", oldLen, newLen)
	}

	var (
		downloadSize int64
		queue        []deb.PackageDownloadTask
	)

	skip_existing_packages := context.Flags().Lookup("skip-existing-packages").Value.Get().(bool)

	context.Progress().Printf("Building download queue...\n")
	queue, downloadSize, err = repo.BuildDownloadQueue(context.PackagePool(), skip_existing_packages)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	defer func() {
		// on any interruption, unlock the mirror
		err := context.ReOpenDatabase()
		if err == nil {
			repo.MarkAsIdle()
			context.CollectionFactory().RemoteRepoCollection().Update(repo)
		}
	}()

	repo.MarkAsUpdating()
	err = context.CollectionFactory().RemoteRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = context.CloseDatabase()
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	// Catch ^C
	sigch := make(chan os.Signal)
	signal.Notify(sigch, os.Interrupt)

	count := len(queue)
	context.Progress().Printf("Download queue: %d items (%s)\n", count, utils.HumanBytes(downloadSize))

	// Download from the queue
	context.Progress().InitBar(downloadSize, true)

	// Download all package files
	ch := make(chan error, count)

	// In separate goroutine (to avoid blocking main), push queue to downloader
	go func() {
		for _, task := range queue {
			context.Downloader().DownloadWithChecksum(repo.PackageURL(task.RepoURI).String(), task.DestinationPath, ch, task.Checksums, ignoreMismatch)
		}

		// We don't need queue after this point
		queue = nil
	}()

	// Wait for all downloads to finish
	errors := make([]string, 0)

	for count > 0 {
		select {
		case <-sigch:
			signal.Stop(sigch)
			return fmt.Errorf("unable to update: interrupted")
		case err = <-ch:
			if err != nil {
				errors = append(errors, err.Error())
			}
			count--
		}
	}

	context.Progress().ShutdownBar()
	signal.Stop(sigch)

	if len(errors) > 0 {
		return fmt.Errorf("unable to update: download errors:\n  %s\n", strings.Join(errors, "\n  "))
	}

	err = context.ReOpenDatabase()
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	repo.FinalizeDownload()
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

	cmd.Flag.Bool("force", false, "force update mirror even if it is locked by another process")
	cmd.Flag.Bool("ignore-checksums", false, "ignore checksum mismatches while downloading package files and metadata")
	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Bool("skip-existing-packages", false, "do not check file existance for packages listed in the internal database of the mirror")
	cmd.Flag.Int64("download-limit", 0, "limit download speed (kbytes/sec)")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
