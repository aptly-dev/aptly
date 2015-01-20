package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/utils"
	"strings"
)

type SigningOptions struct {
	Skip           bool
	Batch          bool
	GpgKey         string
	Keyring        string
	SecretKeyring  string
	Passphrase     string
	PassphraseFile string
}

func getSigner(options *SigningOptions) (utils.Signer, error) {
	if options.Skip {
		return nil, nil
	}

	signer := &utils.GpgSigner{}
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
	return strings.Replace(strings.Replace(path, "__", "_", -1), "_", "/", -1)
}

// GET /publish
func apiPublishList(c *gin.Context) {
	c.JSON(400, gin.H{})
}

// POST /publish/:prefix/repos | /publish/:prefix/snapshots
func apiPublishRepoOrSnapshot(c *gin.Context) {
	param := parseEscapedPath(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)

	var b struct {
		Sources []struct {
			Component string
			Name      string `binding:"required"`
		} `binding:"required"`
		Distribution   string
		Label          string
		Origin         string
		ForceOverwrite bool
		Signing        SigningOptions
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

	if strings.HasSuffix(c.Request.URL.Path, "/snapshots") {
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
	} else if strings.HasSuffix(c.Request.URL.Path, "/repos") {
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
		panic("unknown command")
	}

	collection := context.CollectionFactory().PublishedRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	published, err := deb.NewPublishedRepo(storage, prefix, b.Distribution, context.ArchitecturesList(), components, sources, context.CollectionFactory())
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to publish: %s", err))
		return
	}
	published.Origin = b.Origin
	published.Label = b.Label

	duplicate := collection.CheckDuplicate(published)
	if duplicate != nil {
		context.CollectionFactory().PublishedRepoCollection().LoadComplete(duplicate, context.CollectionFactory())
		c.Fail(400, fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate))
		return
	}

	err = published.Publish(context.PackagePool(), context, context.CollectionFactory(), signer, context.Progress(), b.ForceOverwrite)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to publish: %s", err))
		return
	}

	err = collection.Add(published)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to save to DB: %s", err))
	}

	c.JSON(200, published)
}

// PUT /publish/:prefix/:distribution
func apiPublishUpdateSwitch(c *gin.Context) {
	param := parseEscapedPath(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := c.Params.ByName("distribution")

	var b struct {
		ForceOverwrite bool
		Signing        SigningOptions
	}

	if !c.Bind(&b) {
		return
	}

	signer, err := getSigner(&b.Signing)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	collection := context.CollectionFactory().PublishedRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to update: %s", err))
		return
	}
	if published.SourceKind != "local" {
		c.Fail(500, fmt.Errorf("unable to update: not a local repository"))
		return
	}

	err = collection.LoadComplete(published, context.CollectionFactory())
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to update: %s", err))
		return
	}

	components := published.Components()
	for _, component := range components {
		published.UpdateLocalRepo(component)
	}

	err = published.Publish(context.PackagePool(), context, context.CollectionFactory(), signer, context.Progress(), b.ForceOverwrite)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to update: %s", err))
	}

	err = collection.Update(published)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to save to DB: %s", err))
	}

	err = collection.CleanupPrefixComponentFiles(published.Prefix, components,
		context.GetPublishedStorage(storage), context.CollectionFactory(), context.Progress())
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to update: %s", err))
	}

	c.JSON(200, published)
}

// DELETE /publish/:prefix/:distribution
func apiPublishDrop(c *gin.Context) {
	c.JSON(400, gin.H{})
}
