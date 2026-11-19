package task

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/aptly-dev/aptly/aptly"

	// need to import as check as otherwise List is redeclared
	check "gopkg.in/check.v1"
)

type ListSuite struct{}

var _ = check.Suite(&ListSuite{})

func (s *ListSuite) TestList(c *check.C) {
	list := NewList()
	c.Assert(len(list.GetTasks()), check.Equals, 0)

	task, err := list.RunTaskInBackground("Successful task", nil, func(out aptly.Progress, detail *Detail) (*ProcessReturnValue, error) {
		return nil, nil
	})
	c.Assert(err, check.IsNil)
	_, _ = list.WaitForTaskByID(task.ID)

	tasks := list.GetTasks()
	c.Assert(len(tasks), check.Equals, 1)
	task, _ = list.GetTaskByID(task.ID)
	c.Check(task.State, check.Equals, SUCCEEDED)
	output, _ := list.GetTaskOutputByID(task.ID)
	c.Check(output, check.Equals, "Task succeeded")
	detail, _ := list.GetTaskDetailByID(task.ID)
	c.Check(detail, check.Equals, struct{}{})

	task, err = list.RunTaskInBackground("Faulty task", nil, func(out aptly.Progress, detail *Detail) (*ProcessReturnValue, error) {
		detail.Store("Details")
		out.Printf("Test Progress\n")
		return nil, errors.New("Task failed")
	})
	c.Assert(err, check.IsNil)
	_, _ = list.WaitForTaskByID(task.ID)

	tasks = list.GetTasks()
	c.Assert(len(tasks), check.Equals, 2)
	task, _ = list.GetTaskByID(task.ID)
	c.Check(task.State, check.Equals, FAILED)
	output, _ = list.GetTaskOutputByID(task.ID)
	c.Check(output, check.Equals, "Test Progress\nTask failed with error: Task failed")
	detail, _ = list.GetTaskDetailByID(task.ID)
	c.Check(detail, check.Equals, "Details")
	_, deleteErr := list.DeleteTaskByID(task.ID)
	c.Check(deleteErr, check.IsNil)
	list.Stop()
}

// A holds r1 and r2; B (r1) and C (r2) queue behind it, D (r1) behind B.
// When A finishes, C must not wait for the long-running B, while D still does.
func (s *ListSuite) TestAllUnblockedTasksStart(c *check.C) {
	list := NewList()
	releaseA := make(chan struct{})
	releaseB := make(chan struct{})
	defer func() {
		close(releaseB)
		list.Wait()
		list.Stop()
	}()

	a, _ := list.RunTaskInBackground("A", []string{"r1", "r2"}, func(aptly.Progress, *Detail) (*ProcessReturnValue, error) {
		<-releaseA
		return nil, nil
	})
	_, _ = list.RunTaskInBackground("B", []string{"r1"}, func(aptly.Progress, *Detail) (*ProcessReturnValue, error) {
		<-releaseB
		return nil, nil
	})
	taskC, _ := list.RunTaskInBackground("C", []string{"r2"}, func(aptly.Progress, *Detail) (*ProcessReturnValue, error) {
		return nil, nil
	})
	taskD, _ := list.RunTaskInBackground("D", []string{"r1"}, func(aptly.Progress, *Detail) (*ProcessReturnValue, error) {
		return nil, nil
	})

	close(releaseA)
	_, _ = list.WaitForTaskByID(a.ID)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if t, _ := list.GetTaskByID(taskC.ID); t.State == SUCCEEDED {
			t, _ = list.GetTaskByID(taskD.ID)
			c.Check(t.State, check.Equals, IDLE)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t, _ := list.GetTaskByID(taskC.ID)
	c.Fatalf("C still in state %d 2s after its resources were freed", t.State)
}

// Resource-free tasks waiting behind a global holder are released together;
// each must run once even when earlier ones finish and rescan before the rest are received.
func (s *ListSuite) TestReleasedTasksRunOnce(c *check.C) {
	list := NewList()
	defer list.Stop()
	releaseA := make(chan struct{})

	_, _ = list.RunTaskInBackground("A", []string{AllResourcesKey}, func(aptly.Progress, *Detail) (*ProcessReturnValue, error) {
		<-releaseA
		return nil, nil
	})
	var runs [50]int32
	for i := range runs {
		_, _ = list.RunTaskInBackground(fmt.Sprintf("C%d", i), nil, func(aptly.Progress, *Detail) (*ProcessReturnValue, error) {
			atomic.AddInt32(&runs[i], 1)
			return nil, nil
		})
	}

	close(releaseA)
	list.Wait()
	for i := range runs {
		c.Check(atomic.LoadInt32(&runs[i]), check.Equals, int32(1), check.Commentf("C%d", i))
	}
}
