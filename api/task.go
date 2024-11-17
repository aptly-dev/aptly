package api

import (
	"net/http"
	"strconv"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/task"
	"github.com/gin-gonic/gin"
)

// @Summary Get tasks
// @Description **Get list of available tasks. Each task is returned as in “show” API**
// @Tags Tasks
// @Produce json
// @Success 200 {array} task.Task
// @Router /api/tasks [get]
func apiTasksList(c *gin.Context) {
	list := context.TaskList()
	c.JSON(200, list.GetTasks())
}

// @Summary Clear finished and failed tasks
// @Description **Removes finished and failed tasks from internal task list**
// @Tags Tasks
// @Produce json
// @Success 200 ""
// @Router /api/tasks-clear [post]
func apiTasksClear(c *gin.Context) {
	list := context.TaskList()
	list.Clear()
	c.JSON(200, gin.H{})
}

// @Summary Wait for task completion
// @Description **Waits for and returns when all running tasks are complete**
// @Tags Tasks
// @Produce json
// @Success 200 ""
// @Router /api/tasks-wait [get]
func apiTasksWait(c *gin.Context) {
	list := context.TaskList()
	list.Wait()
	c.JSON(200, gin.H{})
}

// @Summary Wait for task to process
// @Description **Waits for and returns when given Task ID is complete**
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} task.Task
// @Failure 500 {object} Error "invalid syntax, bad id?"
// @Failure 400 {object} Error "Task Not Found"
// @Router /api/tasks/{id}/wait [get]
func apiTasksWaitForTaskByID(c *gin.Context) {
	list := context.TaskList()
	id, err := strconv.ParseInt(c.Params.ByName("id"), 10, 0)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	task, err := list.WaitForTaskByID(int(id))
	if err != nil {
		AbortWithJSONError(c, 400, err)
		return
	}

	c.JSON(200, task)
}

// @Summary Return task information
// @Description **Return task information for a given ID**
// @Tags Tasks
// @Produce plain
// @Param id path int true "Task ID"
// @Success 200 {object} task.Task
// @Failure 500 {object} Error "invalid syntax, bad id?"
// @Failure 404 {object} Error "Task Not Found"
// @Router /api/tasks/{id} [get]
func apiTasksShow(c *gin.Context) {
	list := context.TaskList()
	id, err := strconv.ParseInt(c.Params.ByName("id"), 10, 0)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	var task task.Task
	task, err = list.GetTaskByID(int(id))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	c.JSON(200, task)
}

// @Summary Return task output
// @Description **Return task output for a given ID**
// @Tags Tasks
// @Produce plain
// @Param id path int true "Task ID"
// @Success 200 {object} string "Task output"
// @Failure 500 {object} Error "invalid syntax, bad ID?"
// @Failure 404 {object} Error "Task Not Found"
// @Router /api/tasks/{id}/output [get]
func apiTasksOutputShow(c *gin.Context) {
	list := context.TaskList()
	id, err := strconv.ParseInt(c.Params.ByName("id"), 10, 0)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	var output string
	output, err = list.GetTaskOutputByID(int(id))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	c.JSON(200, output)
}

// @Summary Return task detail
// @Description **Return task detail for a given ID**
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} string "Task detail"
// @Failure 500 {object} Error "invalid syntax, bad ID?"
// @Failure 404 {object} Error "Task Not Found"
// @Router /api/tasks/{id}/detail [get]
func apiTasksDetailShow(c *gin.Context) {
	list := context.TaskList()
	id, err := strconv.ParseInt(c.Params.ByName("id"), 10, 0)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	var detail interface{}
	detail, err = list.GetTaskDetailByID(int(id))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	c.JSON(200, detail)
}

// @Summary Return task return value (status code)
// @Description **Return task return value (status code) by given ID**
// @Tags Tasks
// @Produce plain
// @Param id path int true "Task ID"
// @Success 200 {object} string "msg"
// @Failure 500 {object} Error "invalid syntax, bad ID?"
// @Failure 404 {object} Error "Not Found"
// @Router /api/tasks/{id}/return_value [get]
func apiTasksReturnValueShow(c *gin.Context) {
	list := context.TaskList()
	id, err := strconv.ParseInt(c.Params.ByName("id"), 10, 0)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	output, err := list.GetTaskReturnValueByID(int(id))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	c.JSON(200, output)
}

// @Summary Delete task
// @Description **Delete completed task by given ID. Does not stop task execution**
// @Tags Tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} task.Task
// @Failure 500 {object} Error "invalid syntax, bad ID?"
// @Failure 400 {object} Error "Task in progress or not found"
// @Router /api/tasks/{id} [delete]
func apiTasksDelete(c *gin.Context) {
	list := context.TaskList()
	id, err := strconv.ParseInt(c.Params.ByName("id"), 10, 0)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	var delTask task.Task
	delTask, err = list.DeleteTaskByID(int(id))
	if err != nil {
		AbortWithJSONError(c, 400, err)
		return
	}

	c.JSON(200, delTask)
}

// @Summary Dummy endpoint used for testing.
// @Description **Dummy endpoint used for testing**
// @Tags Tasks
// @Produce json
// @Param _async query bool false "Run task in background using tasks API"
// @Success 200 {object} task.ProcessReturnValue
// @Router /api/tasks-dummy [post]
func apiTasksDummy(c *gin.Context) {
	resources := []string{"dummy"}
	taskName := "Dummy task"
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		out.Printf("Dummy task started\n")
		detail.Store([]int{1, 2, 3})
		out.Printf("Dummy task finished\n")
		return &task.ProcessReturnValue{Code: http.StatusTeapot, Value: []int{1, 2, 3}}, nil
	})
}
