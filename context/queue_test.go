package context

import (
	"testing"
	"github.com/aptly-dev/aptly/utils"
)

func TestQueueConfigurationParsing(t *testing.T) {
	// Test default configuration
	config := utils.ConfigStructure{
		DatabaseBackend: utils.DBConfig{
			Type: "etcd",
			URL:  "localhost:2379",
		},
	}

	// Verify defaults are applied
	if config.DatabaseBackend.WriteQueue.Enabled {
		t.Error("Expected write queue to be disabled by default")
	}

	// Test with explicit configuration
	config.DatabaseBackend.WriteQueue = utils.WriteQConfig{
		Enabled:         true,
		QueueSize:       500,
		MaxWritesPerSec: 50,
		BatchMaxSize:    25,
		BatchMaxWaitMs:  20,
	}

	if !config.DatabaseBackend.WriteQueue.Enabled {
		t.Error("Expected write queue to be enabled")
	}
	if config.DatabaseBackend.WriteQueue.QueueSize != 500 {
		t.Errorf("Expected queue size 500, got %d", config.DatabaseBackend.WriteQueue.QueueSize)
	}
	if config.DatabaseBackend.WriteQueue.MaxWritesPerSec != 50 {
		t.Errorf("Expected max writes per sec 50, got %d", config.DatabaseBackend.WriteQueue.MaxWritesPerSec)
	}
	if config.DatabaseBackend.WriteQueue.BatchMaxSize != 25 {
		t.Errorf("Expected batch max size 25, got %d", config.DatabaseBackend.WriteQueue.BatchMaxSize)
	}
	if config.DatabaseBackend.WriteQueue.BatchMaxWaitMs != 20 {
		t.Errorf("Expected batch max wait 20ms, got %d", config.DatabaseBackend.WriteQueue.BatchMaxWaitMs)
	}
}