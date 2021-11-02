package task

import (
	"fmt"
	"github.com/aptly-dev/aptly/aptly"
	"sync"
)

// List is handling list of processes and makes sure
// only one process is executed at the time
type List struct {
	*sync.Mutex
	tasks   []*Task
	wgTasks map[int]*sync.WaitGroup
	wg      *sync.WaitGroup
	// resources currently used by running tasks
	usedResources *ResourcesSet
	idCounter     int
}

// NewList creates empty task list
func NewList() *List {
	list := &List{
		Mutex:         &sync.Mutex{},
		tasks:         make([]*Task, 0),
		wgTasks:       make(map[int]*sync.WaitGroup),
		wg:            &sync.WaitGroup{},
		usedResources: NewResourcesSet(),
	}
	return list
}

// GetTasks gets complete list of tasks
func (list *List) GetTasks() []Task {
	var tasks []Task
	list.Lock()
	for _, task := range list.tasks {
		tasks = append(tasks, *task)
	}

	list.Unlock()
	return tasks
}

// DeleteTaskByID deletes given task from list. Only finished
// tasks can be deleted.
func (list *List) DeleteTaskByID(ID int) (Task, error) {
	list.Lock()
	defer list.Unlock()

	tasks := list.tasks
	for i, task := range tasks {
		if task.ID == ID {
			if task.State == SUCCEEDED || task.State == FAILED {
				list.tasks = append(tasks[:i], tasks[i+1:]...)
				return *task, nil
			}

			return *task, fmt.Errorf("Task with id %v is still in state=%d", ID, task.State)
		}
	}

	return Task{}, fmt.Errorf("Could not find task with id %v", ID)
}

// GetTaskByID returns task with given id
func (list *List) GetTaskByID(ID int) (Task, error) {
	list.Lock()
	tasks := list.tasks
	list.Unlock()

	for _, task := range tasks {
		if task.ID == ID {
			return *task, nil
		}
	}

	return Task{}, fmt.Errorf("Could not find task with id %v", ID)
}

// GetTaskOutputByID returns standard output of task with given id
func (list *List) GetTaskOutputByID(ID int) (string, error) {
	task, err := list.GetTaskByID(ID)

	if err != nil {
		return "", err
	}

	return task.output.String(), nil
}

// GetTaskDetailByID returns detail of task with given id
func (list *List) GetTaskDetailByID(ID int) (interface{}, error) {
	task, err := list.GetTaskByID(ID)

	if err != nil {
		return nil, err
	}

	detail := task.detail.Load()
	if detail == nil {
		return struct{}{}, nil
	}

	return detail, nil
}

// GetTaskReturnValueByID returns process return value of task with given id
func (list *List) GetTaskReturnValueByID(ID int) (*ProcessReturnValue, error) {
	task, err := list.GetTaskByID(ID)

	if err != nil {
		return nil, err
	}

	return task.processReturnValue, nil
}

// RunTaskInBackground creates task and runs it in background. It won't be run and an error
// returned if there are running tasks which are using needed resources already.
func (list *List) RunTaskInBackground(name string, resources []string, process Process) (Task, *ResourceConflictError) {
	list.Lock()
	defer list.Unlock()

	tasks := list.usedResources.UsedBy(resources)
	if len(tasks) > 0 {
		conflictError := &ResourceConflictError{
			Tasks:   tasks,
			Message: "Needed resources are used by other tasks.",
		}
		return Task{}, conflictError
	}

	list.idCounter++
	wgTask := &sync.WaitGroup{}
	task := NewTask(process, name, list.idCounter)

	list.tasks = append(list.tasks, task)
	list.wgTasks[task.ID] = wgTask
	list.usedResources.MarkInUse(resources, task)

	list.wg.Add(1)
	wgTask.Add(1)

	go func() {

		list.Lock()
		{
			task.State = RUNNING
		}
		list.Unlock()

		retValue, err := process(aptly.Progress(task.output), task.detail)

		list.Lock()
		{
			task.processReturnValue = retValue
			if err != nil {
				task.output.Printf("Task failed with error: %v", err)
				task.State = FAILED
			} else {
				task.output.Print("Task succeeded")
				task.State = SUCCEEDED
			}

			list.usedResources.Free(resources)

			wgTask.Done()
			list.wg.Done()
		}
		list.Unlock()
	}()

	return *task, nil
}

// Clear removes finished tasks from list
func (list *List) Clear() {
	list.Lock()

	var tasks []*Task
	for _, task := range list.tasks {
		if task.State == IDLE || task.State == RUNNING {
			tasks = append(tasks, task)
		}
	}
	list.tasks = tasks

	list.Unlock()
}

// Wait waits till all tasks are processed
func (list *List) Wait() {
	list.wg.Wait()
}

// WaitForTaskByID waits for task with given id to be processed
func (list *List) WaitForTaskByID(ID int) (Task, error) {
	list.Lock()
	wgTask, ok := list.wgTasks[ID]
	list.Unlock()
	if !ok {
		return Task{}, fmt.Errorf("Could not find task with id %v", ID)
	}

	wgTask.Wait()
	return list.GetTaskByID(ID)
}
