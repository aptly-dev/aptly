package api

import (
	"net/http"
	"strconv"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/task"
	"github.com/gin-gonic/gin"
)

// @Summary Get tasks
// @Description Get list of available tasks. Each task is returned as in “show” API.
// @Tags Tasks
// @Produce  json
// @Success 200 {array} task.Task
// @Router /api/tasks [get]
func apiTasksList(c *gin.Context) {
	list := context.TaskList()
	c.JSON(200, list.GetTasks())
}

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Router /api/tasks-clear [post]
func apiTasksClear(c *gin.Context) {
	list := context.TaskList()
	list.Clear()
	c.JSON(200, gin.H{})
}

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Failure 404 {object} Error "Not Found"
// @Router /api/tasks-wait [get]
func apiTasksWait(c *gin.Context) {
	list := context.TaskList()
	list.Wait()
	c.JSON(200, gin.H{})
}

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Failure 404 {object} Error "Not Found"
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

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Failure 404 {object} Error "Not Found"
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

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Failure 404 {object} Error "Not Found"
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

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Failure 404 {object} Error "Not Found"
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

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
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

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Failure 404 {object} Error "Not Found"
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

// FIXME: used for testing only, remove:

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Tasks
// @Produce json
// @Success 200 {object} string "msg"
// @Failure 404 {object} Error "Not Found"
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
