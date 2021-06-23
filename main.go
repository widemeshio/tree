package main

import "context"

func main() {
	logger := NewDevelopmentLogger()
	program := NewProgram("demo-tasks", logger)
	program.Work = func(ctx context.Context, work *Work) error {
		numbers := make(chan int)
		work.StartSub("generator", &generator{numbers})
		work.StartSub("printer", &printer{numbers})

		return work.WaitSubError()
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
		gen.numbers <- i
		work.Debugf("generated number %v", i)
	}
}

type printer struct {
	numbers chan int
}

func (pri *printer) Work(ctx context.Context, work *Work) error {
	for i := range pri.numbers {
		work.Debugf("printing number %v", i)
	}
	return nil
}
