package tree

import (
	"time"
)

// ErrTerminationDeadlineExceeded
type ErrTerminationDeadlineExceeded struct {
	Timeout  time.Duration
	TaskName string
	Message  string
}

// Error returns error message
func (err *ErrTerminationDeadlineExceeded) Error() string {
	return err.Message
}

// Is indicates whether the given target is of type
func (err *ErrTerminationDeadlineExceeded) Is(target error) bool {
	_, matches := target.(*ErrTerminationDeadlineExceeded)
	return matches
}

// ErrTaskAlreadyUsed
type ErrTaskAlreadyUsed struct {
	TaskName string
	Message  string
}

// Error returns error message
func (err *ErrTaskAlreadyUsed) Error() string {
	return err.Message
}

// Is indicates whether the given target is of type
func (err *ErrTaskAlreadyUsed) Is(target error) bool {
	_, matches := target.(*ErrTaskAlreadyUsed)
	return matches
}
