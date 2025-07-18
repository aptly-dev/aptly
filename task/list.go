package task

import (
	"fmt"
	"sync"
	"time"

	"github.com/aptly-dev/aptly/aptly"
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

	queue        chan *Task
	pendingQueue []*Task  // Tasks waiting to be queued
	queueWg      *sync.WaitGroup
	queueDone    chan bool
	stopped      bool
}

// NewList creates empty task list
func NewList() *List {
	list := &List{
		Mutex:         &sync.Mutex{},
		tasks:         make([]*Task, 0),
		wgTasks:       make(map[int]*sync.WaitGroup),
		wg:            &sync.WaitGroup{},
		usedResources: NewResourcesSet(),
		queue:         make(chan *Task, 1), // Small buffer for efficiency
		pendingQueue:  make([]*Task, 0),
		queueWg:       &sync.WaitGroup{},
		queueDone:     make(chan bool),
		stopped:       false,
	}
	list.queueWg.Add(1)
	go list.consumer()
	return list
}

// tryQueueTask attempts to queue a task without blocking
func (list *List) tryQueueTask(task *Task) {
	select {
	case list.queue <- task:
		// Successfully queued
	default:
		// Channel is full, add to pending queue
		list.pendingQueue = append(list.pendingQueue, task)
	}
}

// processPendingQueue tries to move tasks from pending queue to main queue
func (list *List) processPendingQueue() {
	if len(list.pendingQueue) == 0 {
		return
	}
	
	// Try to queue pending tasks
	remaining := make([]*Task, 0)
	for _, task := range list.pendingQueue {
		select {
		case list.queue <- task:
			// Successfully queued
		default:
			// Still can't queue, keep in pending
			remaining = append(remaining, task)
		}
	}
	list.pendingQueue = remaining
}

// consumer is processing the queue
func (list *List) consumer() {
	defer list.queueWg.Done()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case task := <-list.queue:
			list.Lock()
			{
				task.State = RUNNING
			}
			list.Unlock()

			go func(t *Task) {
				// Ensure Done() is always called, even if panic occurs
				defer func() {
					list.Lock()
					defer list.Unlock()

					t.wgTask.Done()
					list.wg.Done()
					list.usedResources.Free(t.resources)
					
					// Now that resources are freed, try to queue idle tasks
					for _, task := range list.tasks {
						if task.State == IDLE {
							// check resources
							blockingTasks := list.usedResources.UsedBy(task.resources)
							if len(blockingTasks) == 0 {
								list.usedResources.MarkInUse(task.resources, task)
								list.tryQueueTask(task)
								break
							}
						}
					}
					
					// Process any pending tasks
					list.processPendingQueue()
				}()

				retValue, err := t.process(aptly.Progress(t.output), t.detail)

				list.Lock()
				{
					t.processReturnValue = retValue
					t.err = err
					if err != nil {
						t.output.Printf("Task failed with error: %v", err)
						t.State = FAILED
					} else {
						t.output.Print("Task succeeded")
						t.State = SUCCEEDED
					}
				}
				list.Unlock()
			}(task)

		case <-ticker.C:
			// Periodically check pending queue
			list.Lock()
			list.processPendingQueue()
			list.Unlock()
			
		case <-list.queueDone:
			return
		}
	}
}

// Stop signals the consumer to stop processing tasks and waits for it to finish
func (list *List) Stop() {
	list.Lock()
	if list.stopped {
		list.Unlock()
		return
	}
	list.stopped = true
	list.Unlock()
	
	close(list.queueDone)
	list.queueWg.Wait()
}

// GetTasks gets complete list of tasks
func (list *List) GetTasks() []Task {
	tasks := []Task{}
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

			return *task, fmt.Errorf("task with id %v is still in state=%d", ID, task.State)
		}
	}

	return Task{}, fmt.Errorf("could not find task with id %v", ID)
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

	return Task{}, fmt.Errorf("could not find task with id %v", ID)
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

// RunTaskInBackground creates task and runs it in background. This will block until the necessary resources
// become available.
func (list *List) RunTaskInBackground(name string, resources []string, process Process) (Task, *ResourceConflictError) {
	list.Lock()
	defer list.Unlock()

	list.idCounter++
	wgTask := &sync.WaitGroup{}
	task := NewTask(process, name, list.idCounter, resources, wgTask)

	list.tasks = append(list.tasks, task)
	list.wgTasks[task.ID] = wgTask

	list.wg.Add(1)
	task.wgTask.Add(1)

	// add task to queue for processing if resources are available
	// if not, task will be queued by the consumer once resources are available
	tasks := list.usedResources.UsedBy(resources)
	if len(tasks) == 0 {
		list.usedResources.MarkInUse(task.resources, task)
		list.tryQueueTask(task)
	}

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
		return Task{}, fmt.Errorf("could not find task with id %v", ID)
	}

	wgTask.Wait()
	return list.GetTaskByID(ID)
}

// GetTaskErrorByID returns the Task error for a given id
func (list *List) GetTaskErrorByID(ID int) (error, error) {
	task, err := list.GetTaskByID(ID)

	if err != nil {
		return nil, err
	}

	return task.err, nil
}
