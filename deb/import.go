package deb

import (
	"github.com/smira/aptly/aptly"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CollectPackageFiles walks filesystem collecting all candidates for package files
func CollectPackageFiles(locations []string, reporter aptly.ResultReporter) (packageFiles, failedFiles []string, err error) {
	for _, location := range locations {
		info, err2 := os.Stat(location)
		if err2 != nil {
			reporter.Warning("Unable to process %s: %s", location, err2)
			failedFiles = append(failedFiles, location)
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

				if strings.HasSuffix(info.Name(), ".deb") || strings.HasSuffix(info.Name(), ".udeb") ||
					strings.HasSuffix(info.Name(), ".dsc") {
					packageFiles = append(packageFiles, path)
				}

				return nil
			})
		} else {
			if strings.HasSuffix(info.Name(), ".deb") || strings.HasSuffix(info.Name(), ".udeb") ||
				strings.HasSuffix(info.Name(), ".dsc") {
				packageFiles = append(packageFiles, location)
			} else {
				reporter.Warning("Unknown file extenstion: %s", location)
				failedFiles = append(failedFiles, location)
				continue
			}
		}
	}

	sort.Strings(packageFiles)

	return
}
