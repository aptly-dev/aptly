package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoInclude(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	verifier, err := getVerifier(context.Flags())
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	if verifier == nil {
		verifier = context.GetVerifier()
	}

	forceReplace := context.Flags().Lookup("force-replace").Value.Get().(bool)
	acceptUnsigned := context.Flags().Lookup("accept-unsigned").Value.Get().(bool)
	ignoreSignatures := context.Flags().Lookup("ignore-signatures").Value.Get().(bool)
	noRemoveFiles := context.Flags().Lookup("no-remove-files").Value.Get().(bool)

	repoTemplate, err := template.New("repo").Parse(context.Flags().Lookup("repo").Value.Get().(string))
	if err != nil {
		return fmt.Errorf("error parsing -repo template: %s", err)
	}

	uploaders := (*deb.Uploaders)(nil)
	uploadersFile := context.Flags().Lookup("uploaders-file").Value.Get().(string)
	if uploadersFile != "" {
		uploaders, err = deb.NewUploadersFromFile(uploadersFile)
		if err != nil {
			return err
		}

		for i := range uploaders.Rules {
			uploaders.Rules[i].CompiledCondition, err = query.Parse(uploaders.Rules[i].Condition)
			if err != nil {
				return fmt.Errorf("error parsing query %s: %s", uploaders.Rules[i].Condition, err)
			}
		}
	}

	reporter := &aptly.ConsoleResultReporter{Progress: context.Progress()}

	var changesFiles, failedFiles, processedFiles []string

	changesFiles, failedFiles = deb.CollectChangesFiles(args, reporter)

	for _, path := range changesFiles {
		var changes *deb.Changes

		changes, err = deb.NewChanges(path)
		if err != nil {
			failedFiles = append(failedFiles, path)
			reporter.Warning("unable to process file %s: %s", path, err)
			continue
		}

		err = changes.VerifyAndParse(acceptUnsigned, ignoreSignatures, verifier)
		if err != nil {
			failedFiles = append(failedFiles, path)
			reporter.Warning("unable to process file %s: %s", changes.ChangesName, err)
			changes.Cleanup()
			continue
		}

		err = changes.Prepare()
		if err != nil {
			failedFiles = append(failedFiles, path)
			reporter.Warning("unable to process file %s: %s", changes.ChangesName, err)
			changes.Cleanup()
			continue
		}

		repoName := &bytes.Buffer{}
		err = repoTemplate.Execute(repoName, changes.Stanza)
		if err != nil {
			return fmt.Errorf("error applying template to repo: %s", err)
		}

		context.Progress().Printf("Loading repository %s for changes file %s...\n", repoName.String(), changes.ChangesName)

		var repo *deb.LocalRepo
		repo, err = context.CollectionFactory().LocalRepoCollection().ByName(repoName.String())
		if err != nil {
			failedFiles = append(failedFiles, path)
			reporter.Warning("unable to process file %s: %s", changes.ChangesName, err)
			changes.Cleanup()
			continue
		}

		currentUploaders := uploaders
		if repo.Uploaders != nil {
			currentUploaders = repo.Uploaders
			for i := range currentUploaders.Rules {
				currentUploaders.Rules[i].CompiledCondition, err = query.Parse(currentUploaders.Rules[i].Condition)
				if err != nil {
					return fmt.Errorf("error parsing query %s: %s", currentUploaders.Rules[i].Condition, err)
				}
			}
		}

		if currentUploaders != nil {
			if err = currentUploaders.IsAllowed(changes); err != nil {
				failedFiles = append(failedFiles, path)
				reporter.Warning("changes file skipped due to uploaders config: %s, keys %#v: %s",
					changes.ChangesName, changes.SignatureKeys, err)
				changes.Cleanup()
				continue
			}
		}

		err = context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
		if err != nil {
			return fmt.Errorf("unable to load repo: %s", err)
		}

		var list *deb.PackageList
		list, err = deb.NewPackageListFromRefList(repo.RefList(), context.CollectionFactory().PackageCollection(), context.Progress())
		if err != nil {
			return fmt.Errorf("unable to load packages: %s", err)
		}

		packageFiles, _ := deb.CollectPackageFiles([]string{changes.TempDir}, reporter)

		var restriction deb.PackageQuery

		restriction, err = changes.PackageQuery()
		if err != nil {
			failedFiles = append(failedFiles, path)
			reporter.Warning("unable to process file %s: %s", changes.ChangesName, err)
			changes.Cleanup()
			continue
		}

		var processedFiles2, failedFiles2 []string

		processedFiles2, failedFiles2, err = deb.ImportPackageFiles(list, packageFiles, forceReplace, verifier, context.PackagePool(),
			context.CollectionFactory().PackageCollection(), reporter, restriction, context.CollectionFactory().ChecksumCollection())

		if err != nil {
			return fmt.Errorf("unable to import package files: %s", err)
		}

		repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

		err = context.CollectionFactory().LocalRepoCollection().Update(repo)
		if err != nil {
			return fmt.Errorf("unable to save: %s", err)
		}

		err = changes.Cleanup()
		if err != nil {
			return err
		}

		for _, file := range failedFiles2 {
			failedFiles = append(failedFiles, filepath.Join(changes.BasePath, filepath.Base(file)))
		}

		for _, file := range processedFiles2 {
			processedFiles = append(processedFiles, filepath.Join(changes.BasePath, filepath.Base(file)))
		}

		processedFiles = append(processedFiles, path)
	}

	if !noRemoveFiles {
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

func makeCmdRepoInclude() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoInclude,
		UsageLine: "include <file.changes>|<directory> ...",
		Short:     "add packages to local repositories based on .changes files",
		Long: `
Command include looks for .changes files in list of arguments or specified directories. Each
.changes file is verified, parsed, referenced files are put into separate temporary directory
and added into local repository. Successfully imported files are removed by default.

Additionally uploads could be restricted with <uploaders.json> file. Rules in this file control
uploads based on GPG key ID of .changes file signature and queries on .changes file fields.

Example:

  $ aptly repo include -repo=foo-release incoming/
`,
		Flag: *flag.NewFlagSet("aptly-repo-include", flag.ExitOnError),
	}

	cmd.Flag.Bool("no-remove-files", false, "don't remove files that have been imported successfully into repository")
	cmd.Flag.Bool("force-replace", false, "when adding package that conflicts with existing package, remove existing package")
	cmd.Flag.String("repo", "{{.Distribution}}", "which repo should files go to, defaults to Distribution field of .changes file")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")
	cmd.Flag.Bool("ignore-signatures", false, "disable verification of .changes file signature")
	cmd.Flag.Bool("accept-unsigned", false, "accept unsigned .changes files")
	cmd.Flag.String("uploaders-file", "", "path to uploaders.json file")

	return cmd
}
