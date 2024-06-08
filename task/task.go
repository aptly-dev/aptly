package task

import (
	"sync"
	"sync/atomic"

	"github.com/aptly-dev/aptly/aptly"
)

// State task is in
type State int

// Detail represents custom task details
type Detail struct {
	atomic.Value
}

// PublishDetail represents publish task details
type PublishDetail struct {
	*Detail
	TotalNumberOfPackages     int64
	RemainingNumberOfPackages int64
}

type ProcessReturnValue struct {
	Code  int
	Value interface{}
}

// Process is a function implementing the actual task logic
type Process func(out aptly.Progress, detail *Detail) (*ProcessReturnValue, error)

const (
	// IDLE when task is waiting
	IDLE State = iota
	// RUNNING when task is running
	RUNNING
	// SUCCEEDED when task is successfully finished
	SUCCEEDED
	// FAILED when task failed
	FAILED
)

// Task represents as task in a queue encapsulates process code
type Task struct {
	output             *Output
	detail             *Detail
	process            Process
	processReturnValue *ProcessReturnValue
	err                error
	Name               string
	ID                 int
	State              State
	resources          []string
	wgTask             *sync.WaitGroup
}

// NewTask creates new task
func NewTask(process Process, name string, ID int, resources []string, wgTask *sync.WaitGroup) *Task {
	task := &Task{
		output:    NewOutput(),
		detail:    &Detail{},
		process:   process,
		Name:      name,
		ID:        ID,
		State:     IDLE,
		resources: resources,
		wgTask:    wgTask,
	}
	return task
}
