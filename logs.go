package tree

import "log"

type Logger interface {
	Debugf(msg string, args ...interface{})
	Named(name string) Logger
}

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
