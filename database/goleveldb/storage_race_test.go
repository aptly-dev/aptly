package goleveldb

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// Test for database close race condition
func TestStorageCloseRace(t *testing.T) {
	// Create temporary storage
	tempdir, err := os.MkdirTemp("", "aptly-race-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempdir)
	
	db, err := internalOpen(tempdir, true)
	if err != nil {
		t.Fatal(err)
	}
	
	storage := &storage{db: db, path: tempdir}
	
	// Put some initial data
	err = storage.Put([]byte("test-key"), []byte("test-value"))
	if err != nil {
		t.Fatal(err)
	}
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	panics := make(chan string, 100)
	
	// Start multiple goroutines doing database operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics <- fmt.Sprintf("goroutine %d panicked: %v", id, r)
				}
			}()
			
			// Continuously perform operations
			for j := 0; j < 100; j++ {
				// Try Get operation
				_, err := storage.Get([]byte("test-key"))
				if err != nil && err.Error() != "database is nil" {
					errors <- fmt.Errorf("get error in goroutine %d: %v", id, err)
					return
				}
				
				// Try Put operation
				err = storage.Put([]byte(fmt.Sprintf("key-%d-%d", id, j)), []byte("value"))
				if err != nil && err.Error() != "database is nil" {
					errors <- fmt.Errorf("put error in goroutine %d: %v", id, err)
					return
				}
				
				// Small delay to increase race window
				time.Sleep(time.Microsecond)
			}
		}(i)
	}
	
	// Start goroutines that close the database
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics <- fmt.Sprintf("close goroutine %d panicked: %v", id, r)
				}
			}()
			
			// Wait a bit then close
			time.Sleep(time.Duration(id*10) * time.Millisecond)
			err := storage.Close()
			if err != nil {
				errors <- fmt.Errorf("close error in goroutine %d: %v", id, err)
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	close(panics)
	
	// Check for panics (indicates race condition bug)
	for panic := range panics {
		t.Errorf("Race condition caused panic: %s", panic)
	}
	
	// Some errors are expected (database closed), but panics are not
	errorCount := 0
	for err := range errors {
		errorCount++
		if errorCount < 10 { // Only log first few errors
			t.Logf("Expected error during race: %v", err)
		}
	}
}

// Test concurrent operations vs close
func TestStorageConcurrentOpsVsClose(t *testing.T) {
	for attempt := 0; attempt < 5; attempt++ {
		// Create fresh storage for each attempt
		tempdir, err := os.MkdirTemp("", "aptly-concurrent-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempdir)
		
		db, err := internalOpen(tempdir, true)
		if err != nil {
			t.Fatal(err)
		}
		
		storage := &storage{db: db, path: tempdir}
		
		var wg sync.WaitGroup
		panicked := make(chan bool, 1)
		
		// Goroutine performing operations
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked <- true
				}
			}()
			
			for i := 0; i < 1000; i++ {
				storage.Get([]byte("key"))
				storage.Put([]byte("key"), []byte("value"))
				storage.Delete([]byte("key"))
			}
		}()
		
		// Goroutine closing database
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // Let operations start
			storage.Close()
		}()
		
		wg.Wait()
		
		// Check if panic occurred
		select {
		case <-panicked:
			t.Errorf("Attempt %d: Panic occurred during concurrent ops vs close", attempt)
		default:
			// No panic - good
		}
		
		close(panicked)
	}
}

// Test multiple concurrent close attempts
func TestStorageMultipleClose(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "aptly-close-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempdir)
	
	db, err := internalOpen(tempdir, true)
	if err != nil {
		t.Fatal(err)
	}
	
	storage := &storage{db: db, path: tempdir}
	
	var wg sync.WaitGroup
	panics := make(chan string, 20)
	
	// Multiple goroutines trying to close
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics <- fmt.Sprintf("close %d panicked: %v", id, r)
				}
			}()
			
			err := storage.Close()
			if err != nil {
				// Error is ok, panic is not
				t.Logf("Close %d got error (expected): %v", id, err)
			}
		}(i)
	}
	
	wg.Wait()
	close(panics)
	
	// Check for panics
	for panic := range panics {
		t.Errorf("Multiple close caused panic: %s", panic)
	}
}

// Test iterator operations during close
func TestStorageIteratorRace(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "aptly-iterator-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempdir)
	
	db, err := internalOpen(tempdir, true)
	if err != nil {
		t.Fatal(err)
	}
	
	storage := &storage{db: db, path: tempdir}
	
	// Add some data
	for i := 0; i < 100; i++ {
		storage.Put([]byte(fmt.Sprintf("key-%03d", i)), []byte("value"))
	}
	
	var wg sync.WaitGroup
	panics := make(chan string, 10)
	
	// Goroutines using iterators
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics <- fmt.Sprintf("iterator %d panicked: %v", id, r)
				}
			}()
			
			// Use methods that create iterators
			storage.KeysByPrefix([]byte("key-"))
			storage.FetchByPrefix([]byte("key-"))
			storage.HasPrefix([]byte("key-"))
		}(i)
	}
	
	// Close database while iterators are running
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		storage.Close()
	}()
	
	wg.Wait()
	close(panics)
	
	// Check for panics
	for panic := range panics {
		t.Errorf("Iterator race caused panic: %s", panic)
	}
}