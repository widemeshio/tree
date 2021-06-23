package tree

import (
	"context"
	"log"
)

type Program struct {
	*Task
}

func NewProgram(name string, logger Logger) *Program {
	return &Program{
		Task: NewTask(name, logger),
	}
}

func (mainTask *Program) Run() {
	ctx := context.Background()
	err := mainTask.Task.Run(ctx)
	if err != nil {
		log.Fatalf("main task failed, %s", err.Error())
	}
}
