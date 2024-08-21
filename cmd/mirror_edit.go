package cmd

import (
	"fmt"
        "bufio"
        "io/ioutil"
        "os"
        "strings"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func getContent(filterarg string) (string, error) {
	var err error
        // Check if filterarg starts with '@'
        if strings.HasPrefix(filterarg, "@") {
                // Remove the '@' character from filterarg
                filterarg = strings.TrimPrefix(filterarg, "@")
                if filterarg == "-" {
                	// If filterarg is "-", read from stdin
                        scanner := bufio.NewScanner(os.Stdin)
						scanner.Split(bufio.ScanLines)
						scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
                        var content strings.Builder
                        for scanner.Scan() {
                                content.WriteString(scanner.Text() + "\n")
                        }
                        err = scanner.Err()
			if err == nil {
                                filterarg = content.String()
                        }
                } else {
                	// Read the file content into a byte slice
			var data []byte
                	data, err = ioutil.ReadFile(filterarg)
                	if err == nil {
                        	filterarg = string(data)
                	}
		}
        }
        return filterarg, err
}

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
	filter := false
	ignoreSignatures := context.Config().GpgDisableVerify
	context.Flags().Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "filter":
			repo.Filter, err = getContent(flag.Value.String())
			filter = true
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
		case "ignore-signatures":
			ignoreSignatures = true
		}
	})

	if filter && err != nil {
		return fmt.Errorf("unable to read package query from file %s: %w", repo.Filter, err)
	}

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

		err = repo.Fetch(context.Downloader(), verifier, ignoreSignatures)
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
