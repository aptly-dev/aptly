package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func printReferencesTo(p *deb.Package) (err error) {
	err = context.CollectionFactory().RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		e := context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
		if e != nil {
			return e
		}
		if repo.RefList() != nil {
			if repo.RefList().Has(p) {
				fmt.Printf("  mirror %s\n", repo)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = context.CollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		e := context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
		if e != nil {
			return e
		}
		if repo.RefList() != nil {
			if repo.RefList().Has(p) {
				fmt.Printf("  local repo %s\n", repo)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = context.CollectionFactory().SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
		e := context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
		if e != nil {
			return e
		}
		if snapshot.RefList().Has(p) {
			fmt.Printf("  snapshot %s\n", snapshot)
		}
		return nil
	})

	return err
}

func aptlyPackageShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	q, err := query.Parse(args[0])
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	withFiles := context.Flags().Lookup("with-files").Value.Get().(bool)
	withReferences := context.Flags().Lookup("with-references").Value.Get().(bool)

	w := bufio.NewWriter(os.Stdout)

	result := q.Query(context.CollectionFactory().PackageCollection())

	err = result.ForEach(func(p *deb.Package) error {
		p.Stanza().WriteTo(w, p.IsSource, false)
		w.Flush()
		fmt.Printf("\n")

		if withFiles {
			fmt.Printf("Files in the pool:\n")
			packagePool := context.PackagePool()
			for _, f := range p.Files() {
				var path string
				path, err = f.GetPoolPath(packagePool)
				if err != nil {
					return err
				}

				if pp, ok := packagePool.(aptly.LocalPackagePool); ok {
					path = pp.FullPath(path)
				}

				fmt.Printf("  %s\n", path)
			}
			fmt.Printf("\n")
		}

		if withReferences {
			fmt.Printf("References to package:\n")
			printReferencesTo(p)
			fmt.Printf("\n")
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	return err
}

func makeCmdPackageShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPackageShow,
		UsageLine: "show <package-query>",
		Short:     "show details about packages matching query",
		Long: `
Command shows displays detailed meta-information about packages
matching query. Information from Debian control file is displayed.
Optionally information about package files and
inclusion into mirrors/snapshots/local repos is shown.

Example:

    $ aptly package show 'nginx-light_1.2.1-2.2+wheezy2_i386'
`,
		Flag: *flag.NewFlagSet("aptly-package-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("with-files", false, "display information about files from package pool")
	cmd.Flag.Bool("with-references", false, "display information about mirrors, snapshots and local repos referencing this package")

	return cmd
}
