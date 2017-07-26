package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/pgp"
	"github.com/smira/aptly/utils"
)

// SigningOptions is a shared between publish API GPG options structure
type SigningOptions struct {
	Skip           bool
	Batch          bool
	GpgKey         string
	Keyring        string
	SecretKeyring  string
	Passphrase     string
	PassphraseFile string
}

func getSigner(options *SigningOptions) (pgp.Signer, error) {
	if options.Skip {
		return nil, nil
	}

	signer := context.GetSigner()
	signer.SetKey(options.GpgKey)
	signer.SetKeyRing(options.Keyring, options.SecretKeyring)
	signer.SetPassphrase(options.Passphrase, options.PassphraseFile)
	signer.SetBatch(options.Batch)

	err := signer.Init()
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// Replace '_' with '/' and double '__' with single '_'
func parseEscapedPath(path string) string {
	result := strings.Replace(strings.Replace(path, "_", "/", -1), "//", "_", -1)
	if result == "" {
		result = "."
	}
	return result
}

// GET /publish
func apiPublishList(c *gin.Context) {
	localCollection := context.CollectionFactory().LocalRepoCollection()
	localCollection.RLock()
	defer localCollection.RUnlock()

	snapshotCollection := context.CollectionFactory().SnapshotCollection()
	snapshotCollection.RLock()
	defer snapshotCollection.RUnlock()

	collection := context.CollectionFactory().PublishedRepoCollection()
	collection.RLock()
	defer collection.RUnlock()

	result := make([]*deb.PublishedRepo, 0, collection.Len())

	err := collection.ForEach(func(repo *deb.PublishedRepo) error {
		err := collection.LoadComplete(repo, context.CollectionFactory())
		if err != nil {
			return err
		}

		result = append(result, repo)

		return nil
	})

	if err != nil {
		c.Fail(500, err)
		return
	}

	c.JSON(200, result)
}

// POST /publish/:prefix
func apiPublishRepoOrSnapshot(c *gin.Context) {
	param := parseEscapedPath(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)

	var b struct {
		SourceKind string `binding:"required"`
		Sources    []struct {
			Component string
			Name      string `binding:"required"`
		} `binding:"required"`
		Distribution         string
		Label                string
		Origin               string
		NotAutomatic         string
		ButAutomaticUpgrades string
		ForceOverwrite       bool
		SkipContents         *bool
		Architectures        []string
		Signing              SigningOptions
	}

	if !c.Bind(&b) {
		return
	}

	signer, err := getSigner(&b.Signing)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	if len(b.Sources) == 0 {
		c.Fail(400, fmt.Errorf("unable to publish: soures are empty"))
		return
	}

	var components []string
	var sources []interface{}

	if b.SourceKind == "snapshot" {
		var snapshot *deb.Snapshot

		snapshotCollection := context.CollectionFactory().SnapshotCollection()
		snapshotCollection.RLock()
		defer snapshotCollection.RUnlock()

		for _, source := range b.Sources {
			components = append(components, source.Component)

			snapshot, err = snapshotCollection.ByName(source.Name)
			if err != nil {
				c.Fail(404, fmt.Errorf("unable to publish: %s", err))
				return
			}

			err = snapshotCollection.LoadComplete(snapshot)
			if err != nil {
				c.Fail(500, fmt.Errorf("unable to publish: %s", err))
				return
			}

			sources = append(sources, snapshot)
		}
	} else if b.SourceKind == deb.SourceLocalRepo {
		var localRepo *deb.LocalRepo

		localCollection := context.CollectionFactory().LocalRepoCollection()
		localCollection.RLock()
		defer localCollection.RUnlock()

		for _, source := range b.Sources {
			components = append(components, source.Component)

			localRepo, err = localCollection.ByName(source.Name)
			if err != nil {
				c.Fail(404, fmt.Errorf("unable to publish: %s", err))
				return
			}

			err = localCollection.LoadComplete(localRepo)
			if err != nil {
				c.Fail(500, fmt.Errorf("unable to publish: %s", err))
			}

			sources = append(sources, localRepo)
		}
	} else {
		c.Fail(400, fmt.Errorf("unknown SourceKind"))
		return
	}

	collection := context.CollectionFactory().PublishedRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	published, err := deb.NewPublishedRepo(storage, prefix, b.Distribution, b.Architectures, components, sources, context.CollectionFactory())
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to publish: %s", err))
		return
	}
	if b.Origin != "" {
		published.Origin = b.Origin
	}
	if b.NotAutomatic != "" {
		published.NotAutomatic = b.NotAutomatic
	}
	if b.ButAutomaticUpgrades != "" {
		published.ButAutomaticUpgrades = b.ButAutomaticUpgrades
	}
	published.Label = b.Label

	published.SkipContents = context.Config().SkipContentsPublishing
	if b.SkipContents != nil {
		published.SkipContents = *b.SkipContents
	}

	duplicate := collection.CheckDuplicate(published)
	if duplicate != nil {
		context.CollectionFactory().PublishedRepoCollection().LoadComplete(duplicate, context.CollectionFactory())
		c.Fail(400, fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate))
		return
	}

	err = published.Publish(context.PackagePool(), context, context.CollectionFactory(), signer, nil, b.ForceOverwrite)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to publish: %s", err))
		return
	}

	err = collection.Add(published)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to save to DB: %s", err))
		return
	}

	c.JSON(201, published)
}

// PUT /publish/:prefix/:distribution
func apiPublishUpdateSwitch(c *gin.Context) {
	param := parseEscapedPath(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := c.Params.ByName("distribution")

	var b struct {
		ForceOverwrite bool
		Signing        SigningOptions
		SkipContents   *bool
		Snapshots      []struct {
			Component string `binding:"required"`
			Name      string `binding:"required"`
		}
	}

	if !c.Bind(&b) {
		return
	}

	signer, err := getSigner(&b.Signing)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	// published.LoadComplete would touch local repo collection
	localRepoCollection := context.CollectionFactory().LocalRepoCollection()
	localRepoCollection.RLock()
	defer localRepoCollection.RUnlock()

	snapshotCollection := context.CollectionFactory().SnapshotCollection()
	snapshotCollection.RLock()
	defer snapshotCollection.RUnlock()

	collection := context.CollectionFactory().PublishedRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		c.Fail(404, fmt.Errorf("unable to update: %s", err))
		return
	}
	err = collection.LoadComplete(published, context.CollectionFactory())
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to update: %s", err))
		return
	}

	var updatedComponents []string

	if published.SourceKind == deb.SourceLocalRepo {
		if len(b.Snapshots) > 0 {
			c.Fail(400, fmt.Errorf("snapshots shouldn't be given when updating local repo"))
			return
		}
		updatedComponents = published.Components()
		for _, component := range updatedComponents {
			published.UpdateLocalRepo(component)
		}
	} else if published.SourceKind == "snapshot" {
		publishedComponents := published.Components()
		for _, snapshotInfo := range b.Snapshots {
			if !utils.StrSliceHasItem(publishedComponents, snapshotInfo.Component) {
				c.Fail(404, fmt.Errorf("component %s is not in published repository", snapshotInfo.Component))
				return
			}

			snapshot, err2 := snapshotCollection.ByName(snapshotInfo.Name)
			if err != nil {
				c.Fail(404, err2)
				return
			}

			err2 = snapshotCollection.LoadComplete(snapshot)
			if err2 != nil {
				c.Fail(500, err2)
				return
			}

			published.UpdateSnapshot(snapshotInfo.Component, snapshot)
			updatedComponents = append(updatedComponents, snapshotInfo.Component)
		}
	} else {
		c.Fail(500, fmt.Errorf("unknown published repository type"))
		return
	}

	if b.SkipContents != nil {
		published.SkipContents = *b.SkipContents
	}

	err = published.Publish(context.PackagePool(), context, context.CollectionFactory(), signer, nil, b.ForceOverwrite)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to update: %s", err))
		return
	}

	err = collection.Update(published)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to save to DB: %s", err))
		return
	}

	err = collection.CleanupPrefixComponentFiles(published.Prefix, updatedComponents,
		context.GetPublishedStorage(storage), context.CollectionFactory(), nil)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to update: %s", err))
		return
	}

	c.JSON(200, published)
}

// DELETE /publish/:prefix/:distribution
func apiPublishDrop(c *gin.Context) {
	force := c.Request.URL.Query().Get("force") == "1"

	param := parseEscapedPath(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := c.Params.ByName("distribution")

	// published.LoadComplete would touch local repo collection
	localRepoCollection := context.CollectionFactory().LocalRepoCollection()
	localRepoCollection.RLock()
	defer localRepoCollection.RUnlock()

	collection := context.CollectionFactory().PublishedRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	err := collection.Remove(context, storage, prefix, distribution,
		context.CollectionFactory(), context.Progress(), force)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to drop: %s", err))
		return
	}

	c.JSON(200, gin.H{})
}
