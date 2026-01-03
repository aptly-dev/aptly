package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/deb"
	"github.com/gin-gonic/gin"
	"github.com/smira/flag"

	. "gopkg.in/check.v1"
)

// PublishedFileMissingSuite reproduces the exact bug where:
// - Package import succeeds
// - Metadata is updated (Packages.gz shows the package)
// - Publish reports success
// - BUT the .deb file is missing from the published pool directory
// - Result: apt-get returns 404 when trying to download the package
type PublishedFileMissingSuite struct {
	context    *ctx.AptlyContext
	flags      *flag.FlagSet
	configFile *os.File
	router     http.Handler
	tempDir    string
	poolPath   string
	publicPath string
}

var _ = Suite(&PublishedFileMissingSuite{})

func (s *PublishedFileMissingSuite) SetUpSuite(c *C) {
	aptly.Version = "publishedFileMissingTest"

	tempDir, err := os.MkdirTemp("", "aptly-published-missing-test")
	c.Assert(err, IsNil)
	s.tempDir = tempDir
	s.poolPath = filepath.Join(tempDir, "pool")
	s.publicPath = filepath.Join(tempDir, "public")

	file, err := os.CreateTemp("", "aptly-published-missing-config")
	c.Assert(err, IsNil)
	s.configFile = file

	config := gin.H{
		"rootDir":                    tempDir,
		"downloadDir":                filepath.Join(tempDir, "download"),
		"architectures":              []string{"amd64"},
		"dependencyFollowSuggests":   false,
		"dependencyFollowRecommends": false,
		"gpgDisableSign":             true,
		"gpgDisableVerify":           true,
		"gpgProvider":                "internal",
		"skipLegacyPool":             true,
		"enableMetricsEndpoint":      false,
	}

	jsonString, err := json.Marshal(config)
	c.Assert(err, IsNil)
	_, err = file.Write(jsonString)
	c.Assert(err, IsNil)

	flags := flag.NewFlagSet("publishedFileMissingTestFlags", flag.ContinueOnError)
	flags.Bool("no-lock", true, "disable database locking for test")
	flags.Int("db-open-attempts", 3, "dummy")
	flags.String("config", s.configFile.Name(), "config file")
	flags.String("architectures", "", "dummy")
	s.flags = flags

	context, err := ctx.NewContext(s.flags)
	c.Assert(err, IsNil)

	s.context = context
	s.router = Router(context)
}

func (s *PublishedFileMissingSuite) TearDownSuite(c *C) {
	if s.configFile != nil {
		_ = os.Remove(s.configFile.Name())
	}
	if s.context != nil {
		s.context.Shutdown()
	}
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
}

func (s *PublishedFileMissingSuite) SetUpTest(c *C) {
	collectionFactory := s.context.NewCollectionFactory()

	localRepoCollection := collectionFactory.LocalRepoCollection()
	_ = localRepoCollection.ForEach(func(repo *deb.LocalRepo) error {
		_ = localRepoCollection.Drop(repo)
		return nil
	})

	publishedCollection := collectionFactory.PublishedRepoCollection()
	_ = publishedCollection.ForEach(func(published *deb.PublishedRepo) error {
		_ = publishedCollection.Remove(s.context, published.Storage, published.Prefix,
			published.Distribution, collectionFactory, nil, true, true)
		return nil
	})
}

func (s *PublishedFileMissingSuite) TearDownTest(c *C) {
	s.SetUpTest(c)
}

func (s *PublishedFileMissingSuite) httpRequest(c *C, method string, url string, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	c.Assert(err, IsNil)
	req.Header.Add("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	return w
}

func (s *PublishedFileMissingSuite) createDebPackage(c *C, uploadID, packageName, version string) {
	uploadPath := s.context.UploadPath()
	uploadDir := filepath.Join(uploadPath, uploadID)
	err := os.MkdirAll(uploadDir, 0755)
	c.Assert(err, IsNil)

	tempDir, err := os.MkdirTemp("", "deb-build")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tempDir)

	debianDir := filepath.Join(tempDir, "DEBIAN")
	err = os.MkdirAll(debianDir, 0755)
	c.Assert(err, IsNil)

	controlContent := fmt.Sprintf(`Package: %s
Version: %s
Section: libs
Priority: optional
Architecture: amd64
Maintainer: Test <test@example.com>
Description: Test package
 Test package for published file missing bug.
`, packageName, version)

	err = os.WriteFile(filepath.Join(debianDir, "control"), []byte(controlContent), 0644)
	c.Assert(err, IsNil)

	usrDir := filepath.Join(tempDir, "usr", "lib")
	err = os.MkdirAll(usrDir, 0755)
	c.Assert(err, IsNil)
	err = os.WriteFile(filepath.Join(usrDir, "lib.so"), []byte("library"), 0644)
	c.Assert(err, IsNil)

	debFile := filepath.Join(uploadDir, fmt.Sprintf("%s_%s_amd64.deb", packageName, version))
	cmd := exec.Command("dpkg-deb", "--build", tempDir, debFile)
	err = cmd.Run()
	c.Assert(err, IsNil)
}

// TestPublishedFileGoMissing reproduces the exact production bug
func (s *PublishedFileMissingSuite) TestPublishedFileGoMissing(c *C) {
	c.Log("=== Reproducing: Package in metadata but 404 on download ===")

	// Create and publish a repository
	repoName := "test-repo"
	distribution := "bullseye"

	createBody, _ := json.Marshal(gin.H{
		"Name":                repoName,
		"DefaultDistribution": distribution,
		"DefaultComponent":    "main",
	})
	resp := s.httpRequest(c, "POST", "/api/repos", createBody)
	c.Assert(resp.Code, Equals, 201, Commentf("Failed to create repo: %s", resp.Body.String()))

	publishBody, _ := json.Marshal(gin.H{
		"SourceKind":    "local",
		"Distribution":  distribution,
		"Architectures": []string{"amd64"},
		"Sources": []gin.H{
			{"Component": "main", "Name": repoName},
		},
		"Signing": gin.H{"Skip": true},
	})
	resp = s.httpRequest(c, "POST", "/api/publish/hrt", publishBody)
	c.Assert(resp.Code, Equals, 201, Commentf("Failed to publish: %s", resp.Body.String()))

	// Create package
	packageName := "hrt-libblobbyclient1"
	version := "20250926.152427+hrtdeb11"
	uploadID := "test-upload-1"

	s.createDebPackage(c, uploadID, packageName, version)

	// Add package
	resp = s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/file/%s?noRemove=0", repoName, uploadID), nil)
	c.Assert(resp.Code, Equals, 200, Commentf("Failed to add package: %s", resp.Body.String()))

	// Update publish
	updateBody, _ := json.Marshal(gin.H{
		"Signing":        gin.H{"Skip": true},
		"ForceOverwrite": true,
	})
	resp = s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/hrt/%s", distribution), updateBody)
	c.Assert(resp.Code, Equals, 200, Commentf("Failed to update publish: %s", resp.Body.String()))

	// Now check if the file is actually accessible in the published location
	publishedStorage := s.context.GetPublishedStorage("")
	publicPath := publishedStorage.(aptly.FileSystemPublishedStorage).PublicPath()

	// Expected file path: hrt/pool/main/h/hrt-libblobbyclient1/hrt-libblobbyclient1_20250926.152427+hrtdeb11_amd64.deb
	expectedPath := filepath.Join(publicPath, "hrt", "pool", "main", "h", packageName,
		fmt.Sprintf("%s_%s_amd64.deb", packageName, version))

	c.Logf("Checking for published file at: %s", expectedPath)

	fileInfo, err := os.Stat(expectedPath)
	fileExists := err == nil

	c.Logf("File exists: %v", fileExists)
	if fileExists {
		c.Logf("File size: %d bytes", fileInfo.Size())
	}

	// Check metadata
	resp = s.httpRequest(c, "GET", fmt.Sprintf("/api/repos/%s/packages", repoName), nil)
	var packages []string
	err = json.Unmarshal(resp.Body.Bytes(), &packages)
	c.Assert(err, IsNil)
	c.Logf("Packages in metadata: %d", len(packages))

	// THE BUG: Metadata says package exists, but file is missing from published location
	if len(packages) > 0 && !fileExists {
		c.Logf("★★★ BUG REPRODUCED! ★★★")
		c.Logf("Metadata shows %d package(s) but file is missing at: %s", len(packages), expectedPath)
		c.Logf("This is exactly what causes: 404 Not Found [IP: 10.20.72.62 3142]")

		c.Fatal("BUG CONFIRMED: Package in metadata but missing from published directory!")
	}

	c.Assert(fileExists, Equals, true, Commentf(
		"Published file should exist at %s when package is in metadata", expectedPath))
}

// TestConcurrentPublishRace tries to trigger the race with concurrent publishes
func (s *PublishedFileMissingSuite) TestConcurrentPublishRace(c *C) {
	c.Log("=== Testing concurrent publish race condition ===")

	const numIterations = 4

	for iteration := 0; iteration < numIterations; iteration++ {
		c.Logf("--- Iteration %d/%d ---", iteration+1, numIterations)

		// Create repo
		repoName := fmt.Sprintf("race-repo-%d", iteration)
		distribution := fmt.Sprintf("dist-%d", iteration)

		createBody, _ := json.Marshal(gin.H{
			"Name":                repoName,
			"DefaultDistribution": distribution,
			"DefaultComponent":    "main",
		})
		resp := s.httpRequest(c, "POST", "/api/repos", createBody)
		c.Assert(resp.Code, Equals, 201)

		publishBody, _ := json.Marshal(gin.H{
			"SourceKind":    "local",
			"Distribution":  distribution,
			"Architectures": []string{"amd64"},
			"Sources": []gin.H{
				{"Component": "main", "Name": repoName},
			},
			"Signing": gin.H{"Skip": true},
		})
		resp = s.httpRequest(c, "POST", "/api/publish/concurrent", publishBody)
		c.Assert(resp.Code, Equals, 201)

		// Create multiple packages
		var wg sync.WaitGroup
		numPackages := 5

		for i := 0; i < numPackages; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				packageName := fmt.Sprintf("pkg-%d-%d", iteration, idx)
				version := "1.0.0"
				uploadID := fmt.Sprintf("upload-%d-%d", iteration, idx)

				s.createDebPackage(c, uploadID, packageName, version)

				// Add package
				resp := s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/file/%s?noRemove=0", repoName, uploadID), nil)
				c.Logf("Package %d add: %d", idx, resp.Code)

				// Small delay
				time.Sleep(time.Duration(5+idx*2) * time.Millisecond)

				// Publish
				updateBody, _ := json.Marshal(gin.H{
					"Signing":        gin.H{"Skip": true},
					"ForceOverwrite": true,
				})
				resp = s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/concurrent/%s", distribution), updateBody)
				c.Logf("Publish %d: %d", idx, resp.Code)
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond)

		// Check all packages
		resp = s.httpRequest(c, "GET", fmt.Sprintf("/api/repos/%s/packages", repoName), nil)
		var packages []string
		err := json.Unmarshal(resp.Body.Bytes(), &packages)
		c.Assert(err, IsNil)

		// Check published files
		publishedStorage := s.context.GetPublishedStorage("")
		publicPath := publishedStorage.(aptly.FileSystemPublishedStorage).PublicPath()

		missingFiles := []string{}
		for i := 0; i < numPackages; i++ {
			packageName := fmt.Sprintf("pkg-%d-%d", iteration, i)
			version := "1.0.0"

			// Calculate pool path
			poolSubdir := string(packageName[0])
			expectedPath := filepath.Join(publicPath, "concurrent", "pool", "main", poolSubdir, packageName,
				fmt.Sprintf("%s_%s_amd64.deb", packageName, version))

			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				missingFiles = append(missingFiles, expectedPath)
			}
		}

		if len(missingFiles) > 0 {
			c.Logf("★★★ BUG DETECTED in iteration %d/%d! ★★★", iteration+1, numIterations)
			c.Logf("Metadata shows %d packages, but %d files are MISSING:", len(packages), len(missingFiles))
			for i, f := range missingFiles {
				c.Logf("  [iter %d] File MISSING %d/%d: %s", iteration+1, i+1, len(missingFiles), f)
			}

			c.Fatalf("BUG REPRODUCED in iteration %d/%d: %d published files missing", iteration+1, numIterations, len(missingFiles))
		} else {
			c.Logf("[iter %d/%d] All %d files present - OK", iteration+1, numIterations, numPackages)
		}
	}

	c.Logf("All %d iterations passed - bug not reproduced with current timing", numIterations)
}

// TestIdenticalPackageRace tests the specific case of identical SHA256 packages
func (s *PublishedFileMissingSuite) TestIdenticalPackageRace(c *C) {
	c.Log("=== AGGRESSIVE test: identical package (same SHA256) race ===")

	const numIterations = 4
	packageName := "shared-package"

	for iter := 0; iter < numIterations; iter++ {
		c.Logf("Iteration %d/%d", iter+1, numIterations)

		// Create two repos that will get the SAME package (unique per iteration)
		repos := []string{fmt.Sprintf("identical-a-%d", iter), fmt.Sprintf("identical-b-%d", iter)}
		dists := []string{fmt.Sprintf("dist-a-%d", iter), fmt.Sprintf("dist-b-%d", iter)}

		for i := range repos {
			createBody, _ := json.Marshal(gin.H{
				"Name":                repos[i],
				"DefaultDistribution": dists[i],
				"DefaultComponent":    "main",
			})
			resp := s.httpRequest(c, "POST", "/api/repos", createBody)
			c.Assert(resp.Code, Equals, 201)

			publishBody, _ := json.Marshal(gin.H{
				"SourceKind":    "local",
				"Distribution":  dists[i],
				"Architectures": []string{"amd64"},
				"Sources": []gin.H{
					{"Component": "main", "Name": repos[i]},
				},
				"Signing": gin.H{"Skip": true},
				"SkipBz2": true,
			})
			resp = s.httpRequest(c, "POST", "/api/publish/identical", publishBody)
			c.Assert(resp.Code, Equals, 201)
		}

		// Create IDENTICAL package file with UNIQUE VERSION per iteration
		version := fmt.Sprintf("1.0.%d", iter)
		uploadID1 := fmt.Sprintf("identical-upload-1-%d", iter)
		uploadID2 := fmt.Sprintf("identical-upload-2-%d", iter)

		s.createDebPackage(c, uploadID1, packageName, version)

		// Copy to second upload (same SHA256)
		uploadPath := s.context.UploadPath()
		src := filepath.Join(uploadPath, uploadID1, fmt.Sprintf("%s_%s_amd64.deb", packageName, version))
		destDir := filepath.Join(uploadPath, uploadID2)
		err := os.MkdirAll(destDir, 0755)
		c.Assert(err, IsNil)
		dest := filepath.Join(destDir, fmt.Sprintf("%s_%s_amd64.deb", packageName, version))

		srcData, readErr := os.ReadFile(src)
		c.Assert(readErr, IsNil)
		err = os.WriteFile(dest, srcData, 0644)
		c.Assert(err, IsNil)

		// Race: add and publish both simultaneously
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			//time.Sleep(5 * time.Millisecond)
			c.Logf("[iter %d] Import A", iter)
			resp := s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/file/%s?noRemove=0", repos[0], uploadID1), nil)
			c.Logf("[iter %d] Import A complete: %d", iter, resp.Code)

			updateBody, _ := json.Marshal(gin.H{"Signing": gin.H{"Skip": true}, "ForceOverwrite": true, "SkipBz2": true})
			c.Logf("[iter %d] Publish A", iter)
			resp = s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/identical/%s", dists[0]), updateBody)
			c.Logf("[iter %d] Publish A complete: %d", iter, resp.Code)
		}()

		go func() {
			defer wg.Done()
			//time.Sleep(7 * time.Millisecond)
			c.Logf("[iter %d] Import B", iter)
			resp := s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/file/%s?noRemove=0", repos[1], uploadID2), nil)
			c.Logf("[iter %d] Import B complete: %d", iter, resp.Code)

			updateBody, _ := json.Marshal(gin.H{"Signing": gin.H{"Skip": true}, "ForceOverwrite": true, "SkipBz2": true})
			c.Logf("[iter %d] Publish B", iter)
			resp = s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/identical/%s", dists[1]), updateBody)
			c.Logf("[iter %d] Publish B complete: %d", iter, resp.Code)
		}()

		//go func() {
			//defer wg.Done()
			//time.Sleep(15 * time.Millisecond)
			//updateBody, _ := json.Marshal(gin.H{"Signing": gin.H{"Skip": true}, "ForceOverwrite": true, "SkipBz2": true})
			//c.Logf("[iter %d] Publish A", iter)
			//resp := s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/identical/%s", dists[0]), updateBody)
			//c.Logf("[iter %d] Publish A complete: %d", iter, resp.Code)
		//}()

		//go func() {
			//defer wg.Done()
			//time.Sleep(18 * time.Millisecond)
			//updateBody, _ := json.Marshal(gin.H{"Signing": gin.H{"Skip": true}, "ForceOverwrite": true, "SkipBz2": true})
			//c.Logf("[iter %d] Publish B", iter)
			//resp := s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/identical/%s", dists[1]), updateBody)
			//c.Logf("[iter %d] Publish B complete: %d", iter, resp.Code)
		//}()

		wg.Wait()
		time.Sleep(200 * time.Millisecond)
		c.Logf("[iter %d] All operations complete", iter)

		// Check the shared pool location
		publishedStorage := s.context.GetPublishedStorage("")
		publicPath := publishedStorage.(aptly.FileSystemPublishedStorage).PublicPath()

		poolSubdir := string(packageName[0])
		sharedPoolPath := filepath.Join(publicPath, "identical", "pool", "main", poolSubdir, packageName,
			fmt.Sprintf("%s_%s_amd64.deb", packageName, version))

		fileInfo, err := os.Stat(sharedPoolPath)
		fileExists := err == nil

		if fileExists {
			c.Logf("[iter %d] File EXISTS at %s (size: %d)", iter, sharedPoolPath, fileInfo.Size())
		} else {
			c.Logf("[iter %d] File MISSING at %s (error: %v)", iter, sharedPoolPath, err)
		}

		// Check metadata
		var packagesA, packagesB []string
		resp := s.httpRequest(c, "GET", fmt.Sprintf("/api/repos/%s/packages", repos[0]), nil)
		err = json.Unmarshal(resp.Body.Bytes(), &packagesA)
		c.Assert(err, IsNil)
		resp = s.httpRequest(c, "GET", fmt.Sprintf("/api/repos/%s/packages", repos[1]), nil)
		err = json.Unmarshal(resp.Body.Bytes(), &packagesB)
		c.Assert(err, IsNil)

		c.Logf("[iter %d] Packages in metadata: A=%d, B=%d", iter, len(packagesA), len(packagesB))

		// THE BUG: Both repos show packages in metadata, but the shared pool file is missing
		if (len(packagesA) > 0 || len(packagesB) > 0) && !fileExists {
			c.Logf("★★★ BUG REPRODUCED in iteration %d! ★★★", iter+1)
			c.Logf("Packages in metadata A: %d, B: %d", len(packagesA), len(packagesB))
			c.Logf("Shared pool file exists: %v", fileExists)
			c.Logf("Pool path: %s", sharedPoolPath)

			// List what files ARE in the pool directory
			poolDir := filepath.Dir(sharedPoolPath)
			if entries, err := os.ReadDir(poolDir); err == nil {
				c.Logf("Files in pool directory %s:", poolDir)
				for _, entry := range entries {
					c.Logf("  - %s", entry.Name())
				}
			}

			c.Fatalf("Metadata shows packages but shared pool file is missing (iteration %d)", iter+1)
		}
	}

	c.Logf("All %d iterations passed - bug not reproduced", numIterations)
}

// TestConcurrentSnapshotPublishToSamePrefix reproduces the EXACT production bug:
// Multiple snapshots are published concurrently to the SAME prefix but different distributions.
// Example from production logs:
//   - trixie-pgdg published to "external/postgres-auto/trixie"
//   - bullseye-pgdg published to "external/postgres-auto/bullseye"
// Both share the same pool directory, causing cleanup race conditions.
func (s *PublishedFileMissingSuite) TestConcurrentSnapshotPublishToSamePrefix(c *C) {
	const numIterations = 4

	for iter := 0; iter < numIterations; iter++ {
		c.Logf("--- Iteration %d/%d ---", iter+1, numIterations)

		// Create two repos with different packages (simulating trixie-pgdg and bullseye-pgdg)
		repoTrixie := fmt.Sprintf("trixie-pgdg-%d", iter)
		repoBullseye := fmt.Sprintf("bullseye-pgdg-%d", iter)

		// Create trixie repo
		createBody, _ := json.Marshal(gin.H{
			"Name":                repoTrixie,
			"DefaultDistribution": "trixie",
			"DefaultComponent":    "main",
		})
		resp := s.httpRequest(c, "POST", "/api/repos", createBody)
		c.Assert(resp.Code, Equals, 201, Commentf("Failed to create trixie repo"))

		// Create bullseye repo
		createBody, _ = json.Marshal(gin.H{
			"Name":                repoBullseye,
			"DefaultDistribution": "bullseye",
			"DefaultComponent":    "main",
		})
		resp = s.httpRequest(c, "POST", "/api/repos", createBody)
		c.Assert(resp.Code, Equals, 201, Commentf("Failed to create bullseye repo"))

		// Add packages to both repos
		numPackages := 3

		// Add packages to trixie repo
		for i := 0; i < numPackages; i++ {
			packageName := fmt.Sprintf("postgresql-17-trixie-pkg%d", i)
			version := fmt.Sprintf("17.0.%d", iter)
			uploadID := fmt.Sprintf("trixie-upload-%d-%d", iter, i)

			s.createDebPackage(c, uploadID, packageName, version)
			resp = s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/file/%s?noRemove=0", repoTrixie, uploadID), nil)
			c.Assert(resp.Code, Equals, 200, Commentf("Failed to add package to trixie"))
		}

		// Add packages to bullseye repo
		for i := 0; i < numPackages; i++ {
			packageName := fmt.Sprintf("postgresql-17-bullseye-pkg%d", i)
			version := fmt.Sprintf("17.0.%d", iter)
			uploadID := fmt.Sprintf("bullseye-upload-%d-%d", iter, i)

			s.createDebPackage(c, uploadID, packageName, version)
			resp = s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/file/%s?noRemove=0", repoBullseye, uploadID), nil)
			c.Assert(resp.Code, Equals, 200, Commentf("Failed to add package to bullseye"))
		}

		// Create snapshots from both repos
		snapshotTrixie := fmt.Sprintf("%s-snap", repoTrixie)
		snapshotBullseye := fmt.Sprintf("%s-snap", repoBullseye)

		createSnapshotBody, _ := json.Marshal(gin.H{"Name": snapshotTrixie})
		resp = s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/snapshots", repoTrixie), createSnapshotBody)
		c.Assert(resp.Code, Equals, 201, Commentf("Failed to create trixie snapshot"))

		createSnapshotBody, _ = json.Marshal(gin.H{"Name": snapshotBullseye})
		resp = s.httpRequest(c, "POST", fmt.Sprintf("/api/repos/%s/snapshots", repoBullseye), createSnapshotBody)
		c.Assert(resp.Code, Equals, 201, Commentf("Failed to create bullseye snapshot"))

		// Publish both snapshots CONCURRENTLY to the SAME prefix
		// This mimics production where both are published to "external/postgres-auto"
		// Use the SAME prefix across all iterations to trigger the race more aggressively
		sharedPrefix := "postgres-auto"

		var wg sync.WaitGroup
		var trixiePublishCode, bullseyePublishCode int

		wg.Add(2)

		// Publish or update trixie snapshot
		go func() {
			defer wg.Done()

			var resp *httptest.ResponseRecorder
			if iter == 0 {
				// First iteration: CREATE
				publishBody, _ := json.Marshal(gin.H{
					"SourceKind":    "snapshot",
					"Distribution":  "trixie",
					"Architectures": []string{"amd64"},
					"Sources": []gin.H{
						{"Name": snapshotTrixie},
					},
					"Signing":        gin.H{"Skip": true},
					"SkipBz2":        true,
					"ForceOverwrite": true,
					"SkipCleanup":    false, // Force cleanup to run
				})
				resp = s.httpRequest(c, "POST", fmt.Sprintf("/api/publish/%s", sharedPrefix), publishBody)
			} else {
				// Subsequent iterations: UPDATE (this is what happens in production)
				updateBody, _ := json.Marshal(gin.H{
					"Snapshots": []gin.H{
						{"Component": "main", "Name": snapshotTrixie},
					},
					"Signing":        gin.H{"Skip": true},
					"SkipBz2":        true,
					"ForceOverwrite": true,
					"SkipCleanup":    false,
				})
				resp = s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/%s/trixie", sharedPrefix), updateBody)
			}
			trixiePublishCode = resp.Code
			c.Logf("[iter %d] Trixie publish/update completed: %d", iter, resp.Code)
		}()

		// Publish or update bullseye snapshot
		go func() {
			defer wg.Done()

			var resp *httptest.ResponseRecorder
			if iter == 0 {
				// First iteration: CREATE
				publishBody, _ := json.Marshal(gin.H{
					"SourceKind":    "snapshot",
					"Distribution":  "bullseye",
					"Architectures": []string{"amd64"},
					"Sources": []gin.H{
						{"Name": snapshotBullseye},
					},
					"Signing":        gin.H{"Skip": true},
					"SkipBz2":        true,
					"ForceOverwrite": true,
					"SkipCleanup":    false,
				})
				resp = s.httpRequest(c, "POST", fmt.Sprintf("/api/publish/%s", sharedPrefix), publishBody)
			} else {
				// Subsequent iterations: UPDATE
				updateBody, _ := json.Marshal(gin.H{
					"Snapshots": []gin.H{
						{"Component": "main", "Name": snapshotBullseye},
					},
					"Signing":        gin.H{"Skip": true},
					"SkipBz2":        true,
					"ForceOverwrite": true,
					"SkipCleanup":    false,
				})
				resp = s.httpRequest(c, "PUT", fmt.Sprintf("/api/publish/%s/bullseye", sharedPrefix), updateBody)
			}
			bullseyePublishCode = resp.Code
			c.Logf("[iter %d] Bullseye publish/update completed: %d", iter, resp.Code)
		}()

		wg.Wait()
		time.Sleep(50 * time.Millisecond)

		// Verify publishes succeeded (201 for create, 200 for update)
		expectedCode := 201
		if iter > 0 {
			expectedCode = 200
		}
		c.Assert(trixiePublishCode, Equals, expectedCode, Commentf("Trixie publish/update should succeed"))
		c.Assert(bullseyePublishCode, Equals, expectedCode, Commentf("Bullseye publish/update should succeed"))

		// Verify ALL package files exist in the published pool
		publishedStorage := s.context.GetPublishedStorage("")
		publicPath := publishedStorage.(aptly.FileSystemPublishedStorage).PublicPath()

		missingFiles := []string{}
		expectedFiles := []string{}

		// Check trixie packages
		for i := 0; i < numPackages; i++ {
			packageName := fmt.Sprintf("postgresql-17-trixie-pkg%d", i)
			version := fmt.Sprintf("17.0.%d", iter)

			poolSubdir := string(packageName[0])
			expectedPath := filepath.Join(publicPath, sharedPrefix, "pool", "main", poolSubdir, packageName,
				fmt.Sprintf("%s_%s_amd64.deb", packageName, version))

			expectedFiles = append(expectedFiles, expectedPath)
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				missingFiles = append(missingFiles, fmt.Sprintf("TRIXIE: %s", filepath.Base(expectedPath)))
			}
		}

		// Check bullseye packages
		for i := 0; i < numPackages; i++ {
			packageName := fmt.Sprintf("postgresql-17-bullseye-pkg%d", i)
			version := fmt.Sprintf("17.0.%d", iter)

			poolSubdir := string(packageName[0])
			expectedPath := filepath.Join(publicPath, sharedPrefix, "pool", "main", poolSubdir, packageName,
				fmt.Sprintf("%s_%s_amd64.deb", packageName, version))

			expectedFiles = append(expectedFiles, expectedPath)
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				missingFiles = append(missingFiles, fmt.Sprintf("BULLSEYE: %s", filepath.Base(expectedPath)))
			}
		}

		// BUG: Files from one distribution are deleted by the other's cleanup
		if len(missingFiles) > 0 {
			c.Logf("★★★ BUG REPRODUCED in iteration %d/%d! ★★★", iter+1, numIterations)
			c.Logf("Both publishes to prefix '%s' succeeded, but %d files are MISSING:", sharedPrefix, len(missingFiles))
			for i, f := range missingFiles {
				c.Logf("  Missing file %d/%d: %s", i+1, len(missingFiles), f)
			}

			c.Logf("\nThis reproduces the exact production bug where:")
			c.Logf("  1. Mirror updates complete successfully")
			c.Logf("  2. Snapshots are created")
			c.Logf("  3. Both snapshots publish to same prefix (different distributions)")
			c.Logf("  4. Cleanup from one publish DELETES files from the other")
			c.Logf("  5. Result: apt-get returns 404 when downloading packages")

			// List what's actually in the pool
			poolDir := filepath.Join(publicPath, sharedPrefix, "pool", "main")
			if entries, err := os.ReadDir(poolDir); err == nil {
				c.Logf("\nActual pool directory contents (%s):", poolDir)
				for _, entry := range entries {
					c.Logf("  - %s/", entry.Name())
				}
			}

			c.Fatalf("BUG CONFIRMED (iteration %d/%d): %d files missing from shared pool",
				iter+1, numIterations, len(missingFiles))
		} else {
			c.Logf("[iter %d/%d] All %d files present - OK", iter+1, numIterations, len(expectedFiles))
		}
	}
	c.Logf("✓ All %d iterations passed - no files missing", numIterations)
}
