package main

import (
	"context"
	"sync"
)

type Task struct {
	name      string
	logger    Logger
	doneChan  chan struct{}
	doneErr   error
	Work      func(ctx context.Context, work *Work) error
	subs      []*Task
	subsMutex sync.Mutex
}

func NewTask(name string, logger Logger) *Task {
	return &Task{
		name:     name,
		logger:   logger.Named(name),
		doneChan: make(chan struct{}),
	}
}

func (task *Task) Run(ctx context.Context) error {
	logger := task.logger
	logger.Debugf("running")
	logger.Debugf("done closing")
	if handler := task.Work; handler != nil {
		exit := make(chan error)
		go func() {
			work := &Work{
				owner:  task,
				Logger: logger,
			}
			exit <- handler(ctx, work)
		}()
		task.doneErr = <-exit
	}
	close(task.doneChan)
	if task.doneErr != nil {
		logger.Debugf("done closed with error, %s", task.doneErr.Error())
	} else {
		logger.Debugf("done closed success")

	}
	return task.doneErr
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

func (task *Task) waitSubExit(ctx context.Context) error {
	task.subsMutex.Lock()
	defer task.subsMutex.Unlock()
	if len(task.subs) == 0 {
		return nil
	}
	sub := task.subs[0]
	<-sub.Done()
	return sub.Err()
}
