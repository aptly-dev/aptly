package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
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

	localRepoCollection := debian.NewLocalRepoCollection(context.database)
	repo, err := localRepoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	err = localRepoCollection.LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to add: %s", err)
	}

	context.progress.Printf("Loading packages...\n")

	packageCollection := debian.NewPackageCollection(context.database)
	list, err := debian.NewPackageListFromRefList(repo.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	packageFiles := []string{}

	for _, location := range args[1:] {
		info, err := os.Stat(location)
		if err != nil {
			context.progress.ColoredPrintf("@y[!]@| @!Unable to process %s: %s@|", location, err)
			continue
		}
		if info.IsDir() {
			err = filepath.Walk(location, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
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
			err    error
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

		checksums, err := utils.ChecksumsForFile(file)
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

		err = packageCollection.Update(p)
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

	err = localRepoCollection.Update(repo)
	if err != nil {
		return fmt.Errorf("unable to save: %s", err)
	}

	if cmd.Flag.Lookup("remove-files").Value.Get().(bool) {
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
Command adds packages to local repository. List of files or directories to be
scanned could be specified. If importing from directory, all files matching *.deb or *.dsc
patterns would be scanned and added to the repository. For source packages, all required
files are added as well automatically.

ex:
  $ aptly repo add testing myapp-0.1.2.deb incoming/
`,
		Flag: *flag.NewFlagSet("aptly-repo-add", flag.ExitOnError),
	}

	cmd.Flag.Bool("remove-files", false, "remove files that have been imported successfully into repository")

	return cmd
}
