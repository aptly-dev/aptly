package task

import (
	check "gopkg.in/check.v1"
)

// Resources test suite (uses the same test runner as task_test.go)

type ResourcesSuite struct{}

var _ = check.Suite(&ResourcesSuite{})

func (s *ResourcesSuite) TestResourceConflictError(c *check.C) {
	// Test ResourceConflictError
	task1 := Task{ID: 1, Name: "Test Task 1"}
	task2 := Task{ID: 2, Name: "Test Task 2"}
	
	err := &ResourceConflictError{
		Tasks:   []Task{task1, task2},
		Message: "Resource conflict detected",
	}
	
	c.Check(err.Error(), check.Equals, "Resource conflict detected")
	c.Check(len(err.Tasks), check.Equals, 2)
	c.Check(err.Tasks[0].ID, check.Equals, 1)
	c.Check(err.Tasks[1].ID, check.Equals, 2)
}

func (s *ResourcesSuite) TestNewResourcesSet(c *check.C) {
	// Test creating new resources set
	rs := NewResourcesSet()
	c.Check(rs, check.NotNil)
	c.Check(rs.set, check.NotNil)
	c.Check(len(rs.set), check.Equals, 0)
}

func (s *ResourcesSuite) TestMarkInUse(c *check.C) {
	rs := NewResourcesSet()
	task := &Task{ID: 1, Name: "Test Task"}
	
	// Mark resources as in use
	resources := []string{"resource1", "resource2"}
	rs.MarkInUse(resources, task)
	
	c.Check(len(rs.set), check.Equals, 2)
	c.Check(rs.set["resource1"], check.Equals, task)
	c.Check(rs.set["resource2"], check.Equals, task)
}

func (s *ResourcesSuite) TestUsedByEmpty(c *check.C) {
	rs := NewResourcesSet()
	
	// Test with empty resource set
	tasks := rs.UsedBy([]string{"resource1"})
	c.Check(len(tasks), check.Equals, 0)
}

func (s *ResourcesSuite) TestUsedByBasic(c *check.C) {
	rs := NewResourcesSet()
	task1 := &Task{ID: 1, Name: "Task 1"}
	task2 := &Task{ID: 2, Name: "Task 2"}
	
	// Mark different resources
	rs.MarkInUse([]string{"resource1"}, task1)
	rs.MarkInUse([]string{"resource2"}, task2)
	
	// Test finding tasks by resource
	tasks := rs.UsedBy([]string{"resource1"})
	c.Check(len(tasks), check.Equals, 1)
	c.Check(tasks[0].ID, check.Equals, 1)
	
	tasks = rs.UsedBy([]string{"resource2"})
	c.Check(len(tasks), check.Equals, 1)
	c.Check(tasks[0].ID, check.Equals, 2)
	
	// Test non-existent resource
	tasks = rs.UsedBy([]string{"nonexistent"})
	c.Check(len(tasks), check.Equals, 0)
}

func (s *ResourcesSuite) TestUsedByAllLocalRepos(c *check.C) {
	rs := NewResourcesSet()
	task1 := &Task{ID: 1, Name: "Local Task 1"}
	task2 := &Task{ID: 2, Name: "Local Task 2"}
	task3 := &Task{ID: 3, Name: "Remote Task"}
	
	// Mark resources with local repo prefix "L"
	rs.MarkInUse([]string{"Lrepo1"}, task1)
	rs.MarkInUse([]string{"Lrepo2"}, task2)
	rs.MarkInUse([]string{"Rremote1"}, task3)
	
	// Test AllLocalReposResourcesKey
	tasks := rs.UsedBy([]string{AllLocalReposResourcesKey})
	c.Check(len(tasks), check.Equals, 2)
	
	// Verify both local repo tasks are found
	var foundTask1, foundTask2 bool
	for _, task := range tasks {
		if task.ID == 1 {
			foundTask1 = true
		}
		if task.ID == 2 {
			foundTask2 = true
		}
	}
	c.Check(foundTask1, check.Equals, true)
	c.Check(foundTask2, check.Equals, true)
}

func (s *ResourcesSuite) TestUsedByAllResources(c *check.C) {
	rs := NewResourcesSet()
	task1 := &Task{ID: 1, Name: "Task 1"}
	task2 := &Task{ID: 2, Name: "Task 2"}
	task3 := &Task{ID: 3, Name: "Task 3"}
	
	// Mark different resources
	rs.MarkInUse([]string{"resource1"}, task1)
	rs.MarkInUse([]string{"resource2"}, task2)
	rs.MarkInUse([]string{"resource3"}, task3)
	
	// Test AllResourcesKey
	tasks := rs.UsedBy([]string{AllResourcesKey})
	c.Check(len(tasks), check.Equals, 3)
	
	// Verify all tasks are found
	taskIDs := make(map[int]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}
	c.Check(taskIDs[1], check.Equals, true)
	c.Check(taskIDs[2], check.Equals, true)
	c.Check(taskIDs[3], check.Equals, true)
}

func (s *ResourcesSuite) TestUsedBySpecialResourceMarked(c *check.C) {
	rs := NewResourcesSet()
	allLocalTask := &Task{ID: 1, Name: "All Local Task"}
	allTask := &Task{ID: 2, Name: "All Task"}
	
	// Mark special resources directly
	rs.MarkInUse([]string{AllLocalReposResourcesKey}, allLocalTask)
	rs.MarkInUse([]string{AllResourcesKey}, allTask)
	
	// Test finding by any resource should include special tasks
	tasks := rs.UsedBy([]string{"any-resource"})
	c.Check(len(tasks), check.Equals, 2)
	
	taskIDs := make(map[int]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}
	c.Check(taskIDs[1], check.Equals, true)
	c.Check(taskIDs[2], check.Equals, true)
}

func (s *ResourcesSuite) TestAppendTaskDuplicates(c *check.C) {
	// Test appendTask function with duplicates
	task1 := &Task{ID: 1, Name: "Task 1"}
	task2 := &Task{ID: 2, Name: "Task 2"}
	
	var tasks []Task
	
	// Add first task
	tasks = appendTask(tasks, task1)
	c.Check(len(tasks), check.Equals, 1)
	c.Check(tasks[0].ID, check.Equals, 1)
	
	// Add second task
	tasks = appendTask(tasks, task2)
	c.Check(len(tasks), check.Equals, 2)
	
	// Try to add first task again (should not duplicate)
	tasks = appendTask(tasks, task1)
	c.Check(len(tasks), check.Equals, 2)
	
	// Verify no duplicate
	taskIDs := make(map[int]int)
	for _, task := range tasks {
		taskIDs[task.ID]++
	}
	c.Check(taskIDs[1], check.Equals, 1)
	c.Check(taskIDs[2], check.Equals, 1)
}

func (s *ResourcesSuite) TestFree(c *check.C) {
	rs := NewResourcesSet()
	task1 := &Task{ID: 1, Name: "Task 1"}
	task2 := &Task{ID: 2, Name: "Task 2"}
	
	// Mark resources
	rs.MarkInUse([]string{"resource1", "resource2"}, task1)
	rs.MarkInUse([]string{"resource3"}, task2)
	
	c.Check(len(rs.set), check.Equals, 3)
	
	// Free some resources
	rs.Free([]string{"resource1", "resource3"})
	c.Check(len(rs.set), check.Equals, 1)
	c.Check(rs.set["resource2"], check.Equals, task1)
	
	// Verify freed resources are no longer in use
	tasks := rs.UsedBy([]string{"resource1"})
	c.Check(len(tasks), check.Equals, 0)
	
	tasks = rs.UsedBy([]string{"resource3"})
	c.Check(len(tasks), check.Equals, 0)
	
	// But resource2 should still be in use
	tasks = rs.UsedBy([]string{"resource2"})
	c.Check(len(tasks), check.Equals, 1)
	c.Check(tasks[0].ID, check.Equals, 1)
}

func (s *ResourcesSuite) TestFreeNonExistentResources(c *check.C) {
	rs := NewResourcesSet()
	task := &Task{ID: 1, Name: "Task 1"}
	
	rs.MarkInUse([]string{"resource1"}, task)
	c.Check(len(rs.set), check.Equals, 1)
	
	// Free non-existent resources (should not panic)
	rs.Free([]string{"nonexistent1", "nonexistent2"})
	c.Check(len(rs.set), check.Equals, 1)
	
	// Free mix of existing and non-existent
	rs.Free([]string{"resource1", "nonexistent"})
	c.Check(len(rs.set), check.Equals, 0)
}

func (s *ResourcesSuite) TestComplexScenario(c *check.C) {
	rs := NewResourcesSet()
	localTask1 := &Task{ID: 1, Name: "Local Task 1"}
	localTask2 := &Task{ID: 2, Name: "Local Task 2"}
	remoteTask := &Task{ID: 3, Name: "Remote Task"}
	globalTask := &Task{ID: 4, Name: "Global Task"}
	
	// Set up complex scenario
	rs.MarkInUse([]string{"Llocal-repo-1"}, localTask1)
	rs.MarkInUse([]string{"Llocal-repo-2"}, localTask2)
	rs.MarkInUse([]string{"Rremote-repo"}, remoteTask)
	rs.MarkInUse([]string{AllResourcesKey}, globalTask)
	
	// Test various queries
	tasks := rs.UsedBy([]string{"Llocal-repo-1"})
	c.Check(len(tasks), check.Equals, 2) // localTask1 + globalTask
	
	tasks = rs.UsedBy([]string{AllLocalReposResourcesKey})
	c.Check(len(tasks), check.Equals, 3) // localTask1 + localTask2 + globalTask
	
	tasks = rs.UsedBy([]string{"Rremote-repo"})
	c.Check(len(tasks), check.Equals, 2) // remoteTask + globalTask
	
	tasks = rs.UsedBy([]string{AllResourcesKey})
	c.Check(len(tasks), check.Equals, 4) // All tasks
}