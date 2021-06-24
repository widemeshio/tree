package tree

import (
	"context"
	"sync"
)

type Task struct {
	name                       string
	logger                     Logger
	isTerminated               bool
	terminationSignalChan      chan struct{}
	terminationSignalChanMutex sync.Mutex
	terminatedChan             chan struct{}
	workErr                    error
	Work                       func(ctx context.Context, work *Work) error
	subs                       []*Task
	subsMutex                  sync.Mutex
}

func NewTask(name string, logger Logger) *Task {
	return &Task{
		name:                  name,
		logger:                logger.Named(name),
		terminationSignalChan: make(chan struct{}),
		terminatedChan:        make(chan struct{}),
	}
}

func (task *Task) work(ctx context.Context) {
	logger := task.logger
	handler := task.Work
	if handler == nil {
		return
	}
	exit := make(chan error)
	go func() {
		work := &Work{
			ctx:    ctx,
			owner:  task,
			Logger: logger,
		}
		exit <- handler(ctx, work)
	}()
	task.workErr = <-exit
}

func (task *Task) Run(ctx context.Context) error {
	logger := task.logger
	logger.Debugf("running")
	ctx, cancel := context.WithCancel(ctx)
	watchCancel := func() {
		logger.Debugf("watching cancel")
		<-task.terminationSignalChan
		logger.Debugf("cancelling")
		cancel()
		logger.Debugf("cancelled")
	}
	go watchCancel()

	logger.Debugf("work starting")
	task.work(ctx)
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
func (task *Task) Terminated() chan struct{} {
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
