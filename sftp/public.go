package sftp

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kr/fs"
	"github.com/pkg/sftp"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/files"
	"golang.org/x/crypto/ssh"
)

type sftpFileInterface interface {
	Close() error
	Write(b []byte) (int, error)
}

type sftpClientInterface interface {
	CreatePath(path string) (sftpFileInterface, error)
	Mkdir(path string) error
	Remove(path string) error
	Rename(oldname, newname string) error
	Stat(p string) (os.FileInfo, error)
	Walk(root string) *fs.Walker
}

type sftpClient struct {
	*sftp.Client
}

func (s *sftpClient) CreatePath(path string) (sftpFileInterface, error) {
	// Call into our sftp.Client, coerce it's type into an interface type for
	// testing.
	return s.Create(path)
}

type sshClientInterface interface {
}

// PublishedStorage abstract file system with published files (actually hosted on S3)
type PublishedStorage struct {
	url  *url.URL
	ssh  sshClientInterface
	sftp sftpClientInterface
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorageInternal creates a new instance of PublishedStorage with
// the provided ssh and sftp clients.
func NewPublishedStorageInternal(url *url.URL, ssh sshClientInterface,
	sftp sftpClientInterface) (*PublishedStorage, error) {
	return &PublishedStorage{
		url:  url,
		ssh:  ssh,
		sftp: sftp,
	}, nil
}

// NewPublishedStorage creates new instance of PublishedStorage from the
// specified URI string.
func NewPublishedStorage(uri string) (*PublishedStorage, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	key, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh/id_rsa"))
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	if url.User != nil && url.User.Username() != "" {
		config.User = url.User.Username()
		password, set := url.User.Password()
		if set {
			config.Auth = append(config.Auth, ssh.Password(password))
		}
	}

	host := url.Host
	if !strings.Contains(host, ":") {
		host = host + ":22" // ssh expects an explicit port in the host string
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, err
	}
	// FIXME: does this leak? hell if I know!
	// defer client.Close()

	sftp, err := sftp.NewClient(client)
	if err != nil {
		return nil, err
	}
	// defer sftp.Close()

	return NewPublishedStorageInternal(url, client, &sftpClient{sftp})
}

func (storage *PublishedStorage) expandPath(path string) string {
	if strings.HasPrefix(path, storage.url.Path) {
		return path
	}
	return filepath.Join(storage.url.Path, path)
}

// String
func (storage *PublishedStorage) String() string {
	return fmt.Sprintf("SFTP: %s", storage.url.String())
}

// MkDir creates directory recursively under public path
func (storage *PublishedStorage) MkDir(path string) error {
	path = storage.expandPath(path)
	fmt.Fprintf(os.Stderr, "MkDir %s\n", path)
	_, err := storage.sftp.Stat(path)
	if err != nil {
		parts := strings.Split(path, "/")
		traversedPath := ""
		if strings.HasPrefix(path, "/") {
			traversedPath = "/"
		}
		for _, part := range parts {
			traversedPath = filepath.Join(traversedPath, part)
			_ = storage.sftp.Mkdir(traversedPath)
		}
		filepath.Split(path)
	}
	_, err = storage.sftp.Stat(path)
	return err
}

// PutFile puts file into published storage at specified path
func (storage *PublishedStorage) PutFile(path string, sourceFilename string) error {
	path = storage.expandPath(path)
	fmt.Fprintf(os.Stderr, "PutFile %s (%s)\n", path, sourceFilename)
	var (
		source *os.File
		err    error
	)
	source, err = os.Open(sourceFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open fail \n")
		return err
	}
	defer source.Close()

	target, err := storage.sftp.CreatePath(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create fail\n")
		return err
	}
	defer target.Close()

	data, err := ioutil.ReadAll(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "readall fail\n")
		return err
	}
	_, err = target.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write fail\n")
		return err
	}

	return nil
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	fmt.Fprintf(os.Stderr, "Remove %s\n", path)
	return storage.sftp.Remove(storage.expandPath(path))
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	fmt.Fprintf(os.Stderr, "RemoveDirs %s\n", path)

	// This is non-recursive.
	// We walk th entire tree under the prefix and delete all files.
	// Meanwhile we put all directories into a map by their weight of slashes
	// i.e. how deep in the tree they are.
	dirMap := make(map[int][]string)
	walker := storage.sftp.Walk(storage.expandPath(path))
	for walker.Step() {
		if err := walker.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR WHILE WALKING %s\n", err)
			continue
		}
		if walker.Stat().Mode().IsDir() {
			dir := walker.Path()
			weight := strings.Count(dir, "/")
			dirMap[weight] = append(dirMap[weight], dir)
			continue
		}
		// If it is a file, drop it immediately.
		err := storage.sftp.Remove(walker.Path())
		if err != nil {
			return err
		}
	}

	// Now that all directories are empty all that's left is to sort the
	// directories by their weight and then delete them in the reverse order
	// such that the deepest directory is deleted first. Thus we always only
	// delete empty directories.
	var keys []int
	for k := range dirMap {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))

	for _, k := range keys {
		dirs := dirMap[k]
		for _, dir := range dirs {
			err := storage.sftp.Remove(dir)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// LinkFromPool links package file from pool to dist's pool location
//
// publishedDirectory is desired location in pool (like prefix/pool/component/liba/libav/)
// sourcePool is instance of aptly.PackagePool
// sourcePath is filepath to package file in package pool
//
// LinkFromPool returns relative path for the published file to be included in package index
func (storage *PublishedStorage) LinkFromPool(publishedDirectory string,
	sourcePool aptly.PackagePool,
	sourcePath, sourceMD5 string, force bool) error {
	fmt.Fprintf(os.Stderr, "LinkFromPool dir:%s pool:  srcpath:%s force:%v\n", publishedDirectory, sourcePath, force)

	// verify that package pool is local pool in filesystem
	_ = sourcePool.(*files.PackagePool)

	baseName := filepath.Base(sourcePath)
	relPath := filepath.Join(publishedDirectory, baseName)
	poolPath := storage.expandPath(relPath)
	fmt.Fprintf(os.Stderr, "LinkFromPool basename:%s relpath:%s poolpath:%s\n", baseName, relPath, poolPath)

	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// FIXME: we do not do md5 validation or anything!
	// if storage.pathCache == nil {
	// 	paths, md5s, err := storage.internalFilelist(storage.prefix, true)
	// 	if err != nil {
	// 		return fmt.Errorf("error caching paths under prefix: %s", err)
	// 	}
	//
	// 	storage.pathCache = make(map[string]string, len(paths))
	//
	// 	for i := range paths {
	// 		storage.pathCache[paths[i]] = md5s[i]
	// 	}
	// }

	// destinationMD5, exists := storage.pathCache[relPath]

	exists := true
	_, err := storage.sftp.Stat(poolPath)
	if err != nil {
		exists = false
	}

	if exists {
		return nil
		// if destinationMD5 == sourceMD5 {
		// 	return nil
		// }
		//
		// if !force && destinationMD5 != sourceMD5 {
		// 	return fmt.Errorf("error putting file to %s: file already exists and is different: %s", poolPath, storage)
		//
		// }
	}

	err = storage.MkDir(filepath.Dir(relPath))
	err = storage.PutFile(relPath, sourcePath)
	// if err == nil {
	// 	storage.pathCache[relPath] = sourceMD5
	// }

	return err
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	fmt.Fprintf(os.Stderr, "Filelist %s\n", prefix)
	expPrefix := storage.expandPath(prefix)
	paths := []string{}
	walker := storage.sftp.Walk(expPrefix)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR WHILE WALKING %s\n", err)
			continue
		}
		if walker.Stat().Mode().IsDir() {
			// We only list files!
			continue
		}
		paths = append(paths, strings.Replace(walker.Path(), expPrefix+"/", "", 1))
	}
	return paths, nil
}

func (storage *PublishedStorage) exists(path string) bool {
	_, err := storage.sftp.Stat(storage.expandPath(path))
	if err != nil {
		return false
	}
	return true
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	fmt.Fprintf(os.Stderr, "RenameFile %s %s\n", oldName, newName)

	err := storage.MkDir(filepath.Dir(newName))
	if err != nil {
		return err
	}

	// SFTP rename is actually link which fails if the target already exists.
	// Alas, nothing to be done than remove first, which technically opens
	// a short time window during which the path isn't there.
	if storage.exists(newName) {
		err = storage.Remove(newName)
		if err != nil {
			return err
		}
	}

	return storage.sftp.Rename(storage.expandPath(oldName),
		storage.expandPath(newName))
}
