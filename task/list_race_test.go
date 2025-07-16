package task

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/aptly-dev/aptly/aptly"
)

// Test 1: TaskList Goroutine Leak (with explicit cleanup)
func TestTaskListGoroutineLeak(t *testing.T) {
	// Record initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Create multiple TaskLists and store them for explicit cleanup
	lists := make([]*TaskList, 10)
	for i := 0; i < 10; i++ {
		lists[i] = NewList()
	}

	// Test proper cleanup with Stop()
	for _, list := range lists {
		list.Stop()
	}

	// Allow time for goroutines to stop
	time.Sleep(50 * time.Millisecond)

	// Check if goroutines properly cleaned up
	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - initialGoroutines

	if leaked > 0 {
		t.Errorf("Goroutine leak detected even with Stop(): %d goroutines leaked", leaked)
		t.Logf("Initial: %d, Final: %d", initialGoroutines, finalGoroutines)
	}
}

// Test 2: Double Close Panic
func TestTaskListDoubleClosePanic(t *testing.T) {
	list := NewList()

	// Call Stop() multiple times concurrently
	var wg sync.WaitGroup
	panicked := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked <- true
				}
			}()
			list.Stop()
		}()
	}

	wg.Wait()
	close(panicked)

	// Check if any goroutine panicked
	for panic := range panicked {
		if panic {
			t.Error("Double close caused panic")
			break
		}
	}
}

// Test 3: Concurrent TaskList Operations
func TestTaskListConcurrentOperations(t *testing.T) {
	list := NewList()
	defer list.Stop()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Simulate concurrent task creation and deletion
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create task
			task, err := list.RunTaskInBackground(
				"test-task",
				[]string{"resource-" + string(rune(id%5))},
				func(progress aptly.Progress, detail *Detail) (*ProcessReturnValue, error) {
					time.Sleep(10 * time.Millisecond)
					return &ProcessReturnValue{Code: 200}, nil
				},
			)

			if err != nil {
				select {
				case errors <- err:
				default:
				}
				return
			}

			// Try to get task details
			_, getErr := list.GetTaskByID(task.ID)
			if getErr != nil {
				select {
				case errors <- getErr:
				default:
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

// Test 4: Resource Management Race
func TestTaskListResourceRace(t *testing.T) {
	list := NewList()
	defer list.Stop()

	var wg sync.WaitGroup
	completedTasks := make(chan int, 100)

	// Create tasks that use the same resource
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			task, err := list.RunTaskInBackground(
				"resource-test",
				[]string{"shared-resource"},
				func(progress aptly.Progress, detail *Detail) (*ProcessReturnValue, error) {
					time.Sleep(50 * time.Millisecond)
					return &ProcessReturnValue{Code: 200}, nil
				},
			)

			if err != nil {
				t.Errorf("Failed to create task: %v", err)
				return
			}

			// Wait for task completion
			_, waitErr := list.WaitForTaskByID(task.ID)
			if waitErr != nil {
				t.Errorf("Failed to wait for task: %v", waitErr)
				return
			}

			completedTasks <- task.ID
		}(i)
	}

	// Wait for all tasks to complete
	go func() {
		wg.Wait()
		close(completedTasks)
	}()

	// Collect completed task IDs
	var completed []int
	for taskID := range completedTasks {
		completed = append(completed, taskID)
	}

	// Check that all tasks completed
	if len(completed) != 20 {
		t.Errorf("Expected 20 completed tasks, got %d", len(completed))
	}

	// Check for resource leaks by examining remaining tasks
	remainingTasks := list.GetTasks()
	runningCount := 0
	for _, task := range remainingTasks {
		if task.State == RUNNING {
			runningCount++
		}
	}

	if runningCount > 0 {
		t.Errorf("Resource leak: %d tasks still running", runningCount)
	}
}
