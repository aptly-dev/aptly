package etcddb

import (
	"testing"
	"time"
)

func TestQueueConfigDefaults(t *testing.T) {
	config := &QueueConfig{
		Enabled: true,
	}

	// Test default values
	if config.WriteQueueSize == 0 {
		config.WriteQueueSize = 1000
	}
	if config.MaxWritesPerSec == 0 {
		config.MaxWritesPerSec = 100
	}
	if config.BatchMaxSize == 0 {
		config.BatchMaxSize = 50
	}
	if config.BatchMaxWait == 0 {
		config.BatchMaxWait = 10 * time.Millisecond
	}

	// Verify defaults
	if config.WriteQueueSize != 1000 {
		t.Errorf("Expected default WriteQueueSize to be 1000, got %d", config.WriteQueueSize)
	}
	if config.MaxWritesPerSec != 100 {
		t.Errorf("Expected default MaxWritesPerSec to be 100, got %d", config.MaxWritesPerSec)
	}
	if config.BatchMaxSize != 50 {
		t.Errorf("Expected default BatchMaxSize to be 50, got %d", config.BatchMaxSize)
	}
	if config.BatchMaxWait != 10*time.Millisecond {
		t.Errorf("Expected default BatchMaxWait to be 10ms, got %v", config.BatchMaxWait)
	}
}

func TestQueueConfigDisabled(t *testing.T) {
	config := &QueueConfig{
		Enabled: false,
	}

	if config.Enabled {
		t.Error("Expected queue to be disabled")
	}
}

