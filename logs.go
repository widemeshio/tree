package tree

import "log"

// Logger basic logger instance for task internals
type Logger interface {
	Debugf(msg string, args ...interface{})
	Named(name string) Logger
}

// NewDevelopmentLogger creates a new instance of development Logger
func NewDevelopmentLogger() Logger {
	return &devLogger{
		name: "",
	}
}

type devLogger struct {
	name string
}

func (logger *devLogger) Debugf(msg string, args ...interface{}) {
	if logger.name != "" {
		msg = logger.name + ": " + msg
	}
	log.Printf(msg, args...)
}

func (logger *devLogger) Named(name string) Logger {
	if logger.name != "" {
		name = logger.name + "." + name
	}
	return &devLogger{
		name: name,
	}
}

var defaultLogger = &nopLogger{}

type nopLogger struct{}

func (logger *nopLogger) Debugf(msg string, args ...interface{}) {}

func (logger *nopLogger) Named(name string) Logger {
	return defaultLogger
}
