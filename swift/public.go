package swift

import (
	"fmt"
	"github.com/ncw/swift"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/files"
	"time"
	"os"
	"path/filepath"
	"net/http"
	"encoding/json"
)

// PublishedStorage abstract file system with published files (actually hosted on Swift)
type PublishedStorage struct {
	conn		    swift.Connection
	container	    string
	prefix	            string
	support_bulk_delete bool
}

type SwiftInfo map[string]interface{}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorage creates new instance of PublishedStorage with specified Swift access
// keys, tenant and tenantId
func NewPublishedStorage(username string, password string, authUrl string, tenant string, tenantId string, container string, prefix string) (*PublishedStorage, error) {
	if username == "" {
		username = os.Getenv("OS_USERNAME")
	}
	if password == "" {
		password = os.Getenv("OS_PASSWORD")
	}
	if authUrl == "" {
		authUrl = os.Getenv("OS_AUTH_URL")
	}
	if tenant == "" {
		tenant = os.Getenv("OS_TENANT_NAME")
	}
	if tenantId == "" {
		tenantId = os.Getenv("OS_TENANT_ID")
	}

	ct := swift.Connection{
		UserName:       username,
		ApiKey:	 password,
		AuthUrl:	authUrl,
		UserAgent:      "aptly/" + aptly.Version,
		Tenant:	 tenant,
		TenantId:       tenantId,
		ConnectTimeout: 60 * time.Second,
		Timeout:	60 * time.Second,
	}
	err := ct.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("Swift authentication failed: %s", err)
	}

	var bulk_delete bool
	resp, err := http.Get(filepath.Join(ct.StorageUrl, "..", "..") + "/info")
	if err == nil {
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		var infos SwiftInfo
		if decoder.Decode(&infos) == nil {
			_, bulk_delete = infos["bulk_delete"]
		}
	}

	result := &PublishedStorage{
		conn:		ct,
		container:	   container,
		prefix:	      prefix,
		support_bulk_delete: bulk_delete,
	}

	return result, nil
}

// String
func (storage *PublishedStorage) String() string {
	return fmt.Sprintf("Swift: %s:%s/%s", storage.conn.StorageUrl, storage.container, storage.prefix)
}

// MkDir creates directory recursively under public path
func (storage *PublishedStorage) MkDir(path string) error {
	// no op for Swift
	return nil
}

// PutFile puts file into published storage at specified path
func (storage *PublishedStorage) PutFile(path string, sourceFilename string) error {
	var (
		source *os.File
		err    error
	)
	source, err = os.Open(sourceFilename)
	if err != nil {
		return err
	}
	defer source.Close()

	_, err = storage.conn.ObjectPut(storage.container, filepath.Join(storage.prefix, path), source, false, "", "", nil)

	if err != nil {
		return fmt.Errorf("error uploading %s to %s: %s", sourceFilename, storage, err)
	}
	return nil
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	err := storage.conn.ObjectDelete(storage.container, filepath.Join(storage.prefix, path))

	if err != nil {
		return fmt.Errorf("error deleting %s from %s: %s", path, storage, err)
	}
	return nil
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	path = filepath.Join(storage.prefix, path)
	opts := swift.ObjectsOpts{
		Prefix: storage.prefix,
	}
	if objects, err := storage.conn.ObjectNamesAll(storage.container, &opts); err != nil {
		return fmt.Errorf("error removing dir %s from %s: %s", path, storage, err)
	} else {
		var multi_delete bool = true
		if storage.support_bulk_delete {
			_, err := storage.conn.BulkDelete(storage.container, objects)
			multi_delete = err != nil
		}
		if multi_delete {
			for _, name := range objects {
				if err := storage.conn.ObjectDelete(storage.container, name); err != nil {
					return err
				}
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
func (storage *PublishedStorage) LinkFromPool(publishedDirectory string, sourcePool aptly.PackagePool,
	sourcePath, sourceMD5 string, force bool) error {
	// verify that package pool is local pool in filesystem
	_ = sourcePool.(*files.PackagePool)

	baseName := filepath.Base(sourcePath)
	relPath := filepath.Join(publishedDirectory, baseName)
	poolPath := filepath.Join(storage.prefix, relPath)

	var (
		info swift.Object
		err  error
	)

	info, _, err = storage.conn.Object(storage.container, poolPath)
	if err != nil {
		if err != swift.ObjectNotFound {
			return fmt.Errorf("error getting information about %s from %s: %s", poolPath, storage, err)
		}
	} else {
		if !force && info.Hash != sourceMD5 {
			return fmt.Errorf("error putting file to %s: file already exists and is different: %s", poolPath, storage)

		}
	}

	return storage.PutFile(relPath, sourcePath)
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	prefix = filepath.Join(storage.prefix, prefix)
	if prefix != "" {
		prefix += "/"
	}
	opts := swift.ObjectsOpts{
		Prefix: prefix,
	}
	contents, err := storage.conn.ObjectNamesAll(storage.container, &opts)
	if err != nil {
		return nil, fmt.Errorf("error listing under prefix %s in %s: %s", prefix, storage, err)
	}

	return contents, nil
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	err := storage.conn.ObjectMove(storage.container, filepath.Join(storage.prefix, oldName), storage.container, filepath.Join(storage.prefix, newName))
	if err != nil {
		return fmt.Errorf("error copying %s -> %s in %s: %s", oldName, newName, storage, err)
	}

	return nil
}
