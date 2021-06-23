package tree

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAwaitAnySubWithError(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTask("any with sub error", logger)
	numbers := make(chan int)
	generator := newGeneratorCrashing(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask *Task
	program.Work = func(ctx context.Context, work *Work) error {
		work.StartSub("generatorAnySub", generator)
		printerTask = work.StartSub("printAnySub", printer)

		_, err := work.AwaitAnySub()
		if err != nil {
			return err
		}
		return nil
	}
	ctx := context.Background()
	err := program.Run(ctx)
	require.Equal(t, errGeneratorCrash, err)
	<-printerTask.Done()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
}

func TestAwaitAnySubSuccess(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTask("any-sub-success", logger)
	numbers := make(chan int)
	generator := newGeneratorOk(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask, generatorTask *Task
	program.Work = func(ctx context.Context, work *Work) error {
		generatorTask = work.StartSub("generatorAnySub", generator)
		printerTask = work.StartSub("printAnySub", printer)

		_, err := work.AwaitAnySub()
		if err != nil {
			return err
		}
		return nil
	}
	ctx := context.Background()
	err := program.Run(ctx)
	require.Nil(t, err, "no error should be returned")
	<-printerTask.Done()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
	<-generatorTask.Done()
	require.Equal(t, falseInt32, generator.cancelled, "generator should not be cancelled, it should have completed normally")
	require.Equal(t, trueInt32, generator.completed, "generator should be done normally")
}

func TestCascadeCancel(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTask("cascade cancel", logger)
	numbers := make(chan int)
	generator := newGeneratorOk(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask, generatorTask *Task
	program.Work = func(ctx context.Context, work *Work) error {
		generatorTask = work.StartSub("generatorAnySub", generator)
		printerTask = work.StartSub("printAnySub", printer)

		work.Debugf("waiting before completing")
		time.Sleep(time.Second)
		work.Debugf("completing")
		return nil
	}
	ctx := context.Background()
	err := program.Run(ctx)
	require.Nil(t, err, "no error should be returned")
	<-printerTask.Done()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
	<-generatorTask.Done()
	require.Equal(t, trueInt32, generator.cancelled, "generator should have been cancelled")
	require.Equal(t, falseInt32, generator.completed, "generator should have not completed normally")
}

func TestTerminateContextDone(t *testing.T) {
	logger := NewDevelopmentLogger()
	program := NewTask("terminate-context-done", logger)
	numbers := make(chan int)
	generator := newGeneratorOk(numbers)
	printer := newPrinterAnySub(numbers)
	var printerTask, generatorTask *Task
	cancelSuccessErr := fmt.Errorf("cancel err")
	program.Work = func(ctx context.Context, work *Work) error {
		generatorTask = work.StartSub("generator", generator)
		printerTask = work.StartSub("print", printer)

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
	}
	ctx := context.Background()
	go func() {
		time.Sleep(time.Second)
		program.Terminate()
	}()
	err := program.Run(ctx)
	require.Equal(t, cancelSuccessErr, err, "task should have been cancelled and specific error returned")
	logger.Debugf("waiting for printer to complete")
	<-printerTask.Done()
	require.Equal(t, trueInt32, printer.cancelled, "printer should be cancelled eventually")
	logger.Debugf("waiting for generator")
	<-generatorTask.Done()
	require.Equal(t, trueInt32, generator.cancelled, "generator should have been cancelled")
	require.Equal(t, falseInt32, generator.completed, "generator should have not completed normally")
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

func (gen *generatorCrashing) Work(ctx context.Context, work *Work) error {
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

func (gen *generatorOk) Work(ctx context.Context, work *Work) error {
	i := 0
	timer := time.NewTicker(time.Second)
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

func (pri *printAnySub) Work(ctx context.Context, work *Work) error {
	done := ctx.Done()

	for {
		select {
		case i := <-pri.numbers:
			work.Debugf("printing number %v", i)
		case <-done:
			work.Debugf("printing done")
			atomic.StoreInt32(&pri.cancelled, trueInt32)
			return nil
		}
	}
}

var trueInt32 = int32(1)
var falseInt32 = int32(0)
