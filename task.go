package tree

import (
	"context"
	"sync"
	"time"
)

// DefaultTerminationDeadline default termination deadline for all tasks
const DefaultTerminationDeadline = 5 * time.Second

// Task controls an asynchronous unit of work
type Task struct {
	name                       string
	options                    Options
	logger                     Logger
	isTerminated               bool
	terminationSignalChan      chan struct{}
	terminationSignalChanMutex sync.Mutex
	terminatedChan             chan struct{}
	workErr                    error
	subs                       []*Task
	subsMutex                  sync.Mutex
	TerminationDeadline        time.Duration
	Work                       WorkHandler
}

// NewTask creates a new instance of task with options
func NewTask(opts ...Option) *Task {
	options := NewOptions(opts...)
	name := options.Name
	return &Task{
		name:                  name,
		options:               options,
		logger:                options.GetLogger().Named(name),
		terminationSignalChan: make(chan struct{}),
		terminatedChan:        make(chan struct{}),
		TerminationDeadline:   DefaultTerminationDeadline,
		Work:                  options.Work,
	}
}

func (task *Task) work(ctx context.Context, cancelDeadlineStartSignal <-chan struct{}) {
	handler := task.Work
	if handler == nil {
		return
	}
	exit := make(chan error, 2)
	go func() {
		work := &Work{
			ctx:   ctx,
			owner: task,
		}
		exit <- handler.Work(ctx, work)
	}()
	terminationDeadline := task.TerminationDeadline
	go func() {
		<-cancelDeadlineStartSignal
		time.Sleep(terminationDeadline)
		exit <- &ErrTerminationDeadlineExceeded{
			TaskName: task.Name(),
			Timeout:  terminationDeadline,
			Message:  "termination deadline exceeded",
		}
	}()
	task.workErr = <-exit
}

func (task *Task) Run(ctx context.Context) error {
	logger := task.logger
	logger.Debugf("running")
	cancelDeadlineStartSignal := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(ctx)
	watchCancel := func() {
		logger.Debugf("watching cancel")
		<-task.terminationSignalChan
		logger.Debugf("cancelling")
		cancel()
		logger.Debugf("cancelled")
		cancelDeadlineStartSignal <- struct{}{}
	}
	go watchCancel()

	logger.Debugf("work starting")
	task.work(ctx, cancelDeadlineStartSignal)
	logger.Debugf("work completed, now requesting termination")
	task.Terminate()
	task.terminateChildren(ctx)

	logger.Debugf("children termination complete")
	err := task.workErr
	if err != nil {
		logger.Debugf("done closed with error, %s", err.Error())
	} else {
		logger.Debugf("done closed success")
	}
	close(task.terminatedChan)
	return err
}

// Terminate requests termination of the task. Safe to be called from any goroutine.
func (task *Task) Terminate() {
	logger := task.logger
	task.terminationSignalChanMutex.Lock()
	defer task.terminationSignalChanMutex.Unlock()
	if task.isTerminated {
		logger.Debugf("already terminated")
		return
	}
	logger.Debugf("terminate closing")
	task.isTerminated = true
	close(task.terminationSignalChan)
	logger.Debugf("terminate closed")
}

// Terminated returns a chan you can watch when the task has been terminated, meaning the task has completed the full run cycle.
// Child tasks have been given a chance to terminate at this point via context cancellation.
func (task *Task) Terminated() <-chan struct{} {
	return task.terminatedChan
}

// Err returns the execution error of the task.
func (task *Task) Err() error {
	return task.workErr
}

// Name returns the name of the task
func (task *Task) Name() string {
	return task.name
}

func (task *Task) startSub(ctx context.Context, sub *Task) {
	task.subsMutex.Lock()
	defer task.subsMutex.Unlock()
	if task.isTerminated {
		panic("unable to start spawn when task already completed")
	}
	task.subs = append(task.subs, sub)
	go sub.Run(ctx)
}

func (task *Task) awaitAnySub(ctx context.Context) (*Task, error) {
	task.subsMutex.Lock()
	defer task.subsMutex.Unlock()
	subs := task.subs
	if len(subs) == 0 {
		return nil, nil
	}
	completed := make(chan *Task, len(subs))
	for _, sub := range subs {
		go func(sub *Task) {
			<-sub.Terminated()
			completed <- sub
		}(sub)
	}
	completeSub := <-completed
	return completeSub, completeSub.Err()
}

func (task *Task) terminateChildren(ctx context.Context) {
	logger := task.logger
	logger.Debugf("terminating children")
	task.subsMutex.Lock()
	defer task.subsMutex.Unlock()
	subs := task.subs
	for _, sub := range subs {
		logger.Debugf("terminating %s", sub.Name())
		sub.Terminate()
	}
}

func (task *Task) childOptions() Options {
	opts := task.options.Apply(clearNonInheritables())
	return *opts
}
