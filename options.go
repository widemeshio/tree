package tree

// Options contains task options
type Options struct {
	// logger for internal task logs, defaults to nop logger
	Logger Logger
}

// GetLogger returns a logger instance or default
func (opts Options) GetLogger() Logger {
	if v := opts.Logger; v != nil {
		return v
	}
	return defaultLogger
}
