package tree

import "context"

// WorkHandler implements work for a task
type WorkHandler interface {
	Work(ctx context.Context, work *Work) error
}

// WorkHandlerFunc converts a function to a handler
type WorkHandlerFunc func(ctx context.Context, work *Work) error

// Work executes the function as work for a task
func (w WorkHandlerFunc) Work(ctx context.Context, work *Work) error {
	return w(ctx, work)
}
