package etcddb

import (
	"testing"
	"time"
	"github.com/aptly-dev/aptly/database"
)

func TestEtcdWithQueue(t *testing.T) {
	// Test with queue enabled
	config := &QueueConfig{
		Enabled:         true,
		WriteQueueSize:  100,
		MaxWritesPerSec: 100,
		BatchMaxSize:    10,
		BatchMaxWait:    10 * time.Millisecond,
	}

	db, err := NewDBWithQueue("localhost:2379", config)
	if err != nil {
		t.Skipf("etcd not available: %v", err)
	}
	defer db.Close()

	// Test basic operations
	testKey := []byte("test-queue-key")
	testValue := []byte("test-queue-value")

	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Give queue time to process
	time.Sleep(100 * time.Millisecond)

	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(testValue) {
		t.Fatalf("Expected %s, got %s", testValue, retrieved)
	}

	// Clean up
	err = db.Delete(testKey)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestEtcdWithoutQueue(t *testing.T) {
	// Test with queue disabled
	config := &QueueConfig{
		Enabled: false,
	}

	db, err := NewDBWithQueue("localhost:2379", config)
	if err != nil {
		t.Skipf("etcd not available: %v", err)
	}
	defer db.Close()

	// Verify it's regular etcd storage
	_, ok := db.(*EtcDStorage)
	if !ok {
		t.Fatal("Expected EtcDStorage type")
	}

	// Test basic operations
	testKey := []byte("test-no-queue-key")
	testValue := []byte("test-no-queue-value")

	err = db.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := db.Get(testKey)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(testValue) {
		t.Fatalf("Expected %s, got %s", testValue, retrieved)
	}

	// Clean up
	err = db.Delete(testKey)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestQueueImplementsInterface(t *testing.T) {
	// Verify that our implementation satisfies the database.Storage interface
	var _ database.Storage = (*EtcDStorage)(nil)
}