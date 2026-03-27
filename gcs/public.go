package gcs

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// PublishedStorage abstracts published files hosted on GCS.
type PublishedStorage struct {
	client          *storage.Client
	bucket          *storage.BucketHandle
	bucketName      string
	prefix          string
	acl             string
	storageClass    string
	encryptionKey   string
	disableMultiDel bool
	debug           bool
	pathCache       map[string]string
}

var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorage creates a GCS-backed published storage.
func NewPublishedStorage(bucket, prefix, credentialsFile, serviceAccountJSON,
	project, defaultACL, storageClass, encryptionKey string,
	disableMultiDel, debug bool) (*PublishedStorage, error) {

	ctx := context.TODO()
	opts := make([]option.ClientOption, 0, 2)

	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	} else if serviceAccountJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(serviceAccountJSON)))
	}

	if project != "" {
		opts = append(opts, option.WithQuotaProject(project))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	result := &PublishedStorage{
		client:          client,
		bucket:          client.Bucket(bucket),
		bucketName:      bucket,
		prefix:          prefix,
		acl:             defaultACL,
		storageClass:    storageClass,
		encryptionKey:   encryptionKey,
		disableMultiDel: disableMultiDel,
		debug:           debug,
	}

	return result, nil
}

func (g *PublishedStorage) String() string {
	return fmt.Sprintf("GCS: %s/%s", g.bucketName, g.prefix)
}

// MkDir creates directory recursively under public path.
func (g *PublishedStorage) MkDir(_ string) error {
	// no-op for GCS
	return nil
}

func (g *PublishedStorage) objectPath(path string) string {
	return filepath.Join(g.prefix, path)
}

func (g *PublishedStorage) objectHandle(path string) *storage.ObjectHandle {
	obj := g.bucket.Object(g.objectPath(path))
	if g.encryptionKey != "" {
		obj = obj.Key([]byte(g.encryptionKey))
	}

	return obj
}

// PutFile puts file into published storage at specified path.
func (g *PublishedStorage) PutFile(path string, sourceFilename string) error {
	source, err := os.Open(sourceFilename)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	if g.debug {
		log.Debug().Msgf("GCS: PutFile '%s'", path)
	}

	err = g.putFile(path, source, "")
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error uploading %s to %s", sourceFilename, g))
	}

	return nil
}

func (g *PublishedStorage) applyACL(obj *storage.ObjectHandle) error {
	switch g.acl {
	case "", "none", "private":
		return nil
	case "public-read":
		return obj.ACL().Set(context.TODO(), storage.AllUsers, storage.RoleReader)
	default:
		return fmt.Errorf("unsupported GCS ACL value: %s", g.acl)
	}
}

func (g *PublishedStorage) putFile(path string, source io.Reader, sourceMD5 string) error {
	obj := g.objectHandle(path)
	writer := obj.NewWriter(context.TODO())

	if g.storageClass != "" {
		writer.ObjectAttrs.StorageClass = g.storageClass
	}
	if sourceMD5 != "" {
		writer.Metadata = map[string]string{"Md5": sourceMD5}
	}

	if _, err := io.Copy(writer, source); err != nil {
		_ = writer.Close()
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	return g.applyACL(obj)
}

func (g *PublishedStorage) getMD5(path string) (string, error) {
	attrs, err := g.objectHandle(path).Attrs(context.TODO())
	if err != nil {
		return "", err
	}

	if attrs.Metadata != nil {
		if md5, ok := attrs.Metadata["Md5"]; ok && md5 != "" {
			return strings.ToLower(md5), nil
		}
	}

	return strings.ToLower(hex.EncodeToString(attrs.MD5)), nil
}

// Remove removes single file under public path.
func (g *PublishedStorage) Remove(path string) error {
	if g.debug {
		log.Debug().Msgf("GCS: Remove '%s'", path)
	}

	err := g.objectHandle(path).Delete(context.TODO())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil
		}

		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			return nil
		}

		return errors.Wrap(err, fmt.Sprintf("error deleting %s from %s", path, g))
	}

	delete(g.pathCache, path)

	return nil
}

// RemoveDirs removes directory structure under public path.
func (g *PublishedStorage) RemoveDirs(path string, _ aptly.Progress) error {
	filelist, _, err := g.internalFilelist(path)
	if err != nil {
		return err
	}

	if g.debug {
		log.Debug().Msgf("GCS: RemoveDirs '%s'", path)
	}

	for _, file := range filelist {
		objPath := filepath.Join(path, file)
		if err := g.Remove(objPath); err != nil {
			return err
		}
	}

	_ = g.disableMultiDel

	return nil
}

// LinkFromPool links package file from pool to dist's pool location.
func (g *PublishedStorage) LinkFromPool(publishedPrefix, publishedRelPath, fileName string, sourcePool aptly.PackagePool,
	sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error {

	publishedDirectory := filepath.Join(publishedPrefix, publishedRelPath)
	relPath := filepath.Join(publishedDirectory, fileName)
	poolPath := filepath.Join(g.prefix, relPath)

	if g.pathCache == nil {
		paths, md5s, err := g.internalFilelist(filepath.Join(publishedPrefix, "pool"))
		if err != nil {
			return errors.Wrap(err, "error caching paths under prefix")
		}

		g.pathCache = make(map[string]string, len(paths))
		for i := range paths {
			g.pathCache[filepath.Join(publishedPrefix, "pool", paths[i])] = md5s[i]
		}
	}

	destinationMD5, exists := g.pathCache[relPath]
	sourceMD5 := strings.ToLower(sourceChecksums.MD5)

	if exists {
		if sourceMD5 == "" {
			return fmt.Errorf("unable to compare object, MD5 checksum missing")
		}

		if len(destinationMD5) != 32 {
			var err error
			destinationMD5, err = g.getMD5(relPath)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error verifying MD5 for %s: %s", g, poolPath))
			}
			g.pathCache[relPath] = destinationMD5
		}

		if destinationMD5 == sourceMD5 {
			return nil
		}

		if !force {
			return fmt.Errorf("error putting file to %s: file already exists and is different: %s", poolPath, g)
		}
	}

	source, err := sourcePool.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	if g.debug {
		log.Debug().Msgf("GCS: LinkFromPool '%s'", relPath)
	}

	err = g.putFile(relPath, source, sourceMD5)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error uploading %s to %s: %s", sourcePath, g, poolPath))
	}

	g.pathCache[relPath] = sourceMD5

	return nil
}

// Filelist returns list of files under prefix.
func (g *PublishedStorage) Filelist(prefix string) ([]string, error) {
	paths, _, err := g.internalFilelist(prefix)
	return paths, err
}

func (g *PublishedStorage) internalFilelist(prefix string) ([]string, []string, error) {
	paths := make([]string, 0, 1024)
	md5s := make([]string, 0, 1024)

	fullPrefix := filepath.Join(g.prefix, prefix)
	if fullPrefix != "" {
		fullPrefix += "/"
	}

	it := g.bucket.Objects(context.TODO(), &storage.Query{Prefix: fullPrefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "error listing under prefix %s in %s", fullPrefix, g)
		}

		path := attrs.Name
		if fullPrefix != "" {
			path = strings.TrimPrefix(path, fullPrefix)
		}
		paths = append(paths, path)

		if attrs.Metadata != nil {
			if md5, ok := attrs.Metadata["Md5"]; ok && md5 != "" {
				md5s = append(md5s, strings.ToLower(md5))
				continue
			}
		}

		md5s = append(md5s, strings.ToLower(hex.EncodeToString(attrs.MD5)))
	}

	return paths, md5s, nil
}

// RenameFile renames (moves) file.
func (g *PublishedStorage) RenameFile(oldName, newName string) error {
	src := g.objectHandle(oldName)
	dst := g.objectHandle(newName)

	if g.debug {
		log.Debug().Msgf("GCS: RenameFile %s -> %s", oldName, newName)
	}

	_, err := dst.CopierFrom(src).Run(context.TODO())
	if err != nil {
		return fmt.Errorf("error copying %s -> %s in %s: %s", oldName, newName, g, err)
	}

	err = g.applyACL(dst)
	if err != nil {
		return err
	}

	return g.Remove(oldName)
}

// SymLink creates a copy of src file and stores link information in metadata.
func (g *PublishedStorage) SymLink(src string, dst string) error {
	source := g.objectHandle(src)
	dest := g.objectHandle(dst)

	if g.debug {
		log.Debug().Msgf("GCS: SymLink %s -> %s", src, dst)
	}

	_, err := dest.CopierFrom(source).Run(context.TODO())
	if err != nil {
		return fmt.Errorf("error symlinking %s -> %s in %s: %s", src, dst, g, err)
	}

	_, err = dest.Update(context.TODO(), storage.ObjectAttrsToUpdate{
		Metadata: map[string]string{"SymLink": src},
	})
	if err != nil {
		return fmt.Errorf("error updating symlink metadata %s -> %s in %s: %s", src, dst, g, err)
	}

	return g.applyACL(dest)
}

// HardLink uses symlink functionality as hard links do not exist on object stores.
func (g *PublishedStorage) HardLink(src string, dst string) error {
	if g.debug {
		log.Debug().Msgf("GCS: HardLink %s -> %s", src, dst)
	}

	return g.SymLink(src, dst)
}

// FileExists returns true if path exists.
func (g *PublishedStorage) FileExists(path string) (bool, error) {
	_, err := g.objectHandle(path).Attrs(context.TODO())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}

		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// ReadLink returns symbolic link target from metadata.
func (g *PublishedStorage) ReadLink(path string) (string, error) {
	attrs, err := g.objectHandle(path).Attrs(context.TODO())
	if err != nil {
		return "", err
	}

	return attrs.Metadata["SymLink"], nil
}
