package tree

import "context"

type Work struct {
	ctx   context.Context
	owner *Task
	Logger
}

// StartSub register a Sub-Task of the current work
func (w *Work) StartSub(name string, handler TaskHandler) *Task {
	newTask := NewTask(name, w.owner.logger)
	newTask.Work = handler.Work
	w.owner.startSub(w.ctx, newTask)
	return newTask
}

func (w *Work) AwaitAnySub() (*Task, error) {
	return w.owner.awaitAnySub(w.ctx)
}

func (w *Work) Terminate() {
	w.owner.Terminate()
}

type TaskHandler interface {
	Work(ctx context.Context, work *Work) error
}
