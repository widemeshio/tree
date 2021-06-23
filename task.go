package tree

import (
	"context"
	"sync"
)

type Task struct {
	name          string
	logger        Logger
	isDone        bool
	doneChan      chan struct{}
	doneChanMutex sync.Mutex
	doneErr       error
	Work          func(ctx context.Context, work *Work) error
	subs          []*Task
	subsMutex     sync.Mutex
}

func NewTask(name string, logger Logger) *Task {
	return &Task{
		name:     name,
		logger:   logger.Named(name),
		doneChan: make(chan struct{}),
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
	task.doneErr = <-exit
}

func (task *Task) Run(ctx context.Context) error {
	logger := task.logger
	logger.Debugf("running")
	ctx, cancel := context.WithCancel(ctx)
	watchCancel := func() {
		logger.Debugf("watching cancel")
		<-task.doneChan
		logger.Debugf("cancelling")
		cancel()
		logger.Debugf("cancelled")
	}
	go watchCancel()

	task.work(ctx)
	task.Terminate()
	task.terminateChildren(ctx)

	err := task.doneErr
	if err != nil {
		logger.Debugf("done closed with error, %s", err.Error())
	} else {
		logger.Debugf("done closed success")
	}
	return err
}

func (task *Task) Terminate() {
	logger := task.logger
	task.doneChanMutex.Lock()
	defer task.doneChanMutex.Unlock()
	if task.isDone {
		logger.Debugf("already terminated")
		return
	}
	logger.Debugf("terminate closing")
	task.isDone = true
	close(task.doneChan)
	logger.Debugf("terminate closed")
}

func (task *Task) Done() chan struct{} {
	return task.doneChan
}
func (task *Task) Err() error {
	return task.doneErr
}

func (task *Task) Name() string {
	return task.name
}

func (task *Task) startSub(ctx context.Context, sub *Task) {
	task.subsMutex.Lock()
	defer task.subsMutex.Unlock()
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
			<-sub.Done()
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
