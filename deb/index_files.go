package deb

import (
	"bufio"
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"os"
	"path/filepath"
	"strings"
)

type indexFiles struct {
	publishedStorage aptly.PublishedStorage
	basePath         string
	renameMap        map[string]string
	generatedFiles   map[string]utils.ChecksumInfo
	tempDir          string
	suffix           string
	indexes          map[string]*indexFile
}

type indexFile struct {
	parent       *indexFiles
	discardable  bool
	ignoreFlat   bool
	compressOnly bool
	compressGzip bool
	compressBzip bool
	compressGxz  bool
	signable     bool
	relativePath string
	tempFilename string
	tempFile     *os.File
	w            *bufio.Writer
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

func (file *indexFile) Finalize(signer utils.Signer) error {
	if file.w == nil {
		if file.discardable {
			return nil
		}
		file.BufWriter()
	}

	err := file.w.Flush()
	if err != nil {
		file.tempFile.Close()
		return fmt.Errorf("unable to write to index file: %s", err)
	}

	if file.compressGzip || file.compressBzip || file.compressGxz {
		err = utils.CompressFile(file.tempFile)
		if err != nil {
			file.tempFile.Close()
			return fmt.Errorf("unable to compress index file: %s", err)
		}
	}

	file.tempFile.Close()

	exts := []string{}
	if !file.ignoreFlat {
		exts = append(exts, "")
	}
	if file.compressGzip {
		exts = append(exts, ".gz")
	}
	if file.compressBzip {
		exts = append(exts, ".bz2")
	}
	if file.compressGxz {
		exts = append(exts, ".xz")
	}

	for _, ext := range exts {
		var checksumInfo utils.ChecksumInfo

		checksumInfo, err = utils.ChecksumsForFile(file.tempFilename + ext)
		if err != nil {
			return fmt.Errorf("unable to collect checksums: %s", err)
		}
		file.parent.generatedFiles[file.relativePath+ext] = checksumInfo
	}

	err = file.parent.publishedStorage.MkDir(filepath.Dir(filepath.Join(file.parent.basePath, file.relativePath)))
	if err != nil {
		return fmt.Errorf("unable to create dir: %s", err)
	}

	for _, ext := range exts {
		if file.compressOnly && !(ext == ".gz" || ext == ".bz2" || ext == ".xz") {
			continue
		}

		err = file.parent.publishedStorage.PutFile(filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+ext),
			file.tempFilename+ext)
		if err != nil {
			return fmt.Errorf("unable to publish file: %s", err)
		}

		if file.parent.suffix != "" {
			file.parent.renameMap[filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+ext)] =
				filepath.Join(file.parent.basePath, file.relativePath+ext)
		}
	}

	if file.signable && signer != nil {
		err = signer.DetachedSign(file.tempFilename, file.tempFilename+".gpg")
		if err != nil {
			return fmt.Errorf("unable to detached sign file: %s", err)
		}

		err = signer.ClearSign(file.tempFilename, filepath.Join(filepath.Dir(file.tempFilename), "In"+filepath.Base(file.tempFilename)))
		if err != nil {
			return fmt.Errorf("unable to clearsign file: %s", err)
		}

		if file.parent.suffix != "" {
			file.parent.renameMap[filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+".gpg")] =
				filepath.Join(file.parent.basePath, file.relativePath+".gpg")
			file.parent.renameMap[filepath.Join(file.parent.basePath, "In"+file.relativePath+file.parent.suffix)] =
				filepath.Join(file.parent.basePath, "In"+file.relativePath)
		}

		err = file.parent.publishedStorage.PutFile(filepath.Join(file.parent.basePath, file.relativePath+file.parent.suffix+".gpg"),
			file.tempFilename+".gpg")
		if err != nil {
			return fmt.Errorf("unable to publish file: %s", err)
		}

		err = file.parent.publishedStorage.PutFile(filepath.Join(file.parent.basePath, "In"+file.relativePath+file.parent.suffix),
			filepath.Join(filepath.Dir(file.tempFilename), "In"+filepath.Base(file.tempFilename)))
		if err != nil {
			return fmt.Errorf("unable to publish file: %s", err)
		}
	}

	return nil
}

func newIndexFiles(publishedStorage aptly.PublishedStorage, basePath, tempDir, suffix string) *indexFiles {
	return &indexFiles{
		publishedStorage: publishedStorage,
		basePath:         basePath,
		renameMap:        make(map[string]string),
		generatedFiles:   make(map[string]utils.ChecksumInfo),
		tempDir:          tempDir,
		suffix:           suffix,
		indexes:          make(map[string]*indexFile),
	}
}

func (files *indexFiles) PackageIndex(component, arch string, udeb bool) *indexFile {
	if arch == "source" {
		udeb = false
	}
	key := fmt.Sprintf("pi-%s-%s-%v", component, arch, udeb)
	file, ok := files.indexes[key]
	if !ok {
		var relativePath string

		if arch == "source" {
			relativePath = filepath.Join(component, "source", "Sources")
		} else {
			if udeb {
				relativePath = filepath.Join(component, "debian-installer", fmt.Sprintf("binary-%s", arch), "Packages")
			} else {
				relativePath = filepath.Join(component, fmt.Sprintf("binary-%s", arch), "Packages")
			}
		}

		file = &indexFile{
			parent:       files,
			discardable:  false,
			compressGzip: true,
			compressBzip: true,
			signable:     false,
			relativePath: relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) ReleaseIndex(component, arch string, udeb bool) *indexFile {
	if arch == "source" {
		udeb = false
	}
	key := fmt.Sprintf("ri-%s-%s-%v", component, arch, udeb)
	file, ok := files.indexes[key]
	if !ok {
		var relativePath string

		if arch == "source" {
			relativePath = filepath.Join(component, "source", "Release")
		} else {
			if udeb {
				relativePath = filepath.Join(component, "debian-installer", fmt.Sprintf("binary-%s", arch), "Release")
			} else {
				relativePath = filepath.Join(component, fmt.Sprintf("binary-%s", arch), "Release")
			}
		}

		file = &indexFile{
			parent:       files,
			discardable:  udeb,
			signable:     false,
			relativePath: relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) ContentsIndex(component, arch string, udeb bool) *indexFile {
	if arch == "source" {
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
			parent:       files,
			discardable:  true,
			ignoreFlat:   true,
			compressOnly: true,
			compressGzip: true,
			signable:     false,
			relativePath: relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) AppStreamIndex(component, name string) *indexFile {
	key := fmt.Sprintf("ai-%s-%s", component, name)
	file, ok := files.indexes[key]
	if !ok {
		relativePath := filepath.Join(component, "appstream", name)
		ext := filepath.Ext(name)

		xz := false
		if ext == ".yml" {
			xz = true
		}

		file = &indexFile{
			parent:       files,
			discardable:  false,
			compressOnly: true,
			compressGzip: true,
			compressGxz:  xz,
			signable:     false,
			relativePath: relativePath,
		}

		files.indexes[key] = file
	}

	return file
}

func (files *indexFiles) ReleaseFile() *indexFile {
	return &indexFile{
		parent:        files,
		discardable:   false,
		signable:      true,
		relativePath:  "Release",
	}
}

func (files *indexFiles) FinalizeAll(progress aptly.Progress) (err error) {
	if progress != nil {
		progress.InitBar(int64(len(files.indexes)), false)
		defer progress.ShutdownBar()
	}

	for _, file := range files.indexes {
		err = file.Finalize(nil)
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
