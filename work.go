package tree

import "context"

type Work struct {
	ctx   context.Context
	owner *Task
}

// Spawn returns a child task running with the given work
func (w *Work) Spawn(handler WorkHandler, opts ...Option) *Task {
	options := w.owner.childOptions()
	options.Apply(WithWork(handler))
	options.Apply(opts...)
	newTask := NewTask(WithOptions(options))
	return w.Attach(newTask)
}

// Attach adds a task as a child of the executing task
func (w *Work) Attach(newTask *Task) *Task {
	w.owner.startSub(w.ctx, newTask)
	return newTask
}

// WaitChild blocks until a child terminates
func (w *Work) WaitChild() (*Task, error) {
	return w.owner.awaitAnySub(w.ctx)
}

// TerminateEarly requests early termination of the current work causing the context of the current work to cancelled.
// Regularly to terminate the current work early, just use a regular go return from the work handler.
func (w *Work) TerminateEarly() {
	w.owner.Terminate()
}

// Task returns the task executing the work
func (w *Work) Task() *Task {
	return w.owner
}
