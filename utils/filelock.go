package utils

import (
	"path/filepath"
	"sync"
)

// FileLockRegistry manages file-level locks to prevent concurrent access
type FileLockRegistry struct {
	locks map[string]*sync.Mutex
	mu    sync.Mutex
}

// Global file lock registry
var fileLocks = &FileLockRegistry{
	locks: make(map[string]*sync.Mutex),
}

// LockFile acquires a lock for the given file path and returns an unlock function
func LockFile(path string) func() {
	// Normalize path to absolute to ensure consistency
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't get absolute path, use the original
		absPath = path
	}
	
	fileLocks.mu.Lock()
	lock, exists := fileLocks.locks[absPath]
	if !exists {
		lock = &sync.Mutex{}
		fileLocks.locks[absPath] = lock
	}
	fileLocks.mu.Unlock()
	
	lock.Lock()
	return func() { lock.Unlock() }
}

// LockFiles acquires locks for multiple file paths and returns an unlock function
func LockFiles(paths []string) func() {
	// Sort paths to prevent deadlock when locking multiple files
	normalizedPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		normalizedPaths = append(normalizedPaths, absPath)
	}
	
	// Simple sorting to ensure consistent lock order
	for i := 0; i < len(normalizedPaths)-1; i++ {
		for j := i + 1; j < len(normalizedPaths); j++ {
			if normalizedPaths[i] > normalizedPaths[j] {
				normalizedPaths[i], normalizedPaths[j] = normalizedPaths[j], normalizedPaths[i]
			}
		}
	}
	
	// Acquire all locks
	unlocks := make([]func(), 0, len(normalizedPaths))
	for _, path := range normalizedPaths {
		unlock := LockFile(path)
		unlocks = append(unlocks, unlock)
	}
	
	// Return function that unlocks all in reverse order
	return func() {
		for i := len(unlocks) - 1; i >= 0; i-- {
			unlocks[i]()
		}
	}
}