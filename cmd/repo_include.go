package cmd

import (
	"fmt"
	"text/template"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
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
	repoTemplateString := context.Flags().Lookup("repo").Value.Get().(string)
	collectionFactory := context.NewCollectionFactory()

	var repoTemplate *template.Template
	repoTemplate, err = template.New("repo").Parse(repoTemplateString)
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

	var changesFiles, failedFiles, failedFiles2 []string

	changesFiles, failedFiles = deb.CollectChangesFiles(args, reporter)
	_, failedFiles2, err = deb.ImportChangesFiles(
		changesFiles, reporter, acceptUnsigned, ignoreSignatures, forceReplace, noRemoveFiles, verifier, repoTemplate,
		context.Progress(), collectionFactory.LocalRepoCollection(), collectionFactory.PackageCollection(),
		context.PackagePool(), collectionFactory.ChecksumCollection,
		uploaders, query.Parse)
	failedFiles = append(failedFiles, failedFiles2...)

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
