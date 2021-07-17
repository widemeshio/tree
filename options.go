package tree

// Options contains task options
type Options struct {
	// name of the task
	Name string

	// logger for internal task logs, defaults to nop logger
	Logger Logger

	// work to be performed by the task
	Work WorkHandler
}

// NewOptions returns new Options with the given options
func NewOptions(opts ...Option) Options {
	options := Options{}
	options.Apply(opts...)
	return options
}

// Apply applies the given options
func (opts *Options) Apply(builders ...Option) *Options {
	for _, builder := range builders {
		builder.apply(opts)
	}
	return opts
}

// Copy returns a copy of the options
func (opts *Options) Copy() Options {
	return *opts
}

// GetLogger returns a logger instance or default
func (opts *Options) GetLogger() Logger {
	if v := opts.Logger; v != nil {
		return v
	}
	return defaultLogger
}

// optionFunc is a helper func that satisfies the Option interface
type optionFunc func(options *Options)

func (f optionFunc) apply(options *Options) {
	f(options)
}

// Option implements an option that can be applied to Options
type Option interface {
	apply(options *Options)
}

// WithName returns an Option that applies a Name to Options
func WithName(name string) Option {
	return optionFunc(func(options *Options) {
		options.Name = name
	})
}

// WithLogger returns an Option that applies a Logger to Options
func WithLogger(logger Logger) Option {
	return optionFunc(func(options *Options) {
		options.Logger = logger
	})
}

// WithWork returns an Option that applies a WorkHandler to Options
func WithWork(handler WorkHandler) Option {
	return optionFunc(func(options *Options) {
		options.Work = handler
	})
}

// WithWorkFunc returns an Option that applies a WorkHandler to Options
func WithWorkFunc(f WorkHandlerFunc) Option {
	return WithWork(f)
}

// WithOptions returns an Option that replaces the given options to Options
func WithOptions(newOptions Options) Option {
	return optionFunc(func(options *Options) {
		*options = newOptions
	})
}

// clearNonInheritables returns an Option that resets the non-inheritable options
func clearNonInheritables() Option {
	return optionFunc(func(options *Options) {
		options.Name = ""
		options.Work = nil
	})
}
