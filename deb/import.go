package deb

import (
	"cmp"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/utils"
	"github.com/saracen/walker"
)

// FileWithSize holds a file path and its size
type FileWithSize struct {
	Path string
	Size int64
}

// CollectPackageFiles walks filesystem collecting all candidates for package files
func CollectPackageFiles(locations []string, reporter aptly.ResultReporter) (packageFiles []FileWithSize, otherFiles, failedFiles []string) {
	packageFilesLock := &sync.Mutex{}
	otherFilesLock := &sync.Mutex{}

	for _, location := range locations {
		info, err2 := os.Stat(location)
		if err2 != nil {
			reporter.Warning("Unable to process %s: %s", location, err2)
			failedFiles = append(failedFiles, location)
			continue
		}
		if info.IsDir() {
			err2 = walker.Walk(location, func(path string, info os.FileInfo) error {
				if info.IsDir() {
					return nil
				}

				if strings.HasSuffix(info.Name(), ".deb") || strings.HasSuffix(info.Name(), ".udeb") ||
					strings.HasSuffix(info.Name(), ".dsc") || strings.HasSuffix(info.Name(), ".ddeb") {
					packageFilesLock.Lock()
					defer packageFilesLock.Unlock()
					packageFiles = append(packageFiles, FileWithSize{Path: path, Size: info.Size()})
				} else if strings.HasSuffix(info.Name(), ".buildinfo") {
					otherFilesLock.Lock()
					defer otherFilesLock.Unlock()
					otherFiles = append(otherFiles, path)
				}

				return nil
			})

			if err2 != nil {
				reporter.Warning("Unable to process %s: %s", location, err2)
				failedFiles = append(failedFiles, location)
				continue
			}
		} else {
			if strings.HasSuffix(info.Name(), ".deb") || strings.HasSuffix(info.Name(), ".udeb") ||
				strings.HasSuffix(info.Name(), ".dsc") || strings.HasSuffix(info.Name(), ".ddeb") {
				packageFiles = append(packageFiles, FileWithSize{Path: location, Size: info.Size()})
			} else if strings.HasSuffix(info.Name(), ".buildinfo") {
				otherFiles = append(otherFiles, location)
			} else {
				reporter.Warning("Unknown file extension: %s", location)
				failedFiles = append(failedFiles, location)
				continue
			}
		}
	}

	slices.SortFunc(packageFiles, func(a, b FileWithSize) int {
		return strings.Compare(a.Path, b.Path)
	})

	return
}

// ImportPackageFiles imports files into local repository
func ImportPackageFiles(list *PackageList, packageFiles []FileWithSize, forceReplace bool, verifier pgp.Verifier,
	pool aptly.PackagePool, collection *PackageCollection, reporter aptly.ResultReporter, progress aptly.Progress,
	restriction PackageQuery,
	checksumStorageProvider aptly.ChecksumStorageProvider) (processedFiles []string, failedFiles []string, err error) {
	if forceReplace {
		list.PrepareIndex()
	}

	checksumStorage := checksumStorageProvider(collection.db)

	// Sort larger files before small ones, as larger files take longer to hash
	// and thus result in a long non-parallelized "tail" if they are processed
	// closer to the end.
	fileSizeOrdering := make([]int, len(packageFiles))
	for i := range packageFiles {
		fileSizeOrdering[i] = i
	}

	slices.SortFunc(fileSizeOrdering, func(a, b int) int {
		return -cmp.Compare(packageFiles[a].Size, packageFiles[b].Size)
	})

	if progress != nil {
		progress.Printf("Calculating checksums of %d file(s)\n", len(packageFiles))
		progress.InitBar(int64(len(packageFiles)), false, aptly.BarGeneralImportFiles)
	}

	var nextOrdIdx atomic.Int64
	packageChecksums := make([]utils.ChecksumInfo, len(packageFiles))
	packageChecksumErrors := make([]error, len(packageFiles))

	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(packageFiles)) {
		wg.Go(func() {
			var buf [512 * 1024]byte

			for {
				ordIdx := nextOrdIdx.Add(1) - 1
				if ordIdx >= int64(len(packageFiles)) {
					return
				}

				i := fileSizeOrdering[ordIdx]
				fsz := packageFiles[i]
				checksum, err := utils.ChecksumsForFile(fsz.Path, buf[:])
				if err != nil {
					packageChecksumErrors[i] = err
				} else {
					packageChecksums[i] = checksum
				}

				if progress != nil {
					progress.AddBar(1)
				}
			}
		})
	}

	wg.Wait()

	if progress != nil {
		progress.ShutdownBar()
		progress.Printf("Importing %d file(s)\n", len(packageFiles))
	}

	for i, fsz := range packageFiles {
		var (
			stanza Stanza
			p      *Package
		)

		file := fsz.Path

		if err := packageChecksumErrors[i]; err != nil {
			// Computing the checksum already failed above.
			reporter.Warning("Unable to read file %s: %s", file, err)
			failedFiles = append(failedFiles, file)
			continue
		}

		checksums := packageChecksums[i]
		candidateProcessedFiles := []string{}
		isSourcePackage := strings.HasSuffix(file, ".dsc")
		isUdebPackage := strings.HasSuffix(file, ".udeb")

		if isSourcePackage {
			stanza, err = GetControlFileFromDsc(file, verifier)

			if err == nil {
				stanza["Package"] = stanza["Source"]
				delete(stanza, "Source")

				p, err = NewSourcePackageFromControlFile(stanza)
			}
		} else {
			stanza, err = GetControlFileFromDeb(file)
			if isUdebPackage {
				p = NewUdebPackageFromControlFile(stanza)
			} else {
				p = NewPackageFromControlFile(stanza)
			}
		}
		if err != nil {
			reporter.Warning("Unable to read file %s: %s", file, err)
			failedFiles = append(failedFiles, file)
			continue
		}

		if p.Name == "" {
			reporter.Warning("Empty package name on %s", file)
			failedFiles = append(failedFiles, file)
			continue
		}

		if p.Version == "" {
			reporter.Warning("Empty version on %s", file)
			failedFiles = append(failedFiles, file)
			continue
		}

		if !isValidVersion(p.Version) {
			reporter.Warning("Version number ('%s') for the '%s' package is invalid", p.Version, p.Name)
			failedFiles = append(failedFiles, file)
			continue
		}

		if p.Architecture == "" {
			reporter.Warning("Empty architecture on %s", file)
			failedFiles = append(failedFiles, file)
			continue
		}

		var files PackageFiles

		if isSourcePackage {
			files = p.Files()
		}

		mainPackageFile := PackageFile{
			Filename:  filepath.Base(file),
			Checksums: checksums,
		}

		mainPackageFile.PoolPath, err = pool.Import(file, mainPackageFile.Filename, &mainPackageFile.Checksums, false, checksumStorage)
		if err != nil {
			reporter.Warning("Unable to import file %s into pool: %s", file, err)
			failedFiles = append(failedFiles, file)
			continue
		}

		candidateProcessedFiles = append(candidateProcessedFiles, file)

		// go over all the other files
		for i := range files {
			sourceFile := filepath.Join(filepath.Dir(file), filepath.Base(files[i].Filename))

			_, err = os.Stat(sourceFile)
			if err == nil {
				files[i].PoolPath, err = pool.Import(sourceFile, files[i].Filename, &files[i].Checksums, false, checksumStorage)
				if err == nil {
					candidateProcessedFiles = append(candidateProcessedFiles, sourceFile)
				}
			} else if os.IsNotExist(err) {
				// if file is not present, try to find it in the pool
				var (
					err2  error
					found bool
				)

				files[i].PoolPath, found, err2 = pool.Verify("", files[i].Filename, &files[i].Checksums, checksumStorage)
				if err2 != nil {
					err = err2
				} else if found {
					// clear error, file is already in the package pool
					err = nil
				}
			}

			if err != nil {
				reporter.Warning("Unable to import file %s into pool: %s", sourceFile, err)
				failedFiles = append(failedFiles, file)
				break
			}
		}
		if err != nil {
			// some files haven't been imported
			continue
		}

		p.UpdateFiles(append(files, mainPackageFile))

		if restriction != nil && !restriction.Matches(p) {
			reporter.Warning("%s has been ignored as it doesn't match restriction", p)
			failedFiles = append(failedFiles, file)
			continue
		}

		err = collection.Update(p)
		if err != nil {
			reporter.Warning("Unable to save package %s: %s", p, err)
			failedFiles = append(failedFiles, file)
			continue
		}

		if forceReplace {
			conflictingPackages := list.Search(Dependency{Pkg: p.Name, Version: p.Version, Relation: VersionEqual, Architecture: p.Architecture}, true, false)
			for _, cp := range conflictingPackages {
				if cp.ArchitectureVariant != p.ArchitectureVariant {
					continue
				}
				reporter.Removed("%s removed due to conflict with package being added", cp)
				list.Remove(cp)
			}
		}

		err = list.Add(p)
		if err != nil {
			reporter.Warning("Unable to add package: %s", err)
			failedFiles = append(failedFiles, file)
			continue
		}

		reporter.Added("%s added", p)
		processedFiles = append(processedFiles, candidateProcessedFiles...)
	}

	err = nil // reset error as only failed files are reported
	return
}
