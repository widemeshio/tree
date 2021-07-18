package tree

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultTerminationDeadline default termination deadline for all tasks
const DefaultTerminationDeadline = 5 * time.Second

// Task controls an asynchronous unit of work
type Task struct {
	name                       string
	options                    Options
	logger                     Logger
	isUsed                     int32
	isTerminated               int32
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

// Start starts the task in its own goroutine
func (task *Task) Start(ctx context.Context) {
	go task.Run(ctx)
}

func (task *Task) validateUse() error {
	if !task.IsUsed() {
		return nil
	}
	return &ErrTaskAlreadyUsed{
		TaskName: task.Name(),
		Message:  "task has already been used",
	}
}

// Run runs the task blocking until completed. A task can only run once, the second time it returns ErrTaskAlreadyUsed.
func (task *Task) Run(ctx context.Context) error {
	logger := task.logger
	if err := task.validateUse(); err != nil {
		return err
	}
	atomic.StoreInt32(&task.isUsed, trueInt32)

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
	if task.IsTerminated() {
		logger.Debugf("already terminated")
		return
	}
	logger.Debugf("terminate closing")
	atomic.StoreInt32(&task.isTerminated, trueInt32)
	close(task.terminationSignalChan)
	logger.Debugf("terminate closed")
}

const trueInt32 = int32(1)

// IsUsed indicates whether the given task has run at some point
func (task *Task) IsUsed() bool {
	return atomic.LoadInt32(&task.isUsed) == trueInt32
}

// IsTerminated returns a value that indicates if the task Terminate function has been called by externals or the task itself after finishing running the work handler. Can be called from any goroutine.
func (task *Task) IsTerminated() bool {
	return atomic.LoadInt32(&task.isTerminated) == trueInt32
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

func (task *Task) attach(ctx context.Context, sub *Task) {
	task.subsMutex.Lock()
	defer task.subsMutex.Unlock()
	task.subs = append(task.subs, sub)
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
