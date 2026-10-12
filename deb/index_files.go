package deb

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/utils"
)

type indexFiles struct {
	publishedStorage aptly.PublishedStorage
	basePath         string
	renameMap        map[string]string
	generatedFiles   map[string]utils.ChecksumInfo
	tempDir          string
	suffix           string
	indexes          map[string]*indexFile
	acquireByHash    bool
	byHashAlgorithms []string
	skipBz2          bool
}

type indexFile struct {
	parent        *indexFiles
	discardable   bool
	compressable  bool
	onlyGzip      bool
	clearSign     bool
	detachedSign  bool
	acquireByHash bool
	relativePath  string
	tempFilename  string
	tempFile      *os.File
	w             *bufio.Writer
}

func (file *indexFile) BufWriter() (*bufio.Writer, error) {
	if file.w == nil {
		var err error
		file.tempFilename = filepath.Join(file.parent.tempDir, strings.Replace(file.relativePath, "/", "_", -1))
		file.tempFile, err = os.Create(file.tempFilename)
		if err != nil {
			return nil, fmt.Errorf("unable to create temporary index file: %s", err)
		}

		file.w = bufio.NewWriter(file.tempFile)
	}

	return file.w, nil
}

func (file *indexFile) Finalize(signer pgp.Signer) error {
	if file.w == nil {
		if file.discardable {
			return nil
		}
		_, _ = file.BufWriter()
	}

	err := file.w.Flush()
	if err != nil {
		_ = file.tempFile.Close()
		return fmt.Errorf("unable to write to index file: %s", err)
	}

	if file.compressable {
		err = utils.CompressFile(file.tempFile, file.onlyGzip || file.parent.skipBz2)
		if err != nil {
			_ = file.tempFile.Close()
			return fmt.Errorf("unable to compress index file: %s", err)
		}
	}

	_ = file.tempFile.Close()

	exts := []string{""}
	cksumExts := exts
	if file.compressable {
		if file.onlyGzip {
			exts = []string{".gz"}
			cksumExts = []string{"", ".gz"}
		} else {
			exts = append(exts, ".gz")
			if !file.parent.skipBz2 {
				exts = append(exts, ".bz2")
			}
			cksumExts = exts
		}
	}

	for _, ext := range cksumExts {
		var checksumInfo utils.ChecksumInfo

		checksumInfo, err = utils.ChecksumsForFile(file.tempFilename + ext)
		if err != nil {
			return fmt.Errorf("unable to collect checksums: %s", err)
		}
		file.parent.generatedFiles[file.relativePath+ext] = checksumInfo
	}

	filedir := filepath.Dir(filepath.Join(file.parent.basePath, file.relativePath))

	err = file.parent.publishedStorage.MkDir(filedir)
	if err != nil {
		return fmt.Errorf("unable to create dir: %s", err)
	}

	if file.acquireByHash {
		algorithms, algErr := file.parent.hashAlgorithms()
		if algErr != nil {
			return algErr
		}
		for _, hash := range algorithms {
			err = file.parent.publishedStorage.MkDir(filepath.Join(filedir, "by-hash", hash))
			if err != nil {
				return fmt.Errorf("unable to create dir: %s", err)
			}
		}
	}

	for _, ext := range exts {
		err = file.parent.publishedStorage.PutFile(filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+ext),
			file.tempFilename+ext)
		if err != nil {
			return fmt.Errorf("unable to publish file: %s", err)
		}

		if file.parent.suffix != "" {
			file.parent.renameMap[filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+ext)] =
				filepath.Join(file.parent.basePath, file.relativePath+ext)
		}

		if file.acquireByHash {
			sums := file.parent.generatedFiles[file.relativePath+ext]
			all := map[string]string{"SHA512": sums.SHA512, "SHA256": sums.SHA256, "SHA1": sums.SHA1, "MD5Sum": sums.MD5}
			algorithms, algErr := file.parent.hashAlgorithms()
			if algErr != nil {
				return algErr
			}
			for _, hash := range algorithms {
				err = packageIndexByHash(file, ext, hash, all[hash])
				if err != nil {
					return fmt.Errorf("unable to build hash file: %s", err)
				}
			}
		}
	}

	if signer != nil {
		gpgExt := ".gpg"
		if file.detachedSign {
			err = signer.DetachedSign(file.tempFilename, file.tempFilename+gpgExt)
			if err != nil {
				return fmt.Errorf("unable to detached sign file: %s", err)
			}

			if file.parent.suffix != "" {
				file.parent.renameMap[filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+gpgExt)] =
					filepath.Join(file.parent.basePath, file.relativePath+gpgExt)
			}

			err = file.parent.publishedStorage.PutFile(filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+gpgExt),
				file.tempFilename+gpgExt)
			if err != nil {
				return fmt.Errorf("unable to publish file: %s", err)
			}

		}

		if file.clearSign {
			err = signer.ClearSign(file.tempFilename, filepath.Join(filepath.Dir(file.tempFilename), "In"+filepath.Base(file.tempFilename)))
			if err != nil {
				return fmt.Errorf("unable to clearsign file: %s", err)
			}

			if file.parent.suffix != "" {
				file.parent.renameMap[filepath.Join(file.parent.basePath, "In"+file.relativePath+file.parent.suffix)] =
					filepath.Join(file.parent.basePath, "In"+file.relativePath)
			}

			err = file.parent.publishedStorage.PutFile(filepath.Join(file.parent.basePath, "In"+file.relativePath+file.parent.suffix),
				filepath.Join(filepath.Dir(file.tempFilename), "In"+filepath.Base(file.tempFilename)))
			if err != nil {
				return fmt.Errorf("unable to publish file: %s", err)
			}
		}
	}

	return nil
}

// knownHashAlgorithms are the algorithms aptly can write Acquire-By-Hash
// entries for, in the order the entries are written.
var knownHashAlgorithms = []string{"MD5Sum", "SHA1", "SHA256", "SHA512"}

// hashAlgorithms is the set of Acquire-By-Hash algorithms to write entries for.
// Empty means all of them, which is what aptly did before the set was
// configurable, so an existing config behaves exactly as before.
//
// The order is knownHashAlgorithms' rather than the config's, so that the
// directories created in one loop and the entries written in another always
// agree whatever order the user listed.
//
// An unrecognised name is an error rather than something to skip. Skipping it
// would turn a typo into "no by-hash entries for that algorithm", which looks
// identical to having configured it away.
func (files *indexFiles) hashAlgorithms() ([]string, error) {
	if len(files.byHashAlgorithms) == 0 {
		return knownHashAlgorithms, nil
	}

	wanted := make(map[string]bool, len(files.byHashAlgorithms))
	for _, want := range files.byHashAlgorithms {
		known := false
		for _, k := range knownHashAlgorithms {
			if k == want {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("unknown Acquire-By-Hash algorithm %q, expected one of %v",
				want, knownHashAlgorithms)
		}
		wanted[want] = true
	}

	ordered := make([]string, 0, len(wanted))
	for _, known := range knownHashAlgorithms {
		if wanted[known] {
			ordered = append(ordered, known)
		}
	}
	return ordered, nil
}

func packageIndexByHash(file *indexFile, ext string, hash string, sum string) error {
	src := filepath.Join(file.parent.basePath, file.relativePath)
	indexfile := path.Base(src + ext)
	src = src + file.parent.suffix + ext
	filedir := filepath.Dir(filepath.Join(file.parent.basePath, file.relativePath))
	dst := filepath.Join(filedir, "by-hash", hash)
	sumfilePath := filepath.Join(dst, sum)

	// link already exists? do nothing
	exists, err := file.parent.publishedStorage.FileExists(sumfilePath)
	if err != nil {
		return fmt.Errorf("Acquire-By-Hash: error checking exists of file %s: %s", sumfilePath, err)
	}
	if exists {
		return nil
	}

	// create the link
	err = file.parent.publishedStorage.HardLink(src, sumfilePath)
	if err != nil {
		return fmt.Errorf("Acquire-By-Hash: error creating hardlink %s: %s", sumfilePath, err)
	}

	// if a previous index file already exists exists, backup symlink
	indexPath := filepath.Join(dst, indexfile)
	oldIndexPath := filepath.Join(dst, indexfile+".old")
	if exists, _ = file.parent.publishedStorage.FileExists(indexPath); exists {
		// if exists, remove old symlink
		if exists, _ = file.parent.publishedStorage.FileExists(oldIndexPath); exists {
			var linkTarget string
			linkTarget, err = file.parent.publishedStorage.ReadLink(oldIndexPath)
			if err == nil {
				// If we managed to resolve the link target: delete it. This is the
				// oldest physical index file we no longer need. Once we drop our
				// old symlink we'll essentially forget about it existing at all.
				_ = file.parent.publishedStorage.Remove(linkTarget)
			}
			_ = file.parent.publishedStorage.Remove(oldIndexPath)
		}
		_ = file.parent.publishedStorage.RenameFile(indexPath, oldIndexPath)
	}

	// create symlink
	err = file.parent.publishedStorage.SymLink(filepath.Join(dst, sum), filepath.Join(dst, indexfile))
	if err != nil {
		return fmt.Errorf("Acquire-By-Hash: error creating symlink %s: %s", filepath.Join(dst, indexfile), err)
	}
	return nil
}

func newIndexFiles(publishedStorage aptly.PublishedStorage, basePath, tempDir, suffix string, acquireByHash bool, byHashAlgorithms []string, skipBz2 bool) *indexFiles {
	return &indexFiles{
		publishedStorage: publishedStorage,
		basePath:         basePath,
		renameMap:        make(map[string]string),
		generatedFiles:   make(map[string]utils.ChecksumInfo),
		tempDir:          tempDir,
		suffix:           suffix,
		indexes:          make(map[string]*indexFile),
		acquireByHash:    acquireByHash,
		byHashAlgorithms: byHashAlgorithms,
		skipBz2:          skipBz2,
	}
}

func (files *indexFiles) PackageIndex(component, arch string, udeb bool, installer bool, distribution string) *indexFile {
	if arch == ArchitectureSource {
		udeb = false
	}
	key := fmt.Sprintf("pi-%s-%s-%v-%v", component, arch, udeb, installer)
	file, ok := files.indexes[key]
	if !ok {
		var relativePath string

		if arch == ArchitectureSource {
			relativePath = filepath.Join(component, "source", "Sources")
		} else {
			if udeb {
				relativePath = filepath.Join(component, "debian-installer", fmt.Sprintf("binary-%s", arch), "Packages")
			} else if installer {
				if distribution == aptly.DistributionFocal {
					relativePath = filepath.Join(component, fmt.Sprintf("installer-%s", arch), "current", "legacy-images", "SHA256SUMS")
				} else {
					relativePath = filepath.Join(component, fmt.Sprintf("installer-%s", arch), "current", "images", "SHA256SUMS")
				}
			} else {
				relativePath = filepath.Join(component, fmt.Sprintf("binary-%s", arch), "Packages")
			}
		}

		file = &indexFile{
			parent:        files,
			discardable:   false,
			compressable:  !installer,
			detachedSign:  installer,
			clearSign:     false,
			acquireByHash: files.acquireByHash,
			relativePath:  relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) ReleaseIndex(component, arch string, udeb bool) *indexFile {
	if arch == ArchitectureSource {
		udeb = false
	}
	key := fmt.Sprintf("ri-%s-%s-%v", component, arch, udeb)
	file, ok := files.indexes[key]
	if !ok {
		var relativePath string

		if arch == ArchitectureSource {
			relativePath = filepath.Join(component, "source", "Release")
		} else {
			if udeb {
				relativePath = filepath.Join(component, "debian-installer", fmt.Sprintf("binary-%s", arch), "Release")
			} else {
				relativePath = filepath.Join(component, fmt.Sprintf("binary-%s", arch), "Release")
			}
		}

		file = &indexFile{
			parent:        files,
			discardable:   udeb,
			compressable:  false,
			detachedSign:  false,
			clearSign:     false,
			acquireByHash: files.acquireByHash,
			relativePath:  relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) ContentsIndex(component, arch string, udeb bool) *indexFile {
	if arch == ArchitectureSource {
		udeb = false
	}
	key := fmt.Sprintf("ci-%s-%s-%v", component, arch, udeb)
	file, ok := files.indexes[key]
	if !ok {
		var relativePath string

		if udeb {
			relativePath = filepath.Join(component, fmt.Sprintf("Contents-udeb-%s", arch))
		} else {
			relativePath = filepath.Join(component, fmt.Sprintf("Contents-%s", arch))
		}

		file = &indexFile{
			parent:        files,
			discardable:   true,
			compressable:  true,
			onlyGzip:      true,
			detachedSign:  false,
			clearSign:     false,
			acquireByHash: files.acquireByHash,
			relativePath:  relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) LegacyContentsIndex(arch string, udeb bool) *indexFile {
	if arch == ArchitectureSource {
		udeb = false
	}
	key := fmt.Sprintf("lci-%s-%v", arch, udeb)
	file, ok := files.indexes[key]
	if !ok {
		var relativePath string

		if udeb {
			relativePath = fmt.Sprintf("Contents-udeb-%s", arch)
		} else {
			relativePath = fmt.Sprintf("Contents-%s", arch)
		}

		file = &indexFile{
			parent:        files,
			discardable:   true,
			compressable:  true,
			onlyGzip:      true,
			detachedSign:  false,
			clearSign:     false,
			acquireByHash: files.acquireByHash,
			relativePath:  relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) SkelIndex(component, path string) *indexFile {
	key := fmt.Sprintf("si-%s-%s", component, path)
	file, ok := files.indexes[key]

	if !ok {
		relativePath := filepath.Join(component, path)

		file = &indexFile{
			parent:       files,
			discardable:  false,
			compressable: false,
			onlyGzip:     false,
			relativePath: relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) ReleaseFile() *indexFile {
	return &indexFile{
		parent:       files,
		discardable:  false,
		compressable: false,
		detachedSign: true,
		clearSign:    true,
		relativePath: "Release",
	}
}

func (files *indexFiles) FinalizeAll(progress aptly.Progress, signer pgp.Signer) (err error) {
	if progress != nil {
		progress.InitBar(int64(len(files.indexes)), false, aptly.BarPublishFinalizeIndexes)
		defer progress.ShutdownBar()
	}

	for _, file := range files.indexes {
		err = file.Finalize(signer)
		if err != nil {
			return
		}
		if progress != nil {
			progress.AddBar(1)
		}
	}

	files.indexes = make(map[string]*indexFile)

	return
}

func (files *indexFiles) RenameFiles() error {
	var err error

	for oldName, newName := range files.renameMap {
		err = files.publishedStorage.RenameFile(oldName, newName)
		if err != nil {
			return fmt.Errorf("unable to rename: %s", err)
		}
	}

	return nil
}
