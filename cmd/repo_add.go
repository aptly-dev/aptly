package cmd

import (
	"fmt"
	"os"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoAdd(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]

	verifier := context.GetVerifier()

	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.LocalRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	err = collectionFactory.LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	context.Progress().Printf("Loading packages...\n")

	list, err := deb.NewPackageListFromRefList(repo.RefList(), collectionFactory.PackageCollection(), context.Progress())
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	forceReplace := context.Flags().Lookup("force-replace").Value.Get().(bool)

	var packageFiles, otherFiles, failedFiles []string

	packageFiles, otherFiles, failedFiles = deb.CollectPackageFiles(args[1:], &aptly.ConsoleResultReporter{Progress: context.Progress()})

	var processedFiles, failedFiles2 []string

	processedFiles, failedFiles2, err = deb.ImportPackageFiles(list, packageFiles, forceReplace, verifier, context.PackagePool(),
		collectionFactory.PackageCollection(), &aptly.ConsoleResultReporter{Progress: context.Progress()}, nil,
		collectionFactory.ChecksumCollection)
	failedFiles = append(failedFiles, failedFiles2...)
	if err != nil {
		return fmt.Errorf("unable to import package files: %s", err)
	}

	processedFiles = append(processedFiles, otherFiles...)

	repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

	err = collectionFactory.LocalRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to save: %s", err)
	}

	if context.Flags().Lookup("remove-files").Value.Get().(bool) {
		processedFiles = utils.StrSliceDeduplicate(processedFiles)

		for _, file := range processedFiles {
			err = os.Remove(file)
			if err != nil {
				return fmt.Errorf("unable to remove file: %s", err)
			}
		}
	}

	if len(failedFiles) > 0 {
		context.Progress().ColoredPrintf("@y[!]@| @!Some files were skipped due to errors:@|")
		for _, file := range failedFiles {
			context.Progress().ColoredPrintf("  %s", file)
		}

		return fmt.Errorf("some files failed to be added")
	}

	return err
}

func makeCmdRepoAdd() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoAdd,
		UsageLine: "add <name> <package file.deb>|<directory> ...",
		Short:     "add packages to local repository",
		Long: `
Command adds packages to local repository from .deb, .udeb (binary packages) and .dsc (source packages) files.
When importing from directory aptly would do recursive scan looking for all files matching *.[u]deb or *.dsc
patterns. Every file discovered would be analyzed to extract metadata, package would then be created and added
to the database. Files would be imported to internal package pool. For source packages, all required files are
added automatically as well. Extra files for source package should be in the same directory as *.dsc file.

Example:

  $ aptly repo add testing myapp-0.1.2.deb incoming/
`,
		Flag: *flag.NewFlagSet("aptly-repo-add", flag.ExitOnError),
	}

	cmd.Flag.Bool("remove-files", false, "remove files that have been imported successfully into repository")
	cmd.Flag.Bool("force-replace", false, "when adding package that conflicts with existing package, remove existing package")

	return cmd
}
