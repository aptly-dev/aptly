package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"

	. "gopkg.in/check.v1"
)

type LinkFromPoolConcurrencySuite struct {
	root          string
	poolDir       string
	storage       *PublishedStorage
	pool          *PackagePool
	cs            aptly.ChecksumStorage
	testFile      string
	testContent   []byte
	testChecksums utils.ChecksumInfo
	srcPoolPath   string
}

var _ = Suite(&LinkFromPoolConcurrencySuite{})

func (s *LinkFromPoolConcurrencySuite) SetUpTest(c *C) {
	s.root = c.MkDir()
	s.poolDir = filepath.Join(s.root, "pool")
	publishDir := filepath.Join(s.root, "public")

	// Create package pool and published storage
	s.pool = NewPackagePool(s.poolDir, true)
	s.storage = NewPublishedStorage(publishDir, "copy", "checksum")
	s.cs = NewMockChecksumStorage()

	// Create test file content
	s.testContent = []byte("test package content for concurrency testing")
	s.testFile = filepath.Join(s.root, "test-package.deb")

	err := os.WriteFile(s.testFile, s.testContent, 0644)
	c.Assert(err, IsNil)

	// Calculate checksums
	md5sum, err := utils.MD5ChecksumForFile(s.testFile)
	c.Assert(err, IsNil)

	s.testChecksums = utils.ChecksumInfo{
		Size: int64(len(s.testContent)),
		MD5:  md5sum,
	}

	// Import the test file into the pool
	s.srcPoolPath, err = s.pool.Import(s.testFile, "test-package.deb", &s.testChecksums, false, s.cs)
	c.Assert(err, IsNil)
}

func (s *LinkFromPoolConcurrencySuite) TestLinkFromPoolConcurrency(c *C) {
	// Test concurrent LinkFromPool operations to ensure no race conditions
	concurrency := 5000
	iterations := 10

	for iter := 0; iter < iterations; iter++ {
		c.Logf("Iteration %d: Testing concurrent LinkFromPool with %d goroutines", iter+1, concurrency)

		destPath := fmt.Sprintf("main/t/test%d", iter)

		var wg sync.WaitGroup
		errors := make(chan error, concurrency)
		successes := make(chan struct{}, concurrency)

		start := time.Now()

		// Launch concurrent LinkFromPool operations
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Use force=true to test the most vulnerable code path (remove-then-create)
				err := s.storage.LinkFromPool(
					"",                 // publishedPrefix
					destPath,           // publishedRelPath
					"test-package.deb", // fileName
					s.pool,             // sourcePool
					s.srcPoolPath,      // sourcePath
					s.testChecksums,    // sourceChecksums
					true,               // force - this triggers vulnerable remove-then-create pattern
				)

				if err != nil {
					errors <- fmt.Errorf("goroutine %d failed: %v", id, err)
				} else {
					successes <- struct{}{}
				}
			}(i)
		}

		// Wait for completion
		wg.Wait()
		duration := time.Since(start)

		close(errors)
		close(successes)

		// Count results
		errorCount := 0
		successCount := 0
		var firstError error

		for err := range errors {
			errorCount++
			if firstError == nil {
				firstError = err
			}
			c.Logf("Race condition error: %v", err)
		}

		for range successes {
			successCount++
		}

		c.Logf("Results: %d successes, %d errors, took %v", successCount, errorCount, duration)

		// Assert no race conditions occurred
		if errorCount > 0 {
			c.Fatalf("Race condition detected in iteration %d! "+
				"Errors: %d out of %d operations (%.1f%% failure rate). "+
				"First error: %v. "+
				"This indicates the fix is not working properly.",
				iter+1, errorCount, concurrency,
				float64(errorCount)/float64(concurrency)*100, firstError)
		}

		// Verify the final file exists and has correct content
		finalFile := filepath.Join(s.storage.rootPath, destPath, "test-package.deb")
		_, err := os.Stat(finalFile)
		c.Assert(err, IsNil, Commentf("Final file should exist after concurrent operations"))

		content, err := os.ReadFile(finalFile)
		c.Assert(err, IsNil, Commentf("Should be able to read final file"))
		c.Assert(content, DeepEquals, s.testContent, Commentf("File content should be intact after concurrent operations"))

		c.Logf("✓ Iteration %d: No race conditions detected", iter+1)
	}

	c.Logf("SUCCESS: Handled %d total concurrent operations across %d iterations with no race conditions",
		concurrency*iterations, iterations)
}

func (s *LinkFromPoolConcurrencySuite) TestLinkFromPoolConcurrencyDifferentFiles(c *C) {
	// Test concurrent operations on different files to ensure no blocking
	concurrency := 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	start := time.Now()

	// Launch concurrent operations on different destination files
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			destPath := fmt.Sprintf("main/t/test-file-%d", id)

			err := s.storage.LinkFromPool(
				"",                 // publishedPrefix
				destPath,           // publishedRelPath
				"test-package.deb", // fileName
				s.pool,             // sourcePool
				s.srcPoolPath,      // sourcePath
				s.testChecksums,    // sourceChecksums
				false,              // force
			)

			if err != nil {
				errors <- fmt.Errorf("goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	duration := time.Since(start)

	close(errors)

	// Count errors
	errorCount := 0
	for err := range errors {
		errorCount++
		c.Logf("Error: %v", err)
	}

	c.Assert(errorCount, Equals, 0, Commentf("No errors should occur when linking to different files"))
	c.Logf("SUCCESS: %d concurrent operations on different files completed in %v", concurrency, duration)

	// Verify all files were created correctly
	for i := 0; i < concurrency; i++ {
		finalFile := filepath.Join(s.storage.rootPath, fmt.Sprintf("main/t/test-file-%d", i), "test-package.deb")
		_, err := os.Stat(finalFile)
		c.Assert(err, IsNil, Commentf("File %d should exist", i))

		content, err := os.ReadFile(finalFile)
		c.Assert(err, IsNil, Commentf("Should be able to read file %d", i))
		c.Assert(content, DeepEquals, s.testContent, Commentf("File %d content should be correct", i))
	}
}

func (s *LinkFromPoolConcurrencySuite) TestLinkFromPoolWithoutForceNoConcurrencyIssues(c *C) {
	// Test that when force=false, concurrent operations fail gracefully without corruption
	concurrency := 20
	destPath := "main/t/single-dest"

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)
	successes := make(chan struct{}, concurrency)

	// First, create the file so subsequent operations will conflict
	err := s.storage.LinkFromPool("", destPath, "test-package.deb", s.pool, s.srcPoolPath, s.testChecksums, false)
	c.Assert(err, IsNil)

	start := time.Now()

	// Launch concurrent operations that should mostly fail
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			err := s.storage.LinkFromPool(
				"",                 // publishedPrefix
				destPath,           // publishedRelPath
				"test-package.deb", // fileName
				s.pool,             // sourcePool
				s.srcPoolPath,      // sourcePath
				s.testChecksums,    // sourceChecksums
				false,              // force=false - should fail if file exists and is same
			)

			if err != nil {
				errors <- err
			} else {
				successes <- struct{}{}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	duration := time.Since(start)

	close(errors)
	close(successes)

	errorCount := 0
	successCount := 0

	for range errors {
		errorCount++
	}

	for range successes {
		successCount++
	}

	c.Logf("Results with force=false: %d successes, %d errors, took %v", successCount, errorCount, duration)

	// With force=false and identical files, operations should succeed (file already exists with same content)
	// No race conditions should cause crashes or corruption
	c.Assert(errorCount, Equals, 0, Commentf("With identical files and force=false, operations should succeed"))

	// Verify the file still exists and has correct content
	finalFile := filepath.Join(s.storage.rootPath, destPath, "test-package.deb")
	content, err := os.ReadFile(finalFile)
	c.Assert(err, IsNil)
	c.Assert(content, DeepEquals, s.testContent, Commentf("File should not be corrupted by concurrent access"))
}
