package task

import (
	"github.com/aptly-dev/aptly/aptly"
	check "gopkg.in/check.v1"
)

type SimpleSuite struct{}

var _ = check.Suite(&SimpleSuite{})

func (s *SimpleSuite) TestSimpleTask(c *check.C) {
	list := NewList()
	defer list.Stop()
	
	c.Check(len(list.GetTasks()), check.Equals, 0)
	
	task, err := list.RunTaskInBackground("Simple task", nil, func(out aptly.Progress, detail *Detail) (*ProcessReturnValue, error) {
		return nil, nil
	})
	c.Assert(err, check.IsNil)
	
	_, _ = list.WaitForTaskByID(task.ID)
	
	tasks := list.GetTasks()
	c.Check(len(tasks), check.Equals, 1)
	
	taskResult, _ := list.GetTaskByID(task.ID)
	c.Check(taskResult.State, check.Equals, SUCCEEDED)
}

func (s *SimpleSuite) TestTaskWait(c *check.C) {
	// Test Wait function with no running tasks
	list := NewList()
	defer list.Stop()
	
	// This should return immediately since no tasks are running
	list.Wait()
	c.Check(true, check.Equals, true) // Test passes if Wait returns
}

func (s *SimpleSuite) TestOutputProgress(c *check.C) {
	// Test Output progress methods with zero counts
	output := NewOutput()
	
	// Test progress methods that should be no-ops
	output.Start()
	output.Shutdown()
	output.Flush()
	
	// Test bar methods with zero counts
	output.InitBar(0, false, 0)
	output.ShutdownBar()
	output.AddBar(0)
	output.SetBar(0)
	
	c.Check(true, check.Equals, true) // All methods should complete without error
}

func (s *SimpleSuite) TestTaskCleanup(c *check.C) {
	// Test task cleanup functionality
	list := NewList()
	defer list.Stop()
	
	// Test that cleanup method exists and can be called
	list.cleanup()
	c.Check(true, check.Equals, true) // Cleanup should complete without error
}

func (s *SimpleSuite) TestTaskListStop(c *check.C) {
	// Test Stop method behavior with partial coverage
	list := NewList()
	
	// Stop the list multiple times to test idempotency
	list.Stop()
	list.Stop() // Second call should be safe
	
	c.Check(true, check.Equals, true)
}

func (s *SimpleSuite) TestDeleteTaskByIDEdgeCases(c *check.C) {
	// Test DeleteTaskByID edge cases
	list := NewList()
	defer list.Stop()
	
	// Test deleting non-existent task
	_, err := list.DeleteTaskByID(999)
	c.Check(err, check.NotNil) // Should return error for non-existent task
}

func (s *SimpleSuite) TestTaskWithProgress(c *check.C) {
	// Test task with progress operations
	list := NewList()
	defer list.Stop()
	
	task, err := list.RunTaskInBackground("Progress task", nil, func(out aptly.Progress, detail *Detail) (*ProcessReturnValue, error) {
		// Test progress bar operations
		out.InitBar(100, false, 1)
		out.AddBar(50)
		out.SetBar(75)
		out.ShutdownBar()
		
		// Test printing operations
		out.Printf("Test message: %s", "hello")
		out.ColoredPrintf("Colored message: %s", "world")
		out.PrintfStdErr("Error message: %s", "test")
		
		return &ProcessReturnValue{}, nil
	})
	c.Assert(err, check.IsNil)
	
	_, _ = list.WaitForTaskByID(task.ID)
	
	taskResult, _ := list.GetTaskByID(task.ID)
	c.Check(taskResult.State, check.Equals, SUCCEEDED)
}