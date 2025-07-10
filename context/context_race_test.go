package context

import (
	"fmt"
	"sync"
	"testing"
	"time"
	
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
)

// Test for unsafe map access race condition
func TestPublishedStorageMapRace(t *testing.T) {
	// Create a context with empty config
	context := &AptlyContext{}
	// publishedStorages is now sync.Map, initialized by zero value
	
	// Mock config
	utils.Config = utils.ConfigStructure{
		RootDir: "/tmp/aptly-test",
		FileSystemPublishRoots: map[string]utils.FileSystemPublishRoot{
			"test": {RootDir: "/tmp/test", LinkMethod: "hardlink"},
		},
	}
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	// Simulate concurrent access to the same storage
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("panic in goroutine %d: %v", id, r)
				}
			}()
			
			// All goroutines try to access the same storage
			storage := context.GetPublishedStorage("filesystem:test")
			if storage == nil {
				errors <- fmt.Errorf("got nil storage in goroutine %d", id)
			}
		}(i)
	}
	
	// Also test different storages to trigger map growth
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("panic in storage %d: %v", id, r)
				}
			}()
			
			// Add new storage configurations
			storageName := fmt.Sprintf("filesystem:test%d", id)
			utils.Config.FileSystemPublishRoots[fmt.Sprintf("test%d", id)] = utils.FileSystemPublishRoot{
				RootDir: fmt.Sprintf("/tmp/test%d", id), 
				LinkMethod: "hardlink",
			}
			
			storage := context.GetPublishedStorage(storageName)
			if storage == nil {
				errors <- fmt.Errorf("got nil storage for %s", storageName)
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for any errors or panics
	for err := range errors {
		t.Errorf("Race condition error: %v", err)
	}
}

// Test for concurrent map writes
func TestPublishedStorageConcurrentWrites(t *testing.T) {
	context := &AptlyContext{}
	
	utils.Config = utils.ConfigStructure{
		RootDir: "/tmp/aptly-test",
		FileSystemPublishRoots: make(map[string]utils.FileSystemPublishRoot),
	}
	
	var wg sync.WaitGroup
	panics := make(chan string, 100)
	
	// Multiple goroutines trying to create different storages simultaneously
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics <- fmt.Sprintf("goroutine %d panicked: %v", id, r)
				}
			}()
			
			storageName := fmt.Sprintf("filesystem:concurrent%d", id)
			utils.Config.FileSystemPublishRoots[fmt.Sprintf("concurrent%d", id)] = utils.FileSystemPublishRoot{
				RootDir: fmt.Sprintf("/tmp/concurrent%d", id),
				LinkMethod: "hardlink",
			}
			
			// This should trigger concurrent map writes
			_ = context.GetPublishedStorage(storageName)
			
			// Add some delay to increase chance of race
			time.Sleep(time.Millisecond)
			
			// Access again to ensure consistency
			storage2 := context.GetPublishedStorage(storageName)
			if storage2 == nil {
				panics <- fmt.Sprintf("inconsistent storage access in goroutine %d", id)
			}
		}(i)
	}
	
	wg.Wait()
	close(panics)
	
	// Check for panics (indicating race condition)
	for panic := range panics {
		t.Errorf("Concurrent map access issue: %s", panic)
	}
}

// Test for storage initialization race
func TestPublishedStorageInitRace(t *testing.T) {
	// Run this test multiple times to increase chance of catching race
	for attempt := 0; attempt < 10; attempt++ {
		context := &AptlyContext{}
		
		utils.Config = utils.ConfigStructure{
			RootDir: "/tmp/aptly-test",
			FileSystemPublishRoots: map[string]utils.FileSystemPublishRoot{
				"race": {RootDir: "/tmp/race", LinkMethod: "hardlink"},
			},
		}
		
		var wg sync.WaitGroup
		storages := make([]aptly.PublishedStorage, 10)
		
		// Multiple goroutines accessing the same non-existent storage
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				storages[idx] = context.GetPublishedStorage("filesystem:race")
			}(i)
		}
		
		wg.Wait()
		
		// All should get the same storage instance
		firstStorage := storages[0]
		for i := 1; i < len(storages); i++ {
			if storages[i] != firstStorage {
				t.Errorf("Attempt %d: Got different storage instances: race condition in initialization", attempt)
				break
			}
		}
	}
}