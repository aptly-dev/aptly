package task

import (
	"errors"

	// need to import as check as otherwise List is redeclared
	check "gopkg.in/check.v1"
)

type ListSuite struct{}

var _ = check.Suite(&ListSuite{})

func (s *ListSuite) TestList(c *check.C) {
	list := NewList()
	c.Assert(len(list.GetTasks()), check.Equals, 0)

	task, err := list.RunTaskInBackground("Successful task", nil, func(out *Output, detail *Detail) error {
		return nil
	})
	c.Assert(err, check.IsNil)
	list.WaitForTaskByID(task.ID)

	tasks := list.GetTasks()
	c.Assert(len(tasks), check.Equals, 1)
	task, _ = list.GetTaskByID(task.ID)
	c.Check(task.State, check.Equals, SUCCEEDED)
	output, _ := list.GetTaskOutputByID(task.ID)
	c.Check(output, check.Equals, "Task succeeded")
	detail, _ := list.GetTaskDetailByID(task.ID)
	c.Check(detail, check.Equals, struct{}{})

	task, err = list.RunTaskInBackground("Faulty task", nil, func(out *Output, detail *Detail) error {
		detail.Store("Details")
		out.WriteString("Test Progress\n")
		return errors.New("Task failed")
	})
	c.Assert(err, check.IsNil)
	list.WaitForTaskByID(task.ID)

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
}
