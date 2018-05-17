package api

import (
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/task"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
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
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	result := make([]*deb.PublishedRepo, 0, collection.Len())

	err := collection.ForEach(func(repo *deb.PublishedRepo) error {
		err := collection.LoadComplete(repo, collectionFactory)
		if err != nil {
			return err
		}

		result = append(result, repo)

		return nil
	})

	if err != nil {
		c.AbortWithError(500, err)
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
		AcquireByHash        *bool
	}

	if c.Bind(&b) != nil {
		return
	}

	signer, err := getSigner(&b.Signing)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	if len(b.Sources) == 0 {
		c.AbortWithError(400, fmt.Errorf("unable to publish: soures are empty"))
		return
	}

	var components []string
	var names []string
	var sources []interface{}
	var resources []string
	collectionFactory := context.NewCollectionFactory()

	if b.SourceKind == "snapshot" {
		var snapshot *deb.Snapshot

		snapshotCollection := collectionFactory.SnapshotCollection()

		for _, source := range b.Sources {
			components = append(components, source.Component)
			names = append(names, source.Name)

			snapshot, err = snapshotCollection.ByName(source.Name)
			if err != nil {
				c.AbortWithError(404, fmt.Errorf("unable to publish: %s", err))
				return
			}

			resources = append(resources, string(snapshot.ResourceKey()))
			err = snapshotCollection.LoadComplete(snapshot)
			if err != nil {
				c.AbortWithError(500, fmt.Errorf("unable to publish: %s", err))
				return
			}

			sources = append(sources, snapshot)
		}
	} else if b.SourceKind == deb.SourceLocalRepo {
		var localRepo *deb.LocalRepo

		localCollection := collectionFactory.LocalRepoCollection()

		for _, source := range b.Sources {
			components = append(components, source.Component)
			names = append(names, source.Name)

			localRepo, err = localCollection.ByName(source.Name)
			if err != nil {
				c.AbortWithError(404, fmt.Errorf("unable to publish: %s", err))
				return
			}

			resources = append(resources, string(localRepo.Key()))
			err = localCollection.LoadComplete(localRepo)
			if err != nil {
				c.AbortWithError(500, fmt.Errorf("unable to publish: %s", err))
			}

			sources = append(sources, localRepo)
		}
	} else {
		c.AbortWithError(400, fmt.Errorf("unknown SourceKind"))
		return
	}

	published, err := deb.NewPublishedRepo(storage, prefix, b.Distribution, b.Architectures, components, sources, collectionFactory)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to publish: %s", err))
		return
	}

	resources = append(resources, string(published.Key()))
	collection := collectionFactory.PublishedRepoCollection()

	taskName := fmt.Sprintf("Publish %s: %s", b.SourceKind, strings.Join(names, ", "))
	task, conflictErr := runTaskInBackground(taskName, resources, func(out *task.Output, detail *task.Detail) error {
		taskDetail := task.PublishDetail{
			Detail: detail,
		}
		publishOutput := &task.PublishOutput{
			Output:        out,
			PublishDetail: taskDetail,
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

		if b.AcquireByHash != nil {
			published.AcquireByHash = *b.AcquireByHash
		}

		duplicate := collection.CheckDuplicate(published)
		if duplicate != nil {
			collectionFactory.PublishedRepoCollection().LoadComplete(duplicate, collectionFactory)
			return fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate)
		}

		err := published.Publish(context.PackagePool(), context, collectionFactory, signer, publishOutput, b.ForceOverwrite)
		if err != nil {
			return fmt.Errorf("unable to publish: %s", err)
		}

		err = collection.Add(published)
		if err != nil {
			return fmt.Errorf("unable to save to DB: %s", err)
		}

		return nil
	})

	if conflictErr != nil {
		c.AbortWithError(409, conflictErr)
		return
	}

	c.JSON(202, task)
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
		SkipCleanup    *bool
		Snapshots      []struct {
			Component string `binding:"required"`
			Name      string `binding:"required"`
		}
		AcquireByHash *bool
	}

	if c.Bind(&b) != nil {
		return
	}

	signer, err := getSigner(&b.Signing)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		c.AbortWithError(404, fmt.Errorf("unable to update: %s", err))
		return
	}
	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to update: %s", err))
		return
	}

	var updatedComponents []string
	var updatedSnapshots []string
	var resources []string

	if published.SourceKind == deb.SourceLocalRepo {
		if len(b.Snapshots) > 0 {
			c.AbortWithError(400, fmt.Errorf("snapshots shouldn't be given when updating local repo"))
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
				c.AbortWithError(404, fmt.Errorf("component %s is not in published repository", snapshotInfo.Component))
				return
			}

			snapshotCollection := collectionFactory.SnapshotCollection()
			snapshot, err2 := snapshotCollection.ByName(snapshotInfo.Name)
			if err2 != nil {
				c.AbortWithError(404, err2)
				return
			}

			err2 = snapshotCollection.LoadComplete(snapshot)
			if err2 != nil {
				c.AbortWithError(500, err2)
				return
			}

			published.UpdateSnapshot(snapshotInfo.Component, snapshot)
			updatedComponents = append(updatedComponents, snapshotInfo.Component)
			updatedSnapshots = append(updatedSnapshots, snapshot.Name)
		}
	} else {
		c.AbortWithError(500, fmt.Errorf("unknown published repository type"))
		return
	}

	if b.SkipContents != nil {
		published.SkipContents = *b.SkipContents
	}

	if b.AcquireByHash != nil {
		published.AcquireByHash = *b.AcquireByHash
	}

	resources = append(resources, string(published.Key()))
	taskName := fmt.Sprintf("Update published %s (%s): %s", published.SourceKind, strings.Join(updatedComponents, " "), strings.Join(updatedSnapshots, ", "))
	currTask, conflictErr := runTaskInBackground(taskName, resources, func(out *task.Output, detail *task.Detail) error {
		err := published.Publish(context.PackagePool(), context, collectionFactory, signer, out, b.ForceOverwrite)
		if err != nil {
			return fmt.Errorf("unable to update: %s", err)
		}

		err = collection.Update(published)
		if err != nil {
			return fmt.Errorf("unable to save to DB: %s", err)
		}

		if b.SkipCleanup == nil || !*b.SkipCleanup {
			err = collection.CleanupPrefixComponentFiles(published.Prefix, updatedComponents,
				context.GetPublishedStorage(storage), collectionFactory, out)
			if err != nil {
				return fmt.Errorf("unable to update: %s", err)
			}
		}

		return nil
	})

	if conflictErr != nil {
		c.AbortWithError(409, conflictErr)
		return
	}

	c.JSON(202, currTask)
}

// DELETE /publish/:prefix/:distribution
func apiPublishDrop(c *gin.Context) {
	force := c.Request.URL.Query().Get("force") == "1"
	skipCleanup := c.Request.URL.Query().Get("SkipCleanup") == "1"

	param := parseEscapedPath(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := c.Params.ByName("distribution")

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to drop: %s", err))
		return
	}

	resources := []string{string(published.Key())}

	taskName := fmt.Sprintf("Delete published %s (%s)", prefix, distribution)
	currTask, conflictErr := runTaskInBackground(taskName, resources, func(out *task.Output, detail *task.Detail) error {
		err := collection.Remove(context, storage, prefix, distribution,
			collectionFactory, out, force, skipCleanup)
		if err != nil {
			return fmt.Errorf("unable to drop: %s", err)
		}

		return nil
	})

	if conflictErr != nil {
		c.AbortWithError(409, conflictErr)
		return
	}

	c.JSON(202, currTask)
}
