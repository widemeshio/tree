package tree

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitChildWithError(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTaskWithOptions("any with sub error", taskOptionsWithLogger(logger))
	numbers := make(chan int)
	generator := newGeneratorCrashing(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask *Task
	program.Work = WorkHandlerFunc(func(ctx context.Context, work *Work) error {
		work.Spawn("generatorAnySub", newTestWorkHandler(generator))
		printerTask = work.Spawn("printAnySub", newTestWorkHandler(printer))

		_, err := work.WaitChild()
		if err != nil {
			return err
		}
		return nil
	})
	ctx := context.Background()
	err := program.Run(ctx)
	require.Equal(t, errGeneratorCrash, err)
	<-printerTask.Terminated()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
}

func TestWaitChildSuccess(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTaskWithOptions("any-sub-success", taskOptionsWithLogger(logger))
	numbers := make(chan int)
	generator := newGeneratorOk(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask, generatorTask *Task
	program.Work = WorkHandlerFunc(func(ctx context.Context, work *Work) error {
		generatorTask = work.Spawn("generatorAnySub", newTestWorkHandler(generator))
		printerTask = work.Spawn("printAnySub", newTestWorkHandler(printer))

		_, err := work.WaitChild()
		if err != nil {
			return err
		}
		return nil
	})
	ctx := context.Background()
	err := program.Run(ctx)
	require.Nil(t, err, "no error should be returned")
	<-printerTask.Terminated()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
	<-generatorTask.Terminated()
	require.Equal(t, falseInt32, generator.cancelled, "generator should not be cancelled, it should have completed normally")
	require.Equal(t, trueInt32, generator.completed, "generator should be done normally")
}

func TestCascadeCancel(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTaskWithOptions("cascade cancel", taskOptionsWithLogger(logger))
	program.TerminationDeadline = 15 * time.Second
	numbers := make(chan int, 1000) // avoid deadlocks in this specific test
	generator := newGeneratorOk(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask, generatorTask *Task
	program.Work = WorkHandlerFunc(newTestWork(func(ctx context.Context, work *testWork) error {
		generatorTask = work.Spawn("generatorAnySub", newTestWorkHandler(generator))
		generatorTask.TerminationDeadline = 5 * time.Second
		printerTask = work.Spawn("printAnySub", newTestWorkHandler(printer))
		printerTask.TerminationDeadline = 5 * time.Second

		work.Debugf("waiting before completing")
		time.Sleep(time.Second)
		work.Debugf("completing")
		return nil
	}))
	ctx := context.Background()
	err := program.Run(ctx)
	require.Nil(t, err, "no error should be returned")
	<-printerTask.Terminated()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
	<-generatorTask.Terminated()
	require.Equal(t, trueInt32, generator.cancelled, "generator should have been cancelled")
	require.Equal(t, falseInt32, generator.completed, "generator should have not completed normally")
}

func TestTerminateContextDone(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTaskWithOptions("terminate-context-done", taskOptionsWithLogger(logger))
	numbers := make(chan int, 1000) // avoid deadlocks in this specific test
	generator := newGeneratorOk(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask, generatorTask *Task
	cancelSuccessErr := fmt.Errorf("cancel err")
	program.Work = WorkHandlerFunc(newTestWork(func(ctx context.Context, work *testWork) error {
		generatorTask = work.Spawn("generator", newTestWorkHandler(generator))
		printerTask = work.Spawn("print", newTestWorkHandler(printer))

		work.Debugf("waiting before completing")
		tick := time.NewTimer(20 * time.Second)
		select {
		case <-ctx.Done():
			return cancelSuccessErr
		case <-tick.C:
			work.Debugf("timeout was completed without ctx done, this should not happen")
			break
		}
		return nil
	}))
	ctx := context.Background()
	go func() {
		time.Sleep(time.Second)
		program.Terminate()
	}()
	err := program.Run(ctx)
	require.Equal(t, cancelSuccessErr, err, "task should have been cancelled and specific error returned")
	logger.Debugf("waiting for printer to complete")
	<-printerTask.Terminated()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
	logger.Debugf("waiting for generator")
	<-generatorTask.Terminated()
	require.Equal(t, trueInt32, generator.cancelled, "generator should have been cancelled")
	require.Equal(t, falseInt32, generator.completed, "generator should have not completed normally")
}

func TestTerminateDeadline(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTaskWithOptions("terminate-context-done", taskOptionsWithLogger(logger))
	program.TerminationDeadline = 7 * time.Second
	numbers := make(chan int)
	generator := newGeneratorOk(numbers)
	var generatorTask *Task
	program.Work = WorkHandlerFunc(func(ctx context.Context, work *Work) error {
		generatorTask = work.Spawn("generator", newTestWorkHandler(generator))
		generatorTask.TerminationDeadline = 3 * time.Second
		<-generatorTask.Terminated()
		return generatorTask.Err()
	})
	ctx := context.Background()
	go func() {
		time.Sleep(time.Second)
		program.Terminate()
	}()
	err := program.Run(ctx)
	var target *ErrTerminationDeadlineExceeded
	require.ErrorAs(t, err, &target)
	require.ErrorAs(t, program.Err(), &target)
	logger.Debugf("waiting for generator")
	<-generatorTask.Terminated()
	require.Equal(t, falseInt32, generator.cancelled, "generator goroutine never had the chance to execute ctx done")
	require.Equal(t, falseInt32, generator.completed, "generator should have not completed normally")
}

func TestWaitChildNoChildren(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTaskWithOptions("wait-no-children", taskOptionsWithLogger(logger))
	program.TerminationDeadline = 7 * time.Second
	program.Work = WorkHandlerFunc(func(ctx context.Context, work *Work) error {
		_, err := work.WaitChild()
		if err != nil {
			return err
		}
		return nil
	})
	ctx := context.Background()
	err := program.Run(ctx)
	require.Nil(t, err)
}

type generatorCrashing struct {
	numbers chan int
}

func newGeneratorCrashing(
	numbers chan int,
) *generatorCrashing {
	return &generatorCrashing{
		numbers: numbers,
	}
}

func (gen *generatorCrashing) Work(ctx context.Context, work *testWork) error {
	i := 0
	for {
		i++
		work.Debugf("generating number %v", i)
		if i == 5 {
			return errGeneratorCrash
		}
		gen.numbers <- i
		work.Debugf("generated number %v", i)
		time.Sleep(time.Second)
	}
}

var errGeneratorCrash = fmt.Errorf("generator intentional crash")

type generatorOk struct {
	numbers   chan int
	cancelled int32
	completed int32
}

func newGeneratorOk(
	numbers chan int,
) *generatorOk {
	return &generatorOk{
		numbers: numbers,
	}
}

func (gen *generatorOk) Work(ctx context.Context, work *testWork) error {
	i := 0
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	working := true
	for working {
		select {
		case <-ctx.Done():
			work.Debugf("ctx done")
			atomic.StoreInt32(&gen.cancelled, trueInt32)
			return nil
		case <-timer.C:
			i++
			work.Debugf("generating number %v", i)
			if i == 5 {
				working = false
				break
			}
			gen.numbers <- i
			work.Debugf("generated number %v", i)
		}
	}
	work.Debugf("completed")
	atomic.StoreInt32(&gen.completed, trueInt32)
	return nil
}

type printAnySub struct {
	numbers   chan int
	cancelled int32
}

func newPrinterAnySub(numbers chan int) *printAnySub {
	return &printAnySub{
		numbers: numbers,
	}
}

func (pri *printAnySub) Work(ctx context.Context, work *testWork) error {

	for {
		select {
		case i := <-pri.numbers:
			work.Debugf("printing number %v", i)
		case <-ctx.Done():
			work.Debugf("printing cancelled")
			atomic.StoreInt32(&pri.cancelled, trueInt32)
			return nil
		}
	}
}

var trueInt32 = int32(1)
var falseInt32 = int32(0)

type Program struct {
	*Task
}

func NewProgram(name string, logger Logger) *Program {
	return &Program{
		Task: NewTaskWithOptions(name, taskOptionsWithLogger(logger)),
	}
}

func (mainTask *Program) Run() {
	ctx := context.Background()
	err := mainTask.Task.Run(ctx)
	if err != nil {
		log.Fatalf("main task failed, %s", err.Error())
	}
}

type testWork struct {
	*Work
	Logger
}

func newTestWork(original func(ctx context.Context, work *testWork) error) func(ctx context.Context, work *Work) error {
	return func(ctx context.Context, work *Work) error {
		return original(ctx, &testWork{
			Work:   work,
			Logger: work.Task().logger,
		})
	}
}

func newTestWorkHandler(handler interface {
	Work(ctx context.Context, work *testWork) error
}) WorkHandler {
	return &testWorkHandler{
		work: newTestWork(handler.Work),
	}
}

type testWorkHandler struct {
	work func(ctx context.Context, work *Work) error
}

func (w *testWorkHandler) Work(ctx context.Context, work *Work) error {
	return w.work(ctx, work)
}

func taskOptionsWithLogger(logger Logger) Options {
	return Options{
		Logger: logger,
	}
}
