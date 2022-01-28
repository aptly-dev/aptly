package task

import (
	"strings"
)

// AllLocalReposResourcesKey to be used as resource key when all local repos are needed
const AllLocalReposResourcesKey = "__alllocalrepos__"

// AllResourcesKey to be used as resource key when all resources are needed
const AllResourcesKey = "__all__"

// ResourceConflictError represents a list tasks
// using conflicitng resources
type ResourceConflictError struct {
	Tasks   []Task
	Message string
}

func (e *ResourceConflictError) Error() string {
	return e.Message
}

// ResourcesSet represents a set of task resources.
// A resource is represented by its unique key
type ResourcesSet struct {
	set map[string]*Task
}

// NewResourcesSet creates new instance of resources set
func NewResourcesSet() *ResourcesSet {
	return &ResourcesSet{make(map[string]*Task)}
}

// MarkInUse given resources as used by given task
func (r *ResourcesSet) MarkInUse(resources []string, task *Task) {
	for _, resource := range resources {
		r.set[resource] = task
	}
}

// UsedBy checks whether one of given resources
// is used by a task and if yes returns slice of such task
func (r *ResourcesSet) UsedBy(resources []string) []Task {
	var tasks []Task
	var task *Task
	var found bool

	for _, resource := range resources {

		if resource == AllLocalReposResourcesKey {
			for taskResource, task := range r.set {
				if strings.HasPrefix(taskResource, "L") {
					tasks = appendTask(tasks, task)
				}
			}
		} else if resource == AllResourcesKey {
			for _, task := range r.set {
				tasks = appendTask(tasks, task)
			}

			break
		}

		task, found = r.set[resource]
		if found {
			tasks = appendTask(tasks, task)
		}
	}

	task, found = r.set[AllLocalReposResourcesKey]
	if found {
		tasks = appendTask(tasks, task)
	}
	task, found = r.set[AllResourcesKey]
	if found {
		tasks = appendTask(tasks, task)
	}

	return tasks
}

// appendTask only appends task to tasks slice if not already
// on slice
func appendTask(tasks []Task, task *Task) []Task {
	needsAppending := true
	for _, givenTask := range tasks {
		if givenTask.ID == task.ID {
			needsAppending = false
			break
		}
	}

	if needsAppending {
		return append(tasks, *task)
	}

	return tasks
}

// Free removes given resources from dependency set
func (r *ResourcesSet) Free(resources []string) {
	for _, resource := range resources {
		delete(r.set, resource)
	}
}
