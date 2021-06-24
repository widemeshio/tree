package tree

import "context"

type Work struct {
	ctx   context.Context
	owner *Task
	Logger
}

// Spawn returns a child task running with the given name and handler.
func (w *Work) Spawn(name string, handler TaskHandler) *Task {
	newTask := NewTask(name, w.owner.logger)
	newTask.Work = handler.Work
	w.owner.startSub(w.ctx, newTask)
	return newTask
}

// AwaitAnySub blocks until any child terminates
func (w *Work) AwaitAnySub() (*Task, error) {
	return w.owner.awaitAnySub(w.ctx)
}

// TerminateEarly requests early termination of the current work causing the context of the current work to cancelled.
// Regularly to terminate the current work early, just use a regular go return from the work handler.
func (w *Work) TerminateEarly() {
	w.owner.Terminate()
}

type TaskHandler interface {
	Work(ctx context.Context, work *Work) error
}
