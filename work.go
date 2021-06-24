package tree

import "context"

type Work struct {
	ctx   context.Context
	owner *Task
	Logger
}

// StartSub registers a child task with the given name and handler
func (w *Work) StartSub(name string, handler TaskHandler) *Task {
	newTask := NewTask(name, w.owner.logger)
	newTask.Work = handler.Work
	w.owner.startSub(w.ctx, newTask)
	return newTask
}

// AwaitAnySub blocks until any child terminates
func (w *Work) AwaitAnySub() (*Task, error) {
	return w.owner.awaitAnySub(w.ctx)
}

// TerminateEarly requests early termination of the current work.
// This causes the context of the work to cancelled.
func (w *Work) TerminateEarly() {
	w.owner.Terminate()
}

type TaskHandler interface {
	Work(ctx context.Context, work *Work) error
}
