package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	logger := NewDevelopmentLogger()
	program := NewProgram("demo-tasks", logger)
	program.Work = func(ctx context.Context, work *Work) error {
		numbers := make(chan int)
		work.StartSub("generator", &generator{numbers})
		work.StartSub("printer", &printer{numbers})

		return work.WaitSubExit()
	}
	program.Run()
}

type generator struct {
	numbers chan int
}

func (gen *generator) Work(ctx context.Context, work *Work) error {
	i := 0
	for {
		i++
		work.Debugf("generating number %v", i)
		if i == 10 {
			return fmt.Errorf("generator crash at 10")
		}
		gen.numbers <- i
		work.Debugf("generated number %v", i)
		time.Sleep(time.Second)
	}
}

type printer struct {
	numbers chan int
}

func (pri *printer) Work(ctx context.Context, work *Work) error {
	done := ctx.Done()

	for {
		select {
		case i := <-pri.numbers:
			work.Debugf("printing number %v", i)
		case <-done:
			work.Debugf("printing done")
		}
	}
}
