// Package debian implements Debian-specific repository handling
package debian

import (
	"fmt"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/utils"
	debc "github.com/smira/godebiancontrol"
	"net/url"
	"strings"
)

// RemoteRepo represents remote (fetchable) Debian repository.
//
// Repostitory could be filtered when fetching by components, architectures
// TODO: support flat format
type RemoteRepo struct {
	ArchiveRoot    string
	Distribution   string
	Components     []string
	Architectures  []string
	archiveRootURL *url.URL
}

// NewRemoteRepo creates new instance of Debian remote repository with specified params
func NewRemoteRepo(archiveRoot string, distribution string, components []string, architectures []string) (*RemoteRepo, error) {
	result := &RemoteRepo{
		ArchiveRoot:   archiveRoot,
		Distribution:  distribution,
		Components:    components,
		Architectures: architectures,
	}

	var err error

	result.archiveRootURL, err = url.Parse(archiveRoot)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// String interface
func (repo *RemoteRepo) String() string {
	return fmt.Sprintf("%s %s", repo.ArchiveRoot, repo.Distribution)
}

// ReleaseURL returns URL to Release file in repo root
// TODO: InRelease, Release.gz, Release.bz2 handling
func (repo *RemoteRepo) ReleaseURL() *url.URL {
	path := &url.URL{Path: fmt.Sprintf("dists/%s/Release", repo.Distribution)}
	return repo.archiveRootURL.ResolveReference(path)
}

// BinaryURL returns URL of Packages file for given component and
// architecture
func (repo *RemoteRepo) BinaryURL(component string, architecture string) *url.URL {
	path := &url.URL{Path: fmt.Sprintf("dists/%s/%s/binary-%s/Packages", repo.Distribution, component, architecture)}
	return repo.archiveRootURL.ResolveReference(path)
}

// PackageURL returns URL of package file relative to repository root
// architecture
func (repo *RemoteRepo) PackageURL(filename string) *url.URL {
	path := &url.URL{Path: filename}
	return repo.archiveRootURL.ResolveReference(path)
}

// Fetch updates information about repository
func (repo *RemoteRepo) Fetch(d utils.Downloader) error {
	// Download release file to temporary URL
	release, err := utils.DownloadTemp(d, repo.ReleaseURL().String())
	if err != nil {
		return err
	}
	defer release.Close()

	paras, err := debc.Parse(release)
	if err != nil {
		return err
	}

	if len(paras) != 1 {
		return fmt.Errorf("wrong number of parts in Release file")
	}

	para := paras[0]

	architectures := strings.Split(para["Architectures"], " ")
	if len(repo.Architectures) == 0 {
		repo.Architectures = architectures
	} else {
		err = utils.StringsIsSubset(repo.Architectures, architectures,
			fmt.Sprintf("architecture %%s not available in repo %s", repo))
		if err != nil {
			return err
		}
	}

	components := strings.Split(para["Components"], " ")
	if len(repo.Components) == 0 {
		repo.Components = components
	} else {
		err = utils.StringsIsSubset(repo.Components, components,
			fmt.Sprintf("component %%s not available in repo %s", repo))
		if err != nil {
			return err
		}
	}

	return nil
}

// Download downloads all repo files
func (repo *RemoteRepo) Download(d utils.Downloader, db database.Storage, packageRepo *Repository) error {
	list := NewPackageList()

	// Download and parse all Release files
	for _, component := range repo.Components {
		for _, architecture := range repo.Architectures {
			packagesReader, packagesFile, err := utils.DownloadTryCompression(d, repo.BinaryURL(component, architecture).String())
			if err != nil {
				return err
			}
			defer packagesFile.Close()

			paras, err := debc.Parse(packagesReader)
			if err != nil {
				return err
			}

			for _, para := range paras {
				p := NewPackageFromControlFile(para)

				list.Add(p)
			}
		}
	}

	// Save package meta information to DB
	list.ForEach(func(p *Package) {
		db.Put(p.Key(), p.Encode())
	})

	// Download all package files
	ch := make(chan error, list.Length())
	count := 0

	list.ForEach(func(p *Package) {
		poolPath, err := packageRepo.PoolPath(p.Filename)
		if err == nil {
			if !p.VerifyFile(poolPath) {
				d.Download(repo.PackageURL(p.Filename).String(), poolPath, ch)
				count++
			}
		}
	})

	// Wait for all downloads to finish
	// TODO: report errors
	for count > 0 {
		_ = <-ch
		count--
	}

	return nil
}
