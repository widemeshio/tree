package tree

import "time"

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
