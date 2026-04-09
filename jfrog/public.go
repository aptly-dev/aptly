package jfrog

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	aptly_utils "github.com/aptly-dev/aptly/utils"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/pkg/errors"
)

// PublishedStorage represents published repository on JFrog Artifactory
type PublishedStorage struct {
	manager        artifactory.ArtifactoryServicesManager
	repository     string
	prefix         string
	plusWorkaround bool
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorageRaw creates jfrog PublishedStorage from raw connection specs
func NewPublishedStorageRaw(
	repository, url, user, password, apiKey, accessToken, prefix string,
	plusWorkaround, debug bool,
) (*PublishedStorage, error) {

	artDetails := auth.NewArtifactoryDetails()
	artDetails.SetUrl(url)
	if user != "" && password != "" {
	    artDetails.SetUser(user)
	    artDetails.SetPassword(password)
	} else if apiKey != "" {
	    artDetails.SetApiKey(apiKey)
	} else if accessToken != "" {
		artDetails.SetAccessToken(accessToken)
    }

	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(artDetails).
		SetDryRun(false).
		Build()
	
	if err != nil {
		return nil, errors.Wrap(err, "error building jfrog client config")
	}

	manager, err := artifactory.New(serviceConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error creating jfrog manager")
	}

	return &PublishedStorage{
		manager:        manager,
		repository:     repository,
		prefix:         prefix,
		plusWorkaround: plusWorkaround,
	}, nil
}

// NewPublishedStorage creates published storage from aptly configuration struct
func NewPublishedStorage(
	account string, root aptly_utils.JFrogPublishRoot,
) (*PublishedStorage, error) {
	return NewPublishedStorageRaw(
		root.Repository, root.Url, root.User, root.Password, root.ApiKey, root.AccessToken,
		root.Prefix, root.PlusWorkaround, root.Debug)
}

func (storage *PublishedStorage) String() string {
	return fmt.Sprintf("jfrog:%s:%s", storage.repository, storage.prefix)
}

func (storage *PublishedStorage) MkDir(path string) error {
	return nil
}

func (storage *PublishedStorage) PutFile(path string, sourceFilename string) error {
    targetPath := filepath.Join(storage.repository, storage.prefix, path)
	if storage.plusWorkaround {
		targetPath = strings.Replace(targetPath, "+", "%2B", -1)
	}
    
	params := services.NewUploadParams()
	params.Pattern = sourceFilename
	params.Target = targetPath
	params.Flat = true

	_, _, err := storage.manager.UploadFiles(artifactory.UploadServiceOptions{}, params)
	return err
}

func (storage *PublishedStorage) Remove(path string) error {
	targetPath := filepath.Join(storage.repository, storage.prefix, path)
	if storage.plusWorkaround {
		targetPath = strings.Replace(targetPath, "+", "%2B", -1)
	}
    
	deleteParams := services.NewDeleteParams()
	deleteParams.SetPattern(targetPath)

	res, err := storage.manager.GetPathsToDelete(deleteParams)
	if err != nil {
		return err
	}
	defer res.Close()
	_, err = storage.manager.DeleteFiles(res)
	return err
}

func (storage *PublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	return storage.Remove(path)
}

func (storage *PublishedStorage) LinkFromPool(publishedPrefix, publishedRelPath, fileName string, sourcePool aptly.PackagePool, sourcePath string, sourceMD5 aptly_utils.ChecksumInfo, force bool) error {
	return storage.PutFile(filepath.Join(publishedPrefix, publishedRelPath, fileName), sourcePath)
}

func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
    searchPattern := filepath.Join(storage.repository, storage.prefix, prefix, "*")
	params := services.NewSearchParams()
	params.Pattern = searchPattern

	reader, err := storage.manager.SearchFiles(params)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
    
	var paths []string

	for element := new(utils.ResultItem); reader.NextRecord(element) == nil; element = new(utils.ResultItem) {
		path := element.Path + "/" + element.Name
		relPath := strings.TrimPrefix(path, storage.repository+"/"+storage.prefix+"/")
		if storage.plusWorkaround {
			relPath = strings.Replace(relPath, "%2B", "+", -1)
		}
		paths = append(paths, relPath)
	}
	
	return paths, nil
}

func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	oldTarget := filepath.Join(storage.repository, storage.prefix, oldName)
	newTarget := filepath.Join(storage.repository, storage.prefix, newName)
	
	if storage.plusWorkaround {
		oldTarget = strings.Replace(oldTarget, "+", "%2B", -1)
		newTarget = strings.Replace(newTarget, "+", "%2B", -1)
	}
    
	params := services.NewMoveCopyParams()
	params.Pattern = oldTarget
	params.Target = newTarget
	params.Flat = true

	_, _, err := storage.manager.Move(params)
	return err
}

func (storage *PublishedStorage) SymLink(src string, dst string) error {
	oldTarget := filepath.Join(storage.repository, storage.prefix, src)
	newTarget := filepath.Join(storage.repository, storage.prefix, dst)
	
	if storage.plusWorkaround {
		oldTarget = strings.Replace(oldTarget, "+", "%2B", -1)
		newTarget = strings.Replace(newTarget, "+", "%2B", -1)
	}
    
	params := services.NewMoveCopyParams()
	params.Pattern = oldTarget
	params.Target = newTarget
	params.Flat = true
	
	props := utils.NewProperties()
	props.AddProperty("SymLink", src)
	params.SetTargetProps(props)

	_, _, err := storage.manager.Copy(params)
	return err
}

func (storage *PublishedStorage) HardLink(src string, dst string) error {
	return storage.SymLink(src, dst)
}

func (storage *PublishedStorage) FileExists(path string) (bool, error) {
    targetPath := filepath.Join(storage.repository, storage.prefix, path)
	if storage.plusWorkaround {
		targetPath = strings.Replace(targetPath, "+", "%2B", -1)
	}
	
	params := services.NewSearchParams()
	params.Pattern = targetPath

	reader, err := storage.manager.SearchFiles(params)
	if err != nil {
		return false, err
	}
	defer reader.Close()
	
	length, err := reader.Length()
	isEmpty := length == 0
	return !isEmpty, err
}

func (storage *PublishedStorage) ReadLink(path string) (string, error) {
    targetPath := filepath.Join(storage.repository, storage.prefix, path)
	if storage.plusWorkaround {
		targetPath = strings.Replace(targetPath, "+", "%2B", -1)
	}
	
	props, err := storage.manager.GetItemProps(targetPath)
	if err != nil {
		return "", nil
	}
	
	for k, v := range props.Properties {
	    if k == "SymLink" && len(v) > 0 {
	        return v[0], nil
	    }
	}
	
	return "", nil
}
