package cmd

import (
	"fmt"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func aptlyRepoAdd(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 2 {
		cmd.Usage()
		return err
	}

	name := args[0]

	verifier := &utils.GpgVerifier{}

	repo, err := context.collectionFactory.LocalRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	err = context.collectionFactory.LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	context.progress.Printf("Loading packages...\n")

	list, err := debian.NewPackageListFromRefList(repo.RefList(), context.collectionFactory.PackageCollection(), context.progress)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	packageFiles := []string{}

	for _, location := range args[1:] {
		info, err2 := os.Stat(location)
		if err2 != nil {
			context.progress.ColoredPrintf("@y[!]@| @!Unable to process %s: %s@|", location, err2)
			continue
		}
		if info.IsDir() {
			err2 = filepath.Walk(location, func(path string, info os.FileInfo, err3 error) error {
				if err3 != nil {
					return err3
				}
				if info.IsDir() {
					return nil
				}

				if strings.HasSuffix(info.Name(), ".deb") || strings.HasSuffix(info.Name(), ".dsc") {
					packageFiles = append(packageFiles, path)
				}

				return nil
			})
		} else {
			if strings.HasSuffix(info.Name(), ".deb") || strings.HasSuffix(info.Name(), ".dsc") {
				packageFiles = append(packageFiles, location)
			} else {
				context.progress.ColoredPrintf("@y[!]@| @!Unknwon file extenstion: %s@|", location)
				continue
			}
		}
	}

	processedFiles := []string{}
	sort.Strings(packageFiles)

	for _, file := range packageFiles {
		var (
			stanza debian.Stanza
			p      *debian.Package
		)

		candidateProcessedFiles := []string{}
		isSourcePackage := strings.HasSuffix(file, ".dsc")

		if isSourcePackage {
			stanza, err = debian.GetControlFileFromDsc(file, verifier)

			if err == nil {
				stanza["Package"] = stanza["Source"]
				delete(stanza, "Source")

				p, err = debian.NewSourcePackageFromControlFile(stanza)
			}
		} else {
			stanza, err = debian.GetControlFileFromDeb(file)
			p = debian.NewPackageFromControlFile(stanza)
		}
		if err != nil {
			context.progress.ColoredPrintf("@y[!]@| @!Unable to read file %s: %s@|", file, err)
			continue
		}

		var checksums utils.ChecksumInfo
		checksums, err = utils.ChecksumsForFile(file)
		if err != nil {
			return err
		}

		if isSourcePackage {
			p.UpdateFiles(append(p.Files(), debian.PackageFile{Filename: filepath.Base(file), Checksums: checksums}))
		} else {
			p.UpdateFiles([]debian.PackageFile{debian.PackageFile{Filename: filepath.Base(file), Checksums: checksums}})
		}

		err = context.packagePool.Import(file, checksums.MD5)
		if err != nil {
			context.progress.ColoredPrintf("@y[!]@| @!Unable to import file %s into pool: %s@|", file, err)
			continue
		}

		candidateProcessedFiles = append(candidateProcessedFiles, file)

		// go over all files, except for the last one (.dsc/.deb itself)
		for _, f := range p.Files() {
			if filepath.Base(f.Filename) == filepath.Base(file) {
				continue
			}
			sourceFile := filepath.Join(filepath.Dir(file), filepath.Base(f.Filename))
			err = context.packagePool.Import(sourceFile, f.Checksums.MD5)
			if err != nil {
				context.progress.ColoredPrintf("@y[!]@| @!Unable to import file %s into pool: %s@|", sourceFile, err)
				break
			}

			candidateProcessedFiles = append(candidateProcessedFiles, sourceFile)
		}
		if err != nil {
			// some files haven't been imported
			continue
		}

		err = context.collectionFactory.PackageCollection().Update(p)
		if err != nil {
			context.progress.ColoredPrintf("@y[!]@| @!Unable to save package %s: %s@|", p, err)
			continue
		}

		err = list.Add(p)
		if err != nil {
			context.progress.ColoredPrintf("@y[!]@| @!Unable to add package to repo %s: %s@|", p, err)
			continue
		}

		context.progress.ColoredPrintf("@g[+]@| %s added@|", p)
		processedFiles = append(processedFiles, candidateProcessedFiles...)
	}

	repo.UpdateRefList(debian.NewPackageRefListFromPackageList(list))

	err = context.collectionFactory.LocalRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to save: %s", err)
	}

	if context.flags.Lookup("remove-files").Value.Get().(bool) {
		processedFiles = utils.StrSliceDeduplicate(processedFiles)

		for _, file := range processedFiles {
			err := os.Remove(file)
			if err != nil {
				return fmt.Errorf("unable to remove file: %s", err)
			}
		}
	}

	return err
}

func makeCmdRepoAdd() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoAdd,
		UsageLine: "add <name> <package file.deb>|<directory> ...",
		Short:     "add packages to local repository",
		Long: `
Command adds packages to local repository from .deb (binary packages) and .dsc (source packages) files.
When importing from directory aptly would do recursive scan looking for all files matching *.deb or *.dsc
patterns. Every file discovered would be analyzed to extract metadata, package would be created and added
to database. Files would be imported to internal package pool. For source packages, all required files are
added as well automatically. Extra files for source package should be in the same directory as *.dsc file.

Example:

  $ aptly repo add testing myapp-0.1.2.deb incoming/
`,
		Flag: *flag.NewFlagSet("aptly-repo-add", flag.ExitOnError),
	}

	cmd.Flag.Bool("remove-files", false, "remove files that have been imported successfully into repository")

	return cmd
}
